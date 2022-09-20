package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/incentive/types"
)

// AccumulateSwapRewards calculates new rewards to distribute this block and updates the global indexes to reflect this.
// The provided rewardPeriod must be valid to avoid panics in calculating time durations.
func (k Keeper) AccumulateEarnRewards(ctx sdk.Context, rewardPeriod types.MultiRewardPeriod) {
	previousAccrualTime, found := k.GetSwapRewardAccrualTime(ctx, rewardPeriod.CollateralType)
	if !found {
		previousAccrualTime = ctx.BlockTime()
	}

	indexes, found := k.GetSwapRewardIndexes(ctx, rewardPeriod.CollateralType)
	if !found {
		indexes = types.RewardIndexes{}
	}

	acc := types.NewAccumulator(previousAccrualTime, indexes)

	totalSource := k.getSwapTotalSourceShares(ctx, rewardPeriod.CollateralType)

	acc.Accumulate(rewardPeriod, totalSource, ctx.BlockTime())

	k.SetSwapRewardAccrualTime(ctx, rewardPeriod.CollateralType, acc.PreviousAccumulationTime)
	if len(acc.Indexes) > 0 {
		// the store panics when setting empty or nil indexes
		k.SetSwapRewardIndexes(ctx, rewardPeriod.CollateralType, acc.Indexes)
	}
}

// getSwapTotalSourceShares fetches the sum of all source shares for a swap reward.
// In the case of swap, these are the total (swap module) shares in a particular pool.
func (k Keeper) getEarnTotalSourceShares(ctx sdk.Context, poolID string) sdk.Dec {
	totalShares, found := k.swapKeeper.GetPoolShares(ctx, poolID)
	if !found {
		totalShares = sdk.ZeroInt()
	}
	return totalShares.ToDec()
}

// InitializeSwapReward creates a new claim with zero rewards and indexes matching the global indexes.
// If the claim already exists it just updates the indexes.
func (k Keeper) InitializeEarnReward(ctx sdk.Context, poolID string, owner sdk.AccAddress) {
	claim, found := k.GetSwapClaim(ctx, owner)
	if !found {
		claim = types.NewSwapClaim(owner, sdk.Coins{}, nil)
	}

	globalRewardIndexes, found := k.GetSwapRewardIndexes(ctx, poolID)
	if !found {
		globalRewardIndexes = types.RewardIndexes{}
	}
	claim.RewardIndexes = claim.RewardIndexes.With(poolID, globalRewardIndexes)

	k.SetSwapClaim(ctx, claim)
}

// SynchronizeSwapReward updates the claim object by adding any accumulated rewards
// and updating the reward index value.
func (k Keeper) SynchronizeEarnReward(ctx sdk.Context, poolID string, owner sdk.AccAddress, shares sdk.Int) {
	claim, found := k.GetSwapClaim(ctx, owner)
	if !found {
		return
	}
	claim = k.synchronizeSwapReward(ctx, claim, poolID, owner, shares)

	k.SetSwapClaim(ctx, claim)
}

// synchronizeSwapReward updates the reward and indexes in a swap claim for one pool.
func (k *Keeper) synchronizeEarnReward(ctx sdk.Context, claim types.EarnClaim, poolID string, owner sdk.AccAddress, shares sdk.Int) types.EarnClaim {
	globalRewardIndexes, found := k.GetEarnRewardIndexes(ctx, poolID)
	if !found {
		// The global factor is only not found if
		// - the pool has not started accumulating rewards yet (either there is no reward specified in params, or the reward start time hasn't been hit)
		// - OR it was wrongly deleted from state (factors should never be removed while unsynced claims exist)
		// If not found we could either skip this sync, or assume the global factor is zero.
		// Skipping will avoid storing unnecessary factors in the claim for non rewarded pools.
		// And in the event a global factor is wrongly deleted, it will avoid this function panicking when calculating rewards.
		return claim
	}

	userRewardIndexes, found := claim.RewardIndexes.Get(poolID)
	if !found {
		// Normally the reward indexes should always be found.
		// But if a pool was not rewarded then becomes rewarded (ie a reward period is added to params), then the indexes will be missing from claims for that pool.
		// So given the reward period was just added, assume the starting value for any global reward indexes, which is an empty slice.
		userRewardIndexes = types.RewardIndexes{}
	}

	newRewards, err := k.CalculateRewards(userRewardIndexes, globalRewardIndexes, shares.ToDec())
	if err != nil {
		// Global reward factors should never decrease, as it would lead to a negative update to claim.Rewards.
		// This panics if a global reward factor decreases or disappears between the old and new indexes.
		panic(fmt.Sprintf("corrupted global reward indexes found: %v", err))
	}

	claim.Reward = claim.Reward.Add(newRewards...)
	claim.RewardIndexes = claim.RewardIndexes.With(poolID, globalRewardIndexes)

	return claim
}

// GetSynchronizedSwapClaim fetches a swap claim from the store and syncs rewards for all rewarded pools.
func (k Keeper) GetSynchronizedEarnClaim(ctx sdk.Context, owner sdk.AccAddress) (types.EarnClaim, bool) {
	claim, found := k.GetEarnClaim(ctx, owner)
	if !found {
		return types.EarnClaim{}, false
	}

	k.IterateEarnRewardIndexes(ctx, func(poolID string, _ types.RewardIndexes) bool {
		shares, found := k.swapKeeper.GetDepositorSharesAmount(ctx, owner, poolID)
		if !found {
			shares = sdk.ZeroInt()
		}

		claim = k.synchronizeEarnReward(ctx, claim, poolID, owner, shares)

		return false
	})

	return claim, true
}
