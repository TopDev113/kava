package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DONTCOVER

var (
	// ErrInvalidDepositDenom error for invalid deposit denoms
	ErrInvalidDepositDenom = sdkerrors.Register(ModuleName, 2, "invalid deposit denom")
	// ErrDepositNotFound error for deposit not found
	ErrDepositNotFound = sdkerrors.Register(ModuleName, 3, "deposit not found")
	// ErrInvalidWithdrawAmount error for invalid withdrawal amount
	ErrInvalidWithdrawAmount = sdkerrors.Register(ModuleName, 4, "invalid withdrawal amount")
	// ErrInsufficientModAccountBalance error for module account with innsufficient balance
	ErrInsufficientModAccountBalance = sdkerrors.Register(ModuleName, 5, "module account has insufficient balance to pay reward")
	// ErrInvalidAccountType error for unsupported accounts
	ErrInvalidAccountType = sdkerrors.Register(ModuleName, 6, "receiver account type not supported")
	// ErrAccountNotFound error for accounts that are not found in state
	ErrAccountNotFound = sdkerrors.Register(ModuleName, 7, "account not found")
	// ErrInvalidReceiver error for when sending and receiving accounts don't match
	ErrInvalidReceiver = sdkerrors.Register(ModuleName, 8, "receiver account must match sender account")
	// ErrMoneyMarketNotFound error for money market param not found
	ErrMoneyMarketNotFound = sdkerrors.Register(ModuleName, 9, "no money market found")
	// ErrDepositsNotFound error for no deposits found
	ErrDepositsNotFound = sdkerrors.Register(ModuleName, 10, "no deposits found")
	// ErrInsufficientLoanToValue error for when an attempted borrow exceeds maximum loan-to-value
	ErrInsufficientLoanToValue = sdkerrors.Register(ModuleName, 11, "not enough collateral supplied by account")
	// ErrMarketNotFound error for when a market for the input denom is not found
	ErrMarketNotFound = sdkerrors.Register(ModuleName, 12, "no market found for denom")
	// ErrPriceNotFound error for when a price for the input market is not found
	ErrPriceNotFound = sdkerrors.Register(ModuleName, 13, "no price found for market")
	// ErrBorrowExceedsAvailableBalance for when a requested borrow exceeds available module acc balances
	ErrBorrowExceedsAvailableBalance = sdkerrors.Register(ModuleName, 14, "exceeds module account balance")
	// ErrBorrowedCoinsNotFound error for when the total amount of borrowed coins cannot be found
	ErrBorrowedCoinsNotFound = sdkerrors.Register(ModuleName, 15, "no borrowed coins found")
	// ErrNegativeBorrowedCoins error for when substracting coins from the total borrowed balance results in a negative amount
	ErrNegativeBorrowedCoins = sdkerrors.Register(ModuleName, 16, "subtraction results in negative borrow amount")
	// ErrGreaterThanAssetBorrowLimit error for when a proposed borrow would increase borrowed amount over the asset's global borrow limit
	ErrGreaterThanAssetBorrowLimit = sdkerrors.Register(ModuleName, 17, "fails global asset borrow limit validation")
	// ErrBorrowEmptyCoins error for when you cannot borrow empty coins
	ErrBorrowEmptyCoins = sdkerrors.Register(ModuleName, 18, "cannot borrow zero coins")
	// ErrBorrowNotFound error for when a user's borrow is not found in the store
	ErrBorrowNotFound = sdkerrors.Register(ModuleName, 19, "borrow not found")
	// ErrPreviousAccrualTimeNotFound error for no previous accrual time found in store
	ErrPreviousAccrualTimeNotFound = sdkerrors.Register(ModuleName, 20, "no previous accrual time found")
	// ErrInsufficientBalanceForRepay error for when requested repay exceeds user's balance
	ErrInsufficientBalanceForRepay = sdkerrors.Register(ModuleName, 21, "insufficient balance")
	// ErrBorrowNotLiquidatable error for when a borrow is within valid LTV and cannot be liquidated
	ErrBorrowNotLiquidatable = sdkerrors.Register(ModuleName, 22, "borrow not liquidatable")
	// ErrInsufficientCoins error for when there are not enough coins for the operation
	ErrInsufficientCoins = sdkerrors.Register(ModuleName, 23, "unrecoverable state - insufficient coins")
	// ErrInsufficientBalanceForBorrow error for when the requested borrow exceeds user's balance
	ErrInsufficientBalanceForBorrow = sdkerrors.Register(ModuleName, 24, "insufficient balance")
	// ErrSuppliedCoinsNotFound error for when the total amount of supplied coins cannot be found
	ErrSuppliedCoinsNotFound = sdkerrors.Register(ModuleName, 25, "no supplied coins found")
	// ErrNegativeSuppliedCoins error for when substracting coins from the total supplied balance results in a negative amount
	ErrNegativeSuppliedCoins = sdkerrors.Register(ModuleName, 26, "subtraction results in negative supplied amount")
	// ErrInvalidWithdrawDenom error for when user attempts to withdraw a non-supplied coin type
	ErrInvalidWithdrawDenom = sdkerrors.Register(ModuleName, 27, "no coins of this type deposited")
	// ErrInvalidRepaymentDenom error for when user attempts to repay a non-borrowed coin type
	ErrInvalidRepaymentDenom = sdkerrors.Register(ModuleName, 28, "no coins of this type borrowed")
	// ErrInvalidIndexFactorDenom error for when index factor denom cannot be found
	ErrInvalidIndexFactorDenom = sdkerrors.Register(ModuleName, 29, "no index factor found for denom")
	// ErrBelowMinimumBorrowValue error for when a proposed borrow position is less than the minimum USD value
	ErrBelowMinimumBorrowValue = sdkerrors.Register(ModuleName, 30, "invalid proposed borrow value")
	// ErrExceedsProtocolBorrowableBalance for when a requested borrow exceeds the module account's borrowable balance
	ErrExceedsProtocolBorrowableBalance = sdkerrors.Register(ModuleName, 31, "exceeds borrowable module account balance")
	// ErrReservesExceedCash for when the protocol is insolvent because available reserves exceeds available cash
	ErrReservesExceedCash = sdkerrors.Register(ModuleName, 32, "insolvency - protocol reserves exceed available cash")
)
