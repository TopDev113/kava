package keeper

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/kava-labs/kava/x/bep3/types"
)

// CreateAtomicSwap creates a new atomic swap.
func (k Keeper) CreateAtomicSwap(ctx sdk.Context, randomNumberHash []byte, timestamp int64, heightSpan uint64,
	sender, recipient sdk.AccAddress, senderOtherChain, recipientOtherChain string,
	amount sdk.Coins, crossChain bool,
) error {
	// Confirm that this is not a duplicate swap
	swapID := types.CalculateSwapID(randomNumberHash, sender, senderOtherChain)
	_, found := k.GetAtomicSwap(ctx, swapID)
	if found {
		return errorsmod.Wrap(types.ErrAtomicSwapAlreadyExists, hex.EncodeToString(swapID))
	}

	// Cannot send coins to a module account
	if k.Maccs[recipient.String()] {
		return errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is a module account", recipient)
	}

	if len(amount) != 1 {
		return fmt.Errorf("amount must contain exactly one coin")
	}
	asset, err := k.GetAsset(ctx, amount[0].Denom)
	if err != nil {
		return err
	}

	err = k.ValidateLiveAsset(ctx, amount[0])
	if err != nil {
		return err
	}

	// Swap amount must be within the specified swap amount limits
	if amount[0].Amount.LT(asset.MinSwapAmount) || amount[0].Amount.GT(asset.MaxSwapAmount) {
		return errorsmod.Wrapf(types.ErrInvalidAmount, "amount %d outside range [%s, %s]", amount[0].Amount, asset.MinSwapAmount, asset.MaxSwapAmount)
	}

	// Unix timestamp must be in range [-15 mins, 30 mins] of the current time
	pastTimestampLimit := ctx.BlockTime().Add(time.Duration(-15) * time.Minute).Unix()
	futureTimestampLimit := ctx.BlockTime().Add(time.Duration(30) * time.Minute).Unix()
	if timestamp < pastTimestampLimit || timestamp >= futureTimestampLimit {
		return errorsmod.Wrap(types.ErrInvalidTimestamp, fmt.Sprintf("block time: %s, timestamp: %s", ctx.BlockTime().String(), time.Unix(timestamp, 0).UTC().String()))
	}

	var direction types.SwapDirection
	if sender.Equals(asset.DeputyAddress) {
		if recipient.Equals(asset.DeputyAddress) {
			return errorsmod.Wrapf(types.ErrInvalidSwapAccount, "deputy cannot be both sender and receiver: %s", asset.DeputyAddress)
		}
		direction = types.SWAP_DIRECTION_INCOMING
	} else {
		if !recipient.Equals(asset.DeputyAddress) {
			return errorsmod.Wrapf(types.ErrInvalidSwapAccount, "deputy must be recipient for outgoing account: %s", recipient)
		}
		direction = types.SWAP_DIRECTION_OUTGOING
	}

	switch direction {
	case types.SWAP_DIRECTION_INCOMING:
		// If recipient's account doesn't exist, register it in state so that the address can send
		// a claim swap tx without needing to be registered in state by receiving a coin transfer.
		recipientAcc := k.accountKeeper.GetAccount(ctx, recipient)
		if recipientAcc == nil {
			newAcc := k.accountKeeper.NewAccountWithAddress(ctx, recipient)
			k.accountKeeper.SetAccount(ctx, newAcc)
		}
		// Incoming swaps have already had their fees collected by the deputy during the relay process.
		err = k.IncrementIncomingAssetSupply(ctx, amount[0])
	case types.SWAP_DIRECTION_OUTGOING:

		// Outgoing swaps must have a height span within the accepted range
		if heightSpan < asset.MinBlockLock || heightSpan > asset.MaxBlockLock {
			return errorsmod.Wrapf(types.ErrInvalidHeightSpan, "height span %d outside range [%d, %d]", heightSpan, asset.MinBlockLock, asset.MaxBlockLock)
		}
		// Amount in outgoing swaps must be able to pay the deputy's fixed fee.
		if amount[0].Amount.LTE(asset.FixedFee.Add(asset.MinSwapAmount)) {
			return errorsmod.Wrap(types.ErrInsufficientAmount, amount[0].String())
		}
		err = k.IncrementOutgoingAssetSupply(ctx, amount[0])
		if err != nil {
			return err
		}
		// Transfer coins to module - only needed for outgoing swaps
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amount)
	default:
		err = fmt.Errorf("invalid swap direction: %s", direction.String())
	}
	if err != nil {
		return err
	}

	// Store the details of the swap
	expireHeight := uint64(ctx.BlockHeight()) + heightSpan
	atomicSwap := types.NewAtomicSwap(amount, randomNumberHash, expireHeight, timestamp, sender,
		recipient, senderOtherChain, recipientOtherChain, 0, types.SWAP_STATUS_OPEN, crossChain, direction)

	// Insert the atomic swap under both keys
	k.SetAtomicSwap(ctx, atomicSwap)
	k.InsertIntoByBlockIndex(ctx, atomicSwap)

	// Emit 'create_atomic_swap' event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCreateAtomicSwap,
			sdk.NewAttribute(types.AttributeKeySender, atomicSwap.Sender.String()),
			sdk.NewAttribute(types.AttributeKeyRecipient, atomicSwap.Recipient.String()),
			sdk.NewAttribute(types.AttributeKeyAtomicSwapID, hex.EncodeToString(atomicSwap.GetSwapID())),
			sdk.NewAttribute(types.AttributeKeyRandomNumberHash, hex.EncodeToString(atomicSwap.RandomNumberHash)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, fmt.Sprintf("%d", atomicSwap.Timestamp)),
			sdk.NewAttribute(types.AttributeKeySenderOtherChain, atomicSwap.SenderOtherChain),
			sdk.NewAttribute(types.AttributeKeyExpireHeight, fmt.Sprintf("%d", atomicSwap.ExpireHeight)),
			sdk.NewAttribute(types.AttributeKeyAmount, atomicSwap.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyDirection, atomicSwap.Direction.String()),
		),
	)

	return nil
}

