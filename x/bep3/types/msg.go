package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	CreateAtomicSwap = "createAtomicSwap"
	ClaimAtomicSwap  = "claimAtomicSwap"
	RefundAtomicSwap = "refundAtomicSwap"
	CalcSwapID       = "calcSwapID"

	Int64Size               = 8
	RandomNumberHashLength  = 32
	RandomNumberLength      = 32
	MaxOtherChainAddrLength = 64
	SwapIDLength            = 32
	MaxExpectedIncomeLength = 64
)

// ensure Msg interface compliance at compile time
var (
	_                      sdk.Msg = &MsgCreateAtomicSwap{}
	_                      sdk.Msg = &MsgClaimAtomicSwap{}
	_                      sdk.Msg = &MsgRefundAtomicSwap{}
	AtomicSwapCoinsAccAddr         = sdk.AccAddress(crypto.AddressHash([]byte("KavaAtomicSwapCoins")))
)

// NewMsgCreateAtomicSwap initializes a new MsgCreateAtomicSwap
func NewMsgCreateAtomicSwap(from, to string, recipientOtherChain,
	senderOtherChain string, randomNumberHash tmbytes.HexBytes, timestamp int64,
	amount sdk.Coins, heightSpan uint64,
) MsgCreateAtomicSwap {
	return MsgCreateAtomicSwap{
		From:                from,
		To:                  to,
		RecipientOtherChain: recipientOtherChain,
		SenderOtherChain:    senderOtherChain,
		RandomNumberHash:    randomNumberHash.String(),
		Timestamp:           timestamp,
		Amount:              amount,
		HeightSpan:          heightSpan,
	}
}

// Route establishes the route for the MsgCreateAtomicSwap
func (msg MsgCreateAtomicSwap) Route() string { return RouterKey }

// Type is the name of MsgCreateAtomicSwap
func (msg MsgCreateAtomicSwap) Type() string { return CreateAtomicSwap }

// String prints the MsgCreateAtomicSwap
func (msg MsgCreateAtomicSwap) String() string {
	return fmt.Sprintf("AtomicSwap{%v#%v#%v#%v#%v#%v#%v#%v}",
		msg.From, msg.To, msg.RecipientOtherChain, msg.SenderOtherChain,
		msg.RandomNumberHash, msg.Timestamp, msg.Amount, msg.HeightSpan)
}

// GetInvolvedAddresses gets the addresses involved in a MsgCreateAtomicSwap
func (msg MsgCreateAtomicSwap) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}

// GetSigners gets the signers of a MsgCreateAtomicSwap
func (msg MsgCreateAtomicSwap) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}

// ValidateBasic validates the MsgCreateAtomicSwap
func (msg MsgCreateAtomicSwap) ValidateBasic() error {
	from, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
	}
	to, err := sdk.AccAddressFromBech32(msg.To)
	if err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
	}
	if from.Empty() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "sender address cannot be empty")
	}
	if to.Empty() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "recipient address cannot be empty")
	}
	if strings.TrimSpace(msg.RecipientOtherChain) == "" {
		return errors.New("missing recipient address on other chain")
	}
	if len(msg.RecipientOtherChain) > MaxOtherChainAddrLength {
		return fmt.Errorf("the length of recipient address on other chain should be less than %d", MaxOtherChainAddrLength)
	}
	if len(msg.SenderOtherChain) > MaxOtherChainAddrLength {
		return fmt.Errorf("the length of sender address on other chain should be less than %d", MaxOtherChainAddrLength)
	}
	randomNumberHash, err := hex.DecodeString(msg.RandomNumberHash)
	if err != nil {
		return fmt.Errorf("random number hash should be valid hex: %v", err)
	}
	if len(randomNumberHash) != RandomNumberHashLength {
		return fmt.Errorf("the length of random number hash should be %d", RandomNumberHashLength)
	}
	if msg.Timestamp <= 0 {
		return errors.New("timestamp must be positive")
	}
	if len(msg.Amount) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "amount cannot be empty")
	}
	if !msg.Amount.IsValid() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}
	if msg.HeightSpan <= 0 {
		return errors.New("height span must be positive")
	}
	return nil
}

