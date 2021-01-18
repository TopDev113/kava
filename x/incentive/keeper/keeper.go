package keeper

import (
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params/subspace"

	"github.com/kava-labs/kava/x/incentive/types"
)

// Keeper keeper for the incentive module
type Keeper struct {
	accountKeeper types.AccountKeeper
	cdc           *codec.Codec
	cdpKeeper     types.CdpKeeper
	key           sdk.StoreKey
	paramSubspace subspace.Subspace
	supplyKeeper  types.SupplyKeeper
}

// NewKeeper creates a new keeper
func NewKeeper(
	cdc *codec.Codec, key sdk.StoreKey, paramstore subspace.Subspace, sk types.SupplyKeeper,
	cdpk types.CdpKeeper, ak types.AccountKeeper,
) Keeper {

	return Keeper{
		accountKeeper: ak,
		cdc:           cdc,
		cdpKeeper:     cdpk,
		key:           key,
		paramSubspace: paramstore.WithKeyTable(types.ParamKeyTable()),
		supplyKeeper:  sk,
	}
}

// GetClaim returns the claim in the store corresponding the the input address collateral type and id and a boolean for if the claim was found
func (k Keeper) GetClaim(ctx sdk.Context, addr sdk.AccAddress) (types.USDXMintingClaim, bool) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.ClaimKeyPrefix)
	bz := store.Get(addr)
	if bz == nil {
		return types.USDXMintingClaim{}, false
	}
	var c types.USDXMintingClaim
	k.cdc.MustUnmarshalBinaryBare(bz, &c)
	return c, true
}

// SetClaim sets the claim in the store corresponding to the input address, collateral type, and id
func (k Keeper) SetClaim(ctx sdk.Context, c types.USDXMintingClaim) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.ClaimKeyPrefix)
	bz := k.cdc.MustMarshalBinaryBare(c)
	store.Set(c.Owner, bz)

}

// DeleteClaim deletes the claim in the store corresponding to the input address, collateral type, and id
func (k Keeper) DeleteClaim(ctx sdk.Context, owner sdk.AccAddress) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.ClaimKeyPrefix)
	store.Delete(owner)
}

// IterateClaims iterates over all claim  objects in the store and preforms a callback function
func (k Keeper) IterateClaims(ctx sdk.Context, cb func(c types.USDXMintingClaim) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.ClaimKeyPrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var c types.USDXMintingClaim
		k.cdc.MustUnmarshalBinaryBare(iterator.Value(), &c)
		if cb(c) {
			break
		}
	}
}

// GetAllClaims returns all Claim objects in the store
func (k Keeper) GetAllClaims(ctx sdk.Context) types.USDXMintingClaims {
	cs := types.USDXMintingClaims{}
	k.IterateClaims(ctx, func(c types.USDXMintingClaim) (stop bool) {
		cs = append(cs, c)
		return false
	})
	return cs
}

// GetPreviousAccrualTime returns the last time a collateral type accrued rewards
func (k Keeper) GetPreviousAccrualTime(ctx sdk.Context, ctype string) (blockTime time.Time, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.BlockTimeKey)
	bz := store.Get([]byte(ctype))
	if bz == nil {
		return time.Time{}, false
	}
	k.cdc.MustUnmarshalBinaryBare(bz, &blockTime)
	return blockTime, true
}

// SetPreviousAccrualTime sets the last time a collateral type accrued rewards
func (k Keeper) SetPreviousAccrualTime(ctx sdk.Context, ctype string, blockTime time.Time) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.BlockTimeKey)
	store.Set([]byte(ctype), k.cdc.MustMarshalBinaryBare(blockTime))
}

// IterateAccrualTimes iterates over all previous accrual times and preforms a callback function
func (k Keeper) IterateAccrualTimes(ctx sdk.Context, cb func(string, time.Time) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.BlockTimeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var accrualTime time.Time
		var collateralType string
		k.cdc.MustUnmarshalBinaryBare(iterator.Value(), &collateralType)
		k.cdc.MustUnmarshalBinaryBare(iterator.Value(), &accrualTime)
		if cb(collateralType, accrualTime) {
			break
		}
	}
}

// GetRewardFactor returns the current reward factor for an individual collateral type
func (k Keeper) GetRewardFactor(ctx sdk.Context, ctype string) (factor sdk.Dec, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.RewardFactorKey)
	bz := store.Get([]byte(ctype))
	if bz == nil {
		return sdk.ZeroDec(), false
	}
	k.cdc.MustUnmarshalBinaryBare(bz, &factor)
	return factor, true
}

// SetRewardFactor sets the current reward factor for an individual collateral type
func (k Keeper) SetRewardFactor(ctx sdk.Context, ctype string, factor sdk.Dec) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.RewardFactorKey)
	store.Set([]byte(ctype), k.cdc.MustMarshalBinaryBare(factor))
}