// ClaimAtomicSwap validates a claim attempt, and if successful, sends the escrowed amount and closes the AtomicSwap.
func (k Keeper) ClaimAtomicSwap(ctx sdk.Context, from sdk.AccAddress, swapID []byte, randomNumber []byte) error {
	atomicSwap, found := k.GetAtomicSwap(ctx, swapID)
	if !found {
		return errorsmod.Wrapf(types.ErrAtomicSwapNotFound, "%s", swapID)
	}

	// Only open atomic swaps can be claimed
	if atomicSwap.Status != types.SWAP_STATUS_OPEN {
		return errorsmod.Wrapf(types.ErrSwapNotClaimable, "status %s", atomicSwap.Status.String())
	}

	//  Calculate hashed secret using submitted number
	hashedSubmittedNumber := types.CalculateRandomHash(randomNumber, atomicSwap.Timestamp)
	hashedSecret := types.CalculateSwapID(hashedSubmittedNumber, atomicSwap.Sender, atomicSwap.SenderOtherChain)

	// Confirm that secret unlocks the atomic swap
	if !bytes.Equal(hashedSecret, atomicSwap.GetSwapID()) {
		return errorsmod.Wrapf(types.ErrInvalidClaimSecret, "the submitted random number is incorrect")
	}

	var err error
	switch atomicSwap.Direction {
	case types.SWAP_DIRECTION_INCOMING:
		err = k.DecrementIncomingAssetSupply(ctx, atomicSwap.Amount[0])
		if err != nil {
			return err
		}
		err = k.IncrementCurrentAssetSupply(ctx, atomicSwap.Amount[0])
		if err != nil {
			return err
		}
		// incoming case - coins should be MINTED, then sent to user
		err = k.bankKeeper.MintCoins(ctx, types.ModuleName, atomicSwap.Amount)
		if err != nil {
			return err
		}
		// Send intended recipient coins
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, atomicSwap.Recipient, atomicSwap.Amount)
		if err != nil {
			return err
		}
	case types.SWAP_DIRECTION_OUTGOING:
		err = k.DecrementOutgoingAssetSupply(ctx, atomicSwap.Amount[0])
		if err != nil {
			return err
		}
		err = k.DecrementCurrentAssetSupply(ctx, atomicSwap.Amount[0])
		if err != nil {
			return err
		}
		// outgoing case  - coins should be burned
		err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, atomicSwap.Amount)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid swap direction: %s", atomicSwap.Direction.String())
	}

	// Complete swap
	atomicSwap.Status = types.SWAP_STATUS_COMPLETED
	atomicSwap.ClosedBlock = ctx.BlockHeight()
	k.SetAtomicSwap(ctx, atomicSwap)

	// Remove from byBlock index and transition to longterm storage
	k.RemoveFromByBlockIndex(ctx, atomicSwap)
	k.InsertIntoLongtermStorage(ctx, atomicSwap)

	// Emit 'claim_atomic_swap' event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeClaimAtomicSwap,
			sdk.NewAttribute(types.AttributeKeyClaimSender, from.String()),
			sdk.NewAttribute(types.AttributeKeyRecipient, atomicSwap.Recipient.String()),
			sdk.NewAttribute(types.AttributeKeyAtomicSwapID, hex.EncodeToString(atomicSwap.GetSwapID())),
			sdk.NewAttribute(types.AttributeKeyRandomNumberHash, hex.EncodeToString(atomicSwap.RandomNumberHash)),
			sdk.NewAttribute(types.AttributeKeyRandomNumber, hex.EncodeToString(randomNumber)),
		),
	)

	return nil
}

