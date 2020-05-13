package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName is the name of the module
	ModuleName = "bep3"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute is the querier route for bep3
	QuerierRoute = ModuleName

	// DefaultParamspace default namestore
	DefaultParamspace = ModuleName
)

// DefaultLongtermStorageDuration is 1 week (assuming a block time of 7 seconds)
const DefaultLongtermStorageDuration uint64 = 86400

// Key prefixes
var (
	AtomicSwapKeyPrefix             = []byte{0x00} // prefix for keys that store AtomicSwaps
	AtomicSwapByBlockPrefix         = []byte{0x01} // prefix for keys of the AtomicSwapsByBlock index
	AssetSupplyKeyPrefix            = []byte{0x02} // prefix for keys that store global asset supply counts
	AtomicSwapLongtermStoragePrefix = []byte{0x03} // prefix for keys of the AtomicSwapLongtermStorage index
)

// GetAtomicSwapByHeightKey is used by the AtomicSwapByBlock index and AtomicSwapLongtermStorage index
func GetAtomicSwapByHeightKey(height uint64, swapID []byte) []byte {
	return append(sdk.Uint64ToBigEndian(height), swapID...)
}
