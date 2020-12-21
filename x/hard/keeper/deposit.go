package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	supplyExported "github.com/cosmos/cosmos-sdk/x/supply/exported"

	"github.com/kava-labs/kava/x/hard/types"
)

// Deposit deposit
func (k Keeper) Deposit(ctx sdk.Context, depositor sdk.AccAddress, coins sdk.Coins) error {
	// Get current stored LTV based on stored borrows/deposits
	prevLtv, shouldRemoveIndex, err := k.GetStoreLTV(ctx, depositor)
	if err != nil {
		return err
	}

	k.SyncOutstandingInterest(ctx, depositor)

	err = k.ValidateDeposit(ctx, coins)
	if err != nil {
		return err
	}

	err = k.supplyKeeper.SendCoinsFromAccountToModule(ctx, depositor, types.ModuleAccountName, coins)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient account funds") {
			accCoins := k.accountKeeper.GetAccount(ctx, depositor).SpendableCoins(ctx.BlockTime())
			for _, coin := range coins {
				_, isNegative := accCoins.SafeSub(sdk.NewCoins(coin))
				if isNegative {
					return sdkerrors.Wrapf(types.ErrBorrowExceedsAvailableBalance,
						"insufficient funds: the requested deposit amount of %s exceeds the total available account funds of %s%s",
						coin, accCoins.AmountOf(coin.Denom), coin.Denom,
					)
				}
			}
		}
	}
	if err != nil {
		return err
	}

	deposit, found := k.GetDeposit(ctx, depositor)
	if !found {
		deposit = types.NewDeposit(depositor, coins)
	} else {
		deposit.Amount = deposit.Amount.Add(coins...)
	}

	k.SetDeposit(ctx, deposit)

	k.UpdateItemInLtvIndex(ctx, prevLtv, shouldRemoveIndex, depositor)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeHardDeposit,
			sdk.NewAttribute(sdk.AttributeKeyAmount, coins.String()),
			sdk.NewAttribute(types.AttributeKeyDepositor, deposit.Depositor.String()),
		),
	)

	return nil
}

// ValidateDeposit validates a deposit
func (k Keeper) ValidateDeposit(ctx sdk.Context, coins sdk.Coins) error {
	params := k.GetParams(ctx)
	for _, depCoin := range coins {
		found := false
		for _, lps := range params.LiquidityProviderSchedules {
			if lps.DepositDenom == depCoin.Denom {
				found = true
			}
		}
		if !found {
			return sdkerrors.Wrapf(types.ErrInvalidDepositDenom, "liquidity provider denom %s not found", depCoin.Denom)
		}
	}

	return nil
}

// Withdraw returns some or all of a deposit back to original depositor
func (k Keeper) Withdraw(ctx sdk.Context, depositor sdk.AccAddress, coins sdk.Coins) error {
	deposit, found := k.GetDeposit(ctx, depositor)
	if !found {
		return sdkerrors.Wrapf(types.ErrDepositNotFound, "no deposit found for %s", depositor)
	}

	// Get current stored LTV based on stored borrows/deposits
	prevLtv, shouldRemoveIndex, err := k.GetStoreLTV(ctx, depositor)
	if err != nil {
		return err
	}

	k.SyncOutstandingInterest(ctx, depositor)

	borrow, found := k.GetBorrow(ctx, depositor)
	if !found {
		borrow = types.Borrow{}
	}

	proposedDepositAmount, isNegative := deposit.Amount.SafeSub(coins)
	if isNegative {
		return types.ErrNegativeBorrowedCoins
	}
	proposedDeposit := types.NewDeposit(deposit.Depositor, proposedDepositAmount)

	valid, err := k.IsWithinValidLtvRange(ctx, proposedDeposit, borrow)
	if err != nil {
		return err
	}

	if !valid {
		return sdkerrors.Wrapf(types.ErrInvalidWithdrawAmount, "proposed withdraw outside loan-to-value range")
	}

	err = k.supplyKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleAccountName, depositor, coins)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeHardWithdrawal,
			sdk.NewAttribute(sdk.AttributeKeyAmount, coins.String()),
			sdk.NewAttribute(types.AttributeKeyDepositor, depositor.String()),
		),
	)

	if deposit.Amount.IsEqual(coins) {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeDeleteHardDeposit,
				sdk.NewAttribute(types.AttributeKeyDepositor, depositor.String()),
			),
		)
		k.DeleteDeposit(ctx, deposit)
		return nil
	}

	deposit.Amount = deposit.Amount.Sub(coins)
	k.SetDeposit(ctx, deposit)

	k.UpdateItemInLtvIndex(ctx, prevLtv, shouldRemoveIndex, depositor)

	return nil
}

// GetTotalDeposited returns the total amount deposited for the input deposit type and deposit denom
func (k Keeper) GetTotalDeposited(ctx sdk.Context, depositDenom string) (total sdk.Int) {
	var macc supplyExported.ModuleAccountI
	macc = k.supplyKeeper.GetModuleAccount(ctx, types.ModuleAccountName)
	return macc.GetCoins().AmountOf(depositDenom)
}
