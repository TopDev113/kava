package cdp

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/cdp/keeper"
	"github.com/kava-labs/kava/x/cdp/types"
)

// InitGenesis sets initial genesis state for cdp module
func InitGenesis(ctx sdk.Context, k keeper.Keeper, pk types.PricefeedKeeper, ak types.AccountKeeper, gs types.GenesisState) {
	if err := gs.Validate(); err != nil {
		panic(fmt.Sprintf("failed to validate %s genesis state: %s", types.ModuleName, err))
	}

	// check if the module accounts exists
	cdpModuleAcc := ak.GetModuleAccount(ctx, types.ModuleName)
	if cdpModuleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}
	liqModuleAcc := ak.GetModuleAccount(ctx, types.LiquidatorMacc)
	if liqModuleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.LiquidatorMacc))
	}

	// validate denoms - check that any collaterals in the params are in the pricefeed,
	// pricefeed MUST call InitGenesis before cdp
	collateralMap := make(map[string]int)
	ap := pk.GetParams(ctx)
	for _, a := range ap.Markets {
		collateralMap[a.MarketID] = 1
	}

	for _, col := range gs.Params.CollateralParams {
		_, found := collateralMap[col.SpotMarketID]
		if !found {
			panic(fmt.Sprintf("%s collateral market %v not found in pricefeed", col.Denom, col.SpotMarketID))
		}
		// sets the status of the pricefeed in the store
		// if pricefeed not active, debt operations are paused
		_ = k.UpdatePricefeedStatus(ctx, col.SpotMarketID)

		_, found = collateralMap[col.LiquidationMarketID]
		if !found {
			panic(fmt.Sprintf("%s collateral market %v not found in pricefeed", col.Denom, col.LiquidationMarketID))
		}
		// sets the status of the pricefeed in the store
		// if pricefeed not active, debt operations are paused
		_ = k.UpdatePricefeedStatus(ctx, col.LiquidationMarketID)
	}

	k.SetParams(ctx, gs.Params)

	for _, gat := range gs.PreviousAccumulationTimes {
		k.SetInterestFactor(ctx, gat.CollateralType, gat.InterestFactor)
		if gat.PreviousAccumulationTime.Unix() > 0 {
			k.SetPreviousAccrualTime(ctx, gat.CollateralType, gat.PreviousAccumulationTime)
		}
	}

	for _, gtp := range gs.TotalPrincipals {
		k.SetTotalPrincipal(ctx, gtp.CollateralType, types.DefaultStableDenom, gtp.TotalPrincipal)
	}
	// add cdps
	for _, cdp := range gs.CDPs {
		if cdp.ID == gs.StartingCdpID {
			panic(fmt.Sprintf("starting cdp id is assigned to an existing cdp: %v", cdp))
		}
		err := k.SetCDP(ctx, cdp)
		if err != nil {
			panic(fmt.Sprintf("error setting cdp: %v", err))
		}
		k.IndexCdpByOwner(ctx, cdp)
		ratio := k.CalculateCollateralToDebtRatio(ctx, cdp.Collateral, cdp.Type, cdp.GetTotalPrincipal())
		k.IndexCdpByCollateralRatio(ctx, cdp.Type, cdp.ID, ratio)
	}

	k.SetNextCdpID(ctx, gs.StartingCdpID)
	k.SetDebtDenom(ctx, gs.DebtDenom)
	k.SetGovDenom(ctx, gs.GovDenom)

	for _, d := range gs.Deposits {
		k.SetDeposit(ctx, d)
	}
}

// ExportGenesis export genesis state for cdp module
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) types.GenesisState {
	params := k.GetParams(ctx)

	cdps := types.CDPs{}
	deposits := types.Deposits{}
	k.IterateAllCdps(ctx, func(cdp types.CDP) (stop bool) {
		syncedCdp := k.SynchronizeInterest(ctx, cdp)
		cdps = append(cdps, syncedCdp)
		k.IterateDeposits(ctx, cdp.ID, func(deposit types.Deposit) (stop bool) {
			deposits = append(deposits, deposit)
			return false
		})
		return false
	})

	cdpID := k.GetNextCdpID(ctx)
	debtDenom := k.GetDebtDenom(ctx)
	govDenom := k.GetGovDenom(ctx)

	var previousAccumTimes types.GenesisAccumulationTimes
	var totalPrincipals types.GenesisTotalPrincipals

	for _, cp := range params.CollateralParams {
		interestFactor, found := k.GetInterestFactor(ctx, cp.Type)
		if !found {
			interestFactor = sdk.OneDec()
		}
		// Governance param changes happen in the end blocker. If a new collateral type is added and then the chain
		// is exported before the BeginBlocker can run, previous accrual time won't be found. We can't set it to
		// current block time because it is not available in the export ctx. We should panic instead of exporting
		// bad state.
		previousAccumTime, f := k.GetPreviousAccrualTime(ctx, cp.Type)
		if !f {
			panic(fmt.Sprintf("expected previous accrual time to be set in state for %s", cp.Type))
		}
		previousAccumTimes = append(previousAccumTimes, types.NewGenesisAccumulationTime(cp.Type, previousAccumTime, interestFactor))

		tp := k.GetTotalPrincipal(ctx, cp.Type, types.DefaultStableDenom)
		genTotalPrincipal := types.NewGenesisTotalPrincipal(cp.Type, tp)
		totalPrincipals = append(totalPrincipals, genTotalPrincipal)
	}

	return types.NewGenesisState(params, cdps, deposits, cdpID, debtDenom, govDenom, previousAccumTimes, totalPrincipals)
}
