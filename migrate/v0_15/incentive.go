package v0_15

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v0_14incentive "github.com/kava-labs/kava/x/incentive/legacy/v0_14"
	v0_15incentive "github.com/kava-labs/kava/x/incentive/types"
)

// Incentive migrates from a v0.14 incentive genesis state to a v0.15 incentive genesis state
func Incentive(incentiveGS v0_14incentive.GenesisState) v0_15incentive.GenesisState {
	// Migrate params
	claimMultipliers := v0_15incentive.Multipliers{}
	for _, m := range incentiveGS.Params.ClaimMultipliers {
		newMultiplier := v0_15incentive.NewMultiplier(v0_15incentive.MultiplierName(m.Name), m.MonthsLockup, m.Factor)
		claimMultipliers = append(claimMultipliers, newMultiplier)
	}

	usdxMintingRewardPeriods := v0_15incentive.RewardPeriods{}
	for _, rp := range incentiveGS.Params.USDXMintingRewardPeriods {
		usdxMintingRewardPeriod := v0_15incentive.NewRewardPeriod(rp.Active,
			rp.CollateralType, rp.Start, rp.End, rp.RewardsPerSecond)
		usdxMintingRewardPeriods = append(usdxMintingRewardPeriods, usdxMintingRewardPeriod)
	}

	hardDelegatorRewardPeriods := v0_15incentive.MultiRewardPeriods{}
	for _, rp := range incentiveGS.Params.HardDelegatorRewardPeriods {
		rewardsPerSecond := sdk.NewCoins(rp.RewardsPerSecond, SwpRewardsPerSecond)
		hardDelegatorRewardPeriod := v0_15incentive.NewMultiRewardPeriod(rp.Active,
			rp.CollateralType, rp.Start, rp.End, rewardsPerSecond)
		hardDelegatorRewardPeriods = append(hardDelegatorRewardPeriods, hardDelegatorRewardPeriod)
	}

	swapRewardPeriods := v0_15incentive.DefaultMultiRewardPeriods
	// TODO add expected swap reward periods

	// Build new params from migrated values
	params := v0_15incentive.NewParams(
		usdxMintingRewardPeriods,
		migrateMultiRewardPeriods(incentiveGS.Params.HardSupplyRewardPeriods),
		migrateMultiRewardPeriods(incentiveGS.Params.HardBorrowRewardPeriods),
		hardDelegatorRewardPeriods,
		swapRewardPeriods,
		claimMultipliers,
		incentiveGS.Params.ClaimEnd,
	)

	// Migrate accumulation times and reward indexes
	usdxGenesisRewardState := migrateGenesisRewardState(incentiveGS.USDXAccumulationTimes, incentiveGS.USDXRewardIndexes)
	hardSupplyGenesisRewardState := migrateGenesisRewardState(incentiveGS.HardSupplyAccumulationTimes, incentiveGS.HardSupplyRewardIndexes)
	hardBorrowGenesisRewardState := migrateGenesisRewardState(incentiveGS.HardBorrowAccumulationTimes, incentiveGS.HardBorrowRewardIndexes)
	delegatorGenesisRewardState := migrateGenesisRewardState(incentiveGS.HardDelegatorAccumulationTimes, incentiveGS.HardDelegatorRewardIndexes)
	swapGenesisRewardState := v0_15incentive.DefaultGenesisRewardState // There is no previous swap rewards so accumulation starts at genesis time.

	// Migrate USDX minting claims
	usdxMintingClaims := v0_15incentive.USDXMintingClaims{}
	for _, claim := range incentiveGS.USDXMintingClaims {
		rewardIndexes := migrateRewardIndexes(claim.RewardIndexes)
		usdxMintingClaim := v0_15incentive.NewUSDXMintingClaim(claim.Owner, claim.Reward, rewardIndexes)
		usdxMintingClaims = append(usdxMintingClaims, usdxMintingClaim)
	}

	// Migrate Hard protocol claims (includes creating new Delegator claims)
	hardClaims := v0_15incentive.HardLiquidityProviderClaims{}
	delegatorClaims := v0_15incentive.DelegatorClaims{}
	for _, claim := range incentiveGS.HardLiquidityProviderClaims {
		// Migrate supply multi reward indexes
		supplyMultiRewardIndexes := migrateMultiRewardIndexes(claim.SupplyRewardIndexes)

		// Migrate borrow multi reward indexes
		borrowMultiRewardIndexes := migrateMultiRewardIndexes(claim.BorrowRewardIndexes)

		// Migrate delegator reward indexes to multi reward indexes inside DelegatorClaims
		delegatorMultiRewardIndexes := v0_15incentive.MultiRewardIndexes{}
		delegatorRewardIndexes := v0_15incentive.RewardIndexes{}
		for _, ri := range claim.DelegatorRewardIndexes {
			// TODO add checks to ensure old reward indexes are as expected
			delegatorRewardIndex := v0_15incentive.NewRewardIndex(v0_14incentive.HardLiquidityRewardDenom, ri.RewardFactor)
			delegatorRewardIndexes = append(delegatorRewardIndexes, delegatorRewardIndex)
		}
		// TODO should this include indexes if none exist on the old claim?
		delegatorMultiRewardIndex := v0_15incentive.NewMultiRewardIndex(v0_14incentive.BondDenom, delegatorRewardIndexes)
		delegatorMultiRewardIndexes = append(delegatorMultiRewardIndexes, delegatorMultiRewardIndex)

		// TODO: It's impossible to distinguish between rewards from delegation vs. liquidity providing
		//		 as they're all combined inside claim.Reward, so I'm just putting them all inside
		// 		 the hard claim to avoid duplicating rewards.
		delegatorClaim := v0_15incentive.NewDelegatorClaim(claim.Owner, sdk.NewCoins(), delegatorMultiRewardIndexes)
		delegatorClaims = append(delegatorClaims, delegatorClaim)

		hardClaim := v0_15incentive.NewHardLiquidityProviderClaim(claim.Owner, claim.Reward,
			supplyMultiRewardIndexes, borrowMultiRewardIndexes)
		hardClaims = append(hardClaims, hardClaim)
	}

	// Add Swap Claims
	swapClaims := v0_15incentive.DefaultSwapClaims

	return v0_15incentive.NewGenesisState(
		params,
		usdxGenesisRewardState,
		hardSupplyGenesisRewardState,
		hardBorrowGenesisRewardState,
		delegatorGenesisRewardState,
		swapGenesisRewardState,
		usdxMintingClaims,
		hardClaims,
		delegatorClaims,
		swapClaims,
	)
}