// GetSignBytes gets the sign bytes of a MsgCreateAtomicSwap
func (msg MsgCreateAtomicSwap) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// NewMsgClaimAtomicSwap initializes a new MsgClaimAtomicSwap
func NewMsgClaimAtomicSwap(from string, swapID, randomNumber tmbytes.HexBytes) MsgClaimAtomicSwap {
	return MsgClaimAtomicSwap{
		From:         from,
		SwapID:       swapID.String(),
		RandomNumber: randomNumber.String(),
	}
}

// Route establishes the route for the MsgClaimAtomicSwap
func (msg MsgClaimAtomicSwap) Route() string { return RouterKey }

// Type is the name of MsgClaimAtomicSwap
func (msg MsgClaimAtomicSwap) Type() string { return ClaimAtomicSwap }

// String prints the MsgClaimAtomicSwap
func (msg MsgClaimAtomicSwap) String() string {
	return fmt.Sprintf("claimAtomicSwap{%v#%v#%v}", msg.From, msg.SwapID, msg.RandomNumber)
}

// GetInvolvedAddresses gets the addresses involved in a MsgClaimAtomicSwap
func (msg MsgClaimAtomicSwap) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}

// GetSigners gets the signers of a MsgClaimAtomicSwap
func (msg MsgClaimAtomicSwap) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}

// ValidateBasic validates the MsgClaimAtomicSwap
func (msg MsgClaimAtomicSwap) ValidateBasic() error {
	from, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
	}
	if from.Empty() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "sender address cannot be empty")
	}
	swapID, err := hex.DecodeString(msg.SwapID)
	if err != nil {
		return fmt.Errorf("swap id should be valid hex: %v", err)
	}
	if len(swapID) != SwapIDLength {
		return fmt.Errorf("the length of swapID should be %d", SwapIDLength)
	}
	randomNumber, err := hex.DecodeString(msg.RandomNumber)
	if err != nil {
		return fmt.Errorf("random number should be valid hex: %v", err)
	}
	if len(randomNumber) != RandomNumberLength {
		return fmt.Errorf("the length of random number should be %d", RandomNumberLength)
	}
	return nil
}

// GetSignBytes gets the sign bytes of a MsgClaimAtomicSwap
func (msg MsgClaimAtomicSwap) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// NewMsgRefundAtomicSwap initializes a new MsgRefundAtomicSwap
func NewMsgRefundAtomicSwap(from string, swapID tmbytes.HexBytes) MsgRefundAtomicSwap {
	return MsgRefundAtomicSwap{
		From:   from,
		SwapID: swapID.String(),
	}
}

// Route establishes the route for the MsgRefundAtomicSwap
func (msg MsgRefundAtomicSwap) Route() string { return RouterKey }

// Type is the name of MsgRefundAtomicSwap
func (msg MsgRefundAtomicSwap) Type() string { return RefundAtomicSwap }

// String prints the MsgRefundAtomicSwap
func (msg MsgRefundAtomicSwap) String() string {
	return fmt.Sprintf("refundAtomicSwap{%v#%v}", msg.From, msg.SwapID)
}

// GetInvolvedAddresses gets the addresses involved in a MsgRefundAtomicSwap
func (msg MsgRefundAtomicSwap) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}

// GetSigners gets the signers of a MsgRefundAtomicSwap
func (msg MsgRefundAtomicSwap) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}

// ValidateBasic validates the MsgRefundAtomicSwap
func (msg MsgRefundAtomicSwap) ValidateBasic() error {
	from, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
	}
	if from.Empty() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "sender address cannot be empty")
	}
	swapID, err := hex.DecodeString(msg.SwapID)
	if err != nil {
		return fmt.Errorf("swap id should be valid hex: %v", err)
	}
	if len(swapID) != SwapIDLength {
		return fmt.Errorf("the length of swapID should be %d", SwapIDLength)
	}
	return nil
}

// GetSignBytes gets the sign bytes of a MsgRefundAtomicSwap
func (msg MsgRefundAtomicSwap) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}
