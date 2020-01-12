package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/x/cdp/types"
)

// CalculateFees returns the fees accumulated since fees were last calculated based on
// the input amount of outstanding debt (principal) and the number of periods (seconds) that have passed
func (k Keeper) CalculateFees(ctx sdk.Context, principal sdk.Coins, periods sdk.Int, denom string) sdk.Coins {
	newFees := sdk.NewCoins()
	for _, pc := range principal {
		// how fees are calculated:
		// feesAccumulated = (outstandingDebt * (feeRate^periods)) - outstandingDebt
		// Note that since we can't do x^y using sdk.Decimal, we are converting to int and using RelativePow
		feePerSecond := k.GetFeeRate(ctx, denom)
		scalar := sdk.NewInt(1000000000000000000)
		feeRateInt := feePerSecond.Mul(sdk.NewDecFromInt(scalar)).TruncateInt()
		accumulator := sdk.NewDecFromInt(types.RelativePow(feeRateInt, periods, scalar)).Mul(sdk.SmallestDec())
		feesAccumulated := (sdk.NewDecFromInt(pc.Amount).Mul(accumulator)).Sub(sdk.NewDecFromInt(pc.Amount))
		// TODO this will always round down, causing precision loss between the sum of all fees in CDPs and surplus coins in liquidator account
		newFees = newFees.Add(sdk.NewCoins(sdk.NewCoin(pc.Denom, feesAccumulated.TruncateInt())))
	}
	return newFees
}

// IncrementTotalPrincipal increments the total amount of debt that has been drawn with that collateral type
func (k Keeper) IncrementTotalPrincipal(ctx sdk.Context, collateralDenom string, principal sdk.Coins) {
	for _, pc := range principal {
		total := k.GetTotalPrincipal(ctx, collateralDenom, pc.Denom)
		total = total.Add(pc.Amount)
		k.SetTotalPrincipal(ctx, collateralDenom, pc.Denom, total)
	}
}

// DecrementTotalPrincipal decrements the total amount of debt that has been drawn for a particular collateral type
func (k Keeper) DecrementTotalPrincipal(ctx sdk.Context, collateralDenom string, principal sdk.Coins) {
	for _, pc := range principal {
		total := k.GetTotalPrincipal(ctx, collateralDenom, pc.Denom)
		total = total.Sub(pc.Amount)
		if total.IsNegative() {
			// can happen in tests due to rounding errors in fee calculation
			total = sdk.ZeroInt()
		}
		k.SetTotalPrincipal(ctx, collateralDenom, pc.Denom, total)
	}
}

// GetTotalPrincipal returns the total amount of principal that has been drawn for a particular collateral
func (k Keeper) GetTotalPrincipal(ctx sdk.Context, collateralDenom string, principalDenom string) (total sdk.Int) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.PrincipalKeyPrefix)
	bz := store.Get([]byte(collateralDenom + principalDenom))
	if bz == nil {
		panic(fmt.Sprintf("total principal of %s for %s collateral not set in genesis", principalDenom, collateralDenom))
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &total)
	return total
}

// SetTotalPrincipal sets the total amount of principal that has been drawn for the input collateral
func (k Keeper) SetTotalPrincipal(ctx sdk.Context, collateralDenom string, principalDenom string, total sdk.Int) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.PrincipalKeyPrefix)
	store.Set([]byte(collateralDenom+principalDenom), k.cdc.MustMarshalBinaryLengthPrefixed(total))
}

// GetFeeRate returns the per second fee rate for the input denom
func (k Keeper) GetFeeRate(ctx sdk.Context, denom string) (fee sdk.Dec) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.AccumulatorKeyPrefix)
	bz := store.Get([]byte(denom))
	if bz == nil {
		panic(fmt.Sprintf("fee rate for %s not set in genesis", denom))
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &fee)
	return fee
}

// SetFeeRate sets the per second fee rate for the input denom
func (k Keeper) SetFeeRate(ctx sdk.Context, denom string, fee sdk.Dec) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.AccumulatorKeyPrefix)
	store.Set([]byte(denom), k.cdc.MustMarshalBinaryLengthPrefixed(fee))
}

// GetPreviousBlockTime get the blocktime for the previous block
func (k Keeper) GetPreviousBlockTime(ctx sdk.Context) (blockTime time.Time, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.PreviousBlockTimeKey)
	b := store.Get([]byte{})
	if b == nil {
		return time.Time{}, false
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &blockTime)
	return blockTime, true
}

// SetPreviousBlockTime set the time of the previous block
func (k Keeper) SetPreviousBlockTime(ctx sdk.Context, blockTime time.Time) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.PreviousBlockTimeKey)
	store.Set([]byte{}, k.cdc.MustMarshalBinaryLengthPrefixed(blockTime))
}
