package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/cdp/types"
)

// CalculateFees returns the fees accumulated since fees were last calculated based on
// the input amount of outstanding debt (principal) and the number of periods (seconds) that have passed
func (k Keeper) CalculateFees(ctx sdk.Context, principal sdk.Coin, periods sdk.Int, denom string) sdk.Coin {
	// how fees are calculated:
	// feesAccumulated = (outstandingDebt * (feeRate^periods)) - outstandingDebt
	// Note that since we can't do x^y using sdk.Decimal, we are converting to int and using RelativePow
	feePerSecond := k.getFeeRate(ctx, denom)
	scalar := sdk.NewInt(1000000000000000000)
	feeRateInt := feePerSecond.Mul(sdk.NewDecFromInt(scalar)).TruncateInt()
	accumulator := sdk.NewDecFromInt(types.RelativePow(feeRateInt, periods, scalar)).Mul(sdk.SmallestDec())
	feesAccumulated := (sdk.NewDecFromInt(principal.Amount).Mul(accumulator)).Sub(sdk.NewDecFromInt(principal.Amount))
	newFees := sdk.NewCoin(principal.Denom, feesAccumulated.TruncateInt())
	return newFees
}

// UpdateFeesForAllCdps updates the fees for each of the CDPs
func (k Keeper) UpdateFeesForAllCdps(ctx sdk.Context, collateralDenom string) error {
	var iterationErr error
	k.IterateCdpsByDenom(ctx, collateralDenom, func(cdp types.CDP) bool {

		oldCollateralToDebtRatio := k.CalculateCollateralToDebtRatio(ctx, cdp.Collateral, cdp.Principal.Add(cdp.AccumulatedFees))
		periods := sdk.NewInt(ctx.BlockTime().Unix()).Sub(sdk.NewInt(cdp.FeesUpdated.Unix()))

		newFees := k.CalculateFees(ctx, cdp.Principal, periods, collateralDenom)

		// exit without updating fees if amount has rounded down to zero
		// cdp will get updated next block when newFees, newFeesSavings, newFeesSurplus >0
		if newFees.IsZero() {
			return false
		}

		// note - only works if principal length is one
		dp, found := k.GetDebtParam(ctx, cdp.Principal.Denom)
		if !found {
			return false
		}
		savingsRate := dp.SavingsRate

		newFeesSavings := sdk.NewDecFromInt(newFees.Amount).Mul(savingsRate).RoundInt()
		newFeesSurplus := newFees.Amount.Sub(newFeesSavings)

		// similar to checking for rounding to zero of all fees, but in this case we
		// need to handle cases where we expect surplus or savings fees to be zero, namely
		// if newFeesSavings = 0, check if savings rate is not zero
		// if newFeesSurplus = 0, check if savings rate is not one
		if (newFeesSavings.IsZero() && !savingsRate.IsZero()) || (newFeesSurplus.IsZero() && !savingsRate.Equal(sdk.OneDec())) {
			return false
		}
		// mint debt coins to the cdp account
		k.MintDebtCoins(ctx, types.ModuleName, k.GetDebtDenom(ctx), newFees)
		previousDebt := k.GetTotalPrincipal(ctx, collateralDenom, dp.Denom)
		newDebt := previousDebt.Add(newFees.Amount)
		k.SetTotalPrincipal(ctx, collateralDenom, dp.Denom, newDebt)

		// mint surplus coins divided between the liquidator and savings module accounts.
		k.supplyKeeper.MintCoins(ctx, types.LiquidatorMacc, sdk.NewCoins(sdk.NewCoin(dp.Denom, newFeesSurplus)))
		k.supplyKeeper.MintCoins(ctx, types.SavingsRateMacc, sdk.NewCoins(sdk.NewCoin(dp.Denom, newFeesSavings)))

		// now add the new fees fees to the accumulated fees for the cdp
		cdp.AccumulatedFees = cdp.AccumulatedFees.Add(newFees)

		// and set the fees updated time to the current block time since we just updated it
		cdp.FeesUpdated = ctx.BlockTime()
		collateralToDebtRatio := k.CalculateCollateralToDebtRatio(ctx, cdp.Collateral, cdp.Principal.Add(cdp.AccumulatedFees))
		k.RemoveCdpCollateralRatioIndex(ctx, cdp.Collateral.Denom, cdp.ID, oldCollateralToDebtRatio)
		err := k.SetCdpAndCollateralRatioIndex(ctx, cdp, collateralToDebtRatio)
		if err != nil {
			iterationErr = err
			return true
		}
		return false // this returns true when you want to stop iterating. Since we want to iterate through all we return false
	})
	if iterationErr != nil {
		return iterationErr
	}
	return nil
}

// IncrementTotalPrincipal increments the total amount of debt that has been drawn with that collateral type
func (k Keeper) IncrementTotalPrincipal(ctx sdk.Context, collateralDenom string, principal sdk.Coin) {
	total := k.GetTotalPrincipal(ctx, collateralDenom, principal.Denom)
	total = total.Add(principal.Amount)
	k.SetTotalPrincipal(ctx, collateralDenom, principal.Denom, total)

}

// DecrementTotalPrincipal decrements the total amount of debt that has been drawn for a particular collateral type
func (k Keeper) DecrementTotalPrincipal(ctx sdk.Context, collateralDenom string, principal sdk.Coin) {
	total := k.GetTotalPrincipal(ctx, collateralDenom, principal.Denom)
	total = total.Sub(principal.Amount)
	if total.IsNegative() {
		// can happen in tests due to rounding errors in fee calculation
		total = sdk.ZeroInt()
	}
	k.SetTotalPrincipal(ctx, collateralDenom, principal.Denom, total)
}

// GetTotalPrincipal returns the total amount of principal that has been drawn for a particular collateral
func (k Keeper) GetTotalPrincipal(ctx sdk.Context, collateralDenom string, principalDenom string) (total sdk.Int) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.PrincipalKeyPrefix)
	bz := store.Get([]byte(collateralDenom + principalDenom))
	if bz == nil {
		k.SetTotalPrincipal(ctx, collateralDenom, principalDenom, sdk.ZeroInt())
		return sdk.ZeroInt()
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &total)
	return total
}

// SetTotalPrincipal sets the total amount of principal that has been drawn for the input collateral
func (k Keeper) SetTotalPrincipal(ctx sdk.Context, collateralDenom string, principalDenom string, total sdk.Int) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.PrincipalKeyPrefix)
	store.Set([]byte(collateralDenom+principalDenom), k.cdc.MustMarshalBinaryLengthPrefixed(total))
}
