package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/cdp/types"
)

// SeizeCollateral liquidates the collateral in the input cdp.
// the following operations are performed:
// 1. Collateral for all deposits is sent from the cdp module to the liquidator module account
// 2. The liquidation penalty is applied
// 3. Debt coins are sent from the cdp module to the liquidator module account
// 4. The total amount of principal outstanding for that collateral type is decremented
// (this is the equivalent of saying that fees are no longer accumulated by a cdp once it gets liquidated)
func (k Keeper) SeizeCollateral(ctx sdk.Context, cdp types.CDP) error {
	// Calculate the previous collateral ratio
	oldCollateralToDebtRatio := k.CalculateCollateralToDebtRatio(ctx, cdp.Collateral, cdp.Type, cdp.GetTotalPrincipal())

	// Move debt coins from cdp to liquidator account
	deposits := k.GetDeposits(ctx, cdp.ID)
	debt := cdp.GetTotalPrincipal().Amount
	modAccountDebt := k.getModAccountDebt(ctx, types.ModuleName)
	debt = sdk.MinInt(debt, modAccountDebt)
	debtCoin := sdk.NewCoin(k.GetDebtDenom(ctx), debt)
	err := k.supplyKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, types.LiquidatorMacc, sdk.NewCoins(debtCoin))
	if err != nil {
		return err
	}

	// liquidate deposits and send collateral from cdp to liquidator
	for _, dep := range deposits {
		err := k.supplyKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, types.LiquidatorMacc, sdk.NewCoins(dep.Amount))
		if err != nil {
			return err
		}
		k.DeleteDeposit(ctx, dep.CdpID, dep.Depositor)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCdpLiquidation,
				sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
				sdk.NewAttribute(types.AttributeKeyCdpID, fmt.Sprintf("%d", cdp.ID)),
				sdk.NewAttribute(types.AttributeKeyDeposit, dep.String()),
			),
		)
	}

	err = k.AuctionCollateral(ctx, deposits, cdp.Type, debt, cdp.Principal.Denom)
	if err != nil {
		return err
	}

	// Decrement total principal for this collateral type
	coinsToDecrement := cdp.GetTotalPrincipal()
	k.DecrementTotalPrincipal(ctx, cdp.Type, coinsToDecrement)

	// Delete CDP from state
	k.RemoveCdpOwnerIndex(ctx, cdp)
	k.RemoveCdpCollateralRatioIndex(ctx, cdp.Type, cdp.ID, oldCollateralToDebtRatio)
	return k.DeleteCDP(ctx, cdp)
}

// LiquidateCdps seizes collateral from all CDPs below the input liquidation ratio
func (k Keeper) LiquidateCdps(ctx sdk.Context, marketID string, collateralType string, liquidationRatio sdk.Dec) error {
	price, err := k.pricefeedKeeper.GetCurrentPrice(ctx, marketID)
	if err != nil {
		return err
	}
	priceDivLiqRatio := price.Price.Quo(liquidationRatio)
	if priceDivLiqRatio.IsZero() {
		priceDivLiqRatio = sdk.SmallestDec()
	}
	// price = $0.5
	// liquidation ratio = 1.5
	// normalizedRatio = (1/(0.5/1.5)) = 3
	normalizedRatio := sdk.OneDec().Quo(priceDivLiqRatio)
	cdpsToLiquidate := k.GetAllCdpsByCollateralTypeAndRatio(ctx, collateralType, normalizedRatio)
	for _, c := range cdpsToLiquidate {
		err := k.SeizeCollateral(ctx, c)
		if err != nil {
			return err
		}
	}
	return nil
}

// ApplyLiquidationPenalty multiplies the input debt amount by the liquidation penalty
func (k Keeper) ApplyLiquidationPenalty(ctx sdk.Context, collateralType string, debt sdk.Int) sdk.Int {
	penalty := k.getLiquidationPenalty(ctx, collateralType)
	return sdk.NewDecFromInt(debt).Mul(penalty).RoundInt()
}

func (k Keeper) getModAccountDebt(ctx sdk.Context, accountName string) sdk.Int {
	macc := k.supplyKeeper.GetModuleAccount(ctx, accountName)
	return macc.GetCoins().AmountOf(k.GetDebtDenom(ctx))
}
