package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Borrow defines an amount of coins borrowed from a harvest module account
type Borrow struct {
	Borrower sdk.AccAddress `json:"borrower" yaml:"borrower"`
	Amount   sdk.Coins      `json:"amount" yaml:"amount"`
}

// NewBorrow returns a new Borrow instance
func NewBorrow(borrower sdk.AccAddress, amount sdk.Coins) Borrow {
	return Borrow{
		Borrower: borrower,
		Amount:   amount,
	}
}
