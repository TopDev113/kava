package v0_15

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName name used throughout module
	ModuleName = "validatorvesting"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// QuerierRoute should be set to module name
	QuerierRoute = ModuleName

	// QueryPath shortened name for public API (cli and REST)
	QueryPath = "vesting"
)

var (
	// BlocktimeKey key for the time of the previous block
	BlocktimeKey = []byte{0x00}
	// ValidatorVestingAccountPrefix store prefix for validator vesting accounts
	ValidatorVestingAccountPrefix = []byte{0x01}
)

// ValidatorVestingAccountKey returns the account address bytes prefixed by ValidatorVestingAccountPrefix
func ValidatorVestingAccountKey(addr sdk.AccAddress) []byte {
	return append(ValidatorVestingAccountPrefix, addr.Bytes()...)
}