func migrateMultiRewardPeriods(oldPeriods v0_14incentive.MultiRewardPeriods) v0_15incentive.MultiRewardPeriods {
	newPeriods := v0_15incentive.MultiRewardPeriods{}
	for _, rp := range oldPeriods {
		newPeriod := v0_15incentive.NewMultiRewardPeriod(
			rp.Active,
			rp.CollateralType,
			rp.Start,
			rp.End,
			rp.RewardsPerSecond,
		)
		newPeriods = append(newPeriods, newPeriod)
	}
	return newPeriods
}

func migrateGenesisRewardState(oldAccumulationTimes v0_14incentive.GenesisAccumulationTimes, oldIndexes v0_14incentive.GenesisRewardIndexesSlice) v0_15incentive.GenesisRewardState {
	accumulationTimes := v0_15incentive.AccumulationTimes{}
	for _, t := range oldAccumulationTimes {
		newAccumulationTime := v0_15incentive.NewAccumulationTime(t.CollateralType, t.PreviousAccumulationTime)
		accumulationTimes = append(accumulationTimes, newAccumulationTime)
	}
	multiRewardIndexes := v0_15incentive.MultiRewardIndexes{}
	for _, gri := range oldIndexes {
		multiRewardIndex := v0_15incentive.NewMultiRewardIndex(gri.CollateralType, migrateRewardIndexes(gri.RewardIndexes))
		multiRewardIndexes = append(multiRewardIndexes, multiRewardIndex)
	}
	return v0_15incentive.NewGenesisRewardState(
		accumulationTimes,
		multiRewardIndexes,
	)
}

func migrateMultiRewardIndexes(oldIndexes v0_14incentive.MultiRewardIndexes) v0_15incentive.MultiRewardIndexes {
	newIndexes := v0_15incentive.MultiRewardIndexes{}
	for _, mri := range oldIndexes {
		multiRewardIndex := v0_15incentive.NewMultiRewardIndex(
			mri.CollateralType,
			migrateRewardIndexes(mri.RewardIndexes),
		)
		newIndexes = append(newIndexes, multiRewardIndex)
	}
	return newIndexes
}

func migrateRewardIndexes(oldIndexes v0_14incentive.RewardIndexes) v0_15incentive.RewardIndexes {
	newIndexes := v0_15incentive.RewardIndexes{}
	for _, ri := range oldIndexes {
		rewardIndex := v0_15incentive.NewRewardIndex(ri.CollateralType, ri.RewardFactor)
		newIndexes = append(newIndexes, rewardIndex)
	}
	return newIndexes
}