// RefundAtomicSwap refunds an AtomicSwap, sending assets to the original sender and closing the AtomicSwap.
func (k Keeper) RefundAtomicSwap(ctx sdk.Context, from sdk.AccAddress, swapID []byte) error {
	atomicSwap, found := k.GetAtomicSwap(ctx, swapID)
	if !found {
		return errorsmod.Wrapf(types.ErrAtomicSwapNotFound, "%s", swapID)
	}
	// Only expired swaps may be refunded
	if atomicSwap.Status != types.SWAP_STATUS_EXPIRED {
		return errorsmod.Wrapf(types.ErrSwapNotRefundable, "status %s", atomicSwap.Status.String())
	}

	var err error
	switch atomicSwap.Direction {
	case types.SWAP_DIRECTION_INCOMING:
		err = k.DecrementIncomingAssetSupply(ctx, atomicSwap.Amount[0])
	case types.SWAP_DIRECTION_OUTGOING:
		err = k.DecrementOutgoingAssetSupply(ctx, atomicSwap.Amount[0])
		if err != nil {
			return err
		}
		// Refund coins to original swap sender for outgoing swaps
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, atomicSwap.Sender, atomicSwap.Amount)
	default:
		err = fmt.Errorf("invalid swap direction: %s", atomicSwap.Direction.String())
	}

	if err != nil {
		return err
	}

	// Complete swap
	atomicSwap.Status = types.SWAP_STATUS_COMPLETED
	atomicSwap.ClosedBlock = ctx.BlockHeight()
	k.SetAtomicSwap(ctx, atomicSwap)

	// Transition to longterm storage
	k.InsertIntoLongtermStorage(ctx, atomicSwap)

	// Emit 'refund_atomic_swap' event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRefundAtomicSwap,
			sdk.NewAttribute(types.AttributeKeyRefundSender, from.String()),
			sdk.NewAttribute(types.AttributeKeySender, atomicSwap.Sender.String()),
			sdk.NewAttribute(types.AttributeKeyAtomicSwapID, hex.EncodeToString(atomicSwap.GetSwapID())),
			sdk.NewAttribute(types.AttributeKeyRandomNumberHash, hex.EncodeToString(atomicSwap.RandomNumberHash)),
		),
	)

	return nil
}

// UpdateExpiredAtomicSwaps finds all AtomicSwaps that are past (or at) their ending times and expires them.
func (k Keeper) UpdateExpiredAtomicSwaps(ctx sdk.Context) {
	var expiredSwapIDs []string
	k.IterateAtomicSwapsByBlock(ctx, uint64(ctx.BlockHeight()), func(id []byte) bool {
		atomicSwap, found := k.GetAtomicSwap(ctx, id)
		if !found {
			// NOTE: shouldn't happen. Continue to next item.
			return false
		}
		// Expire the uncompleted swap and update both indexes
		atomicSwap.Status = types.SWAP_STATUS_EXPIRED
		// Note: claimed swaps have already been removed from byBlock index.
		k.RemoveFromByBlockIndex(ctx, atomicSwap)
		k.SetAtomicSwap(ctx, atomicSwap)
		expiredSwapIDs = append(expiredSwapIDs, hex.EncodeToString(atomicSwap.GetSwapID()))
		return false
	})

	// Emit 'swaps_expired' event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSwapsExpired,
			sdk.NewAttribute(types.AttributeKeyAtomicSwapIDs, fmt.Sprintf("%s", expiredSwapIDs)),
			sdk.NewAttribute(types.AttributeExpirationBlock, fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)
}

// DeleteClosedAtomicSwapsFromLongtermStorage removes swaps one week after completion.
func (k Keeper) DeleteClosedAtomicSwapsFromLongtermStorage(ctx sdk.Context) {
	k.IterateAtomicSwapsLongtermStorage(ctx, uint64(ctx.BlockHeight()), func(id []byte) bool {
		swap, found := k.GetAtomicSwap(ctx, id)
		if !found {
			// NOTE: shouldn't happen. Continue to next item.
			return false
		}
		k.RemoveAtomicSwap(ctx, swap.GetSwapID())
		k.RemoveFromLongtermStorage(ctx, swap)
		return false
	})
}
