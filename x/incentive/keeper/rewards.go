package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/x/incentive/types"
)

// InitializeClaim creates a new claim with zero rewards and indexes matching
// the global indexes. If the claim already exists it just updates the indexes.
func (k Keeper) InitializeClaim(
	ctx sdk.Context,
	claimType types.ClaimType,
	sourceID string,
	owner sdk.AccAddress,
) {
	claim, found := k.GetClaim(ctx, claimType, owner)
	if !found {
		claim = types.NewClaim(claimType, owner, sdk.Coins{}, nil)
	}

	globalRewardIndexes, found := k.GetRewardIndexesOfClaimType(ctx, claimType, sourceID)
	if !found {
		globalRewardIndexes = types.RewardIndexes{}
	}

	claim.RewardIndexes = claim.RewardIndexes.With(sourceID, globalRewardIndexes)
	k.SetClaim(ctx, claim)
}

// SynchronizeClaim updates the claim object by adding any accumulated rewards
// and updating the reward index value.
func (k Keeper) SynchronizeClaim(
	ctx sdk.Context,
	claimType types.ClaimType,
	sourceID string,
	owner sdk.AccAddress,
	shares sdk.Dec,
) {
	claim, found := k.GetClaim(ctx, claimType, owner)
	if !found {
		return
	}

	claim = k.synchronizeClaim(ctx, claim, sourceID, owner, shares)
	k.SetClaim(ctx, claim)
}

// synchronizeClaim updates the reward and indexes in a claim for one sourceID.
func (k *Keeper) synchronizeClaim(
	ctx sdk.Context,
	claim types.Claim,
	sourceID string,
	owner sdk.AccAddress,
	shares sdk.Dec,
) types.Claim {
	globalRewardIndexes, found := k.GetRewardIndexesOfClaimType(ctx, claim.Type, sourceID)
	if !found {
		// The global factor is only not found if
		// - the pool has not started accumulating rewards yet (either there is no reward specified in params, or the reward start time hasn't been hit)
		// - OR it was wrongly deleted from state (factors should never be removed while unsynced claims exist)
		// If not found we could either skip this sync, or assume the global factor is zero.
		// Skipping will avoid storing unnecessary factors in the claim for non rewarded pools.
		// And in the event a global factor is wrongly deleted, it will avoid this function panicking when calculating rewards.
		return claim
	}

	userRewardIndexes, found := claim.RewardIndexes.Get(sourceID)
	if !found {
		// Normally the reward indexes should always be found.
		// But if a pool was not rewarded then becomes rewarded (ie a reward period is added to params), then the indexes will be missing from claims for that pool.
		// So given the reward period was just added, assume the starting value for any global reward indexes, which is an empty slice.
		userRewardIndexes = types.RewardIndexes{}
	}

	newRewards, err := k.CalculateRewards(userRewardIndexes, globalRewardIndexes, shares)
	if err != nil {
		// Global reward factors should never decrease, as it would lead to a negative update to claim.Rewards.
		// This panics if a global reward factor decreases or disappears between the old and new indexes.
		panic(fmt.Sprintf("corrupted global reward indexes found: %v", err))
	}

	claim.Reward = claim.Reward.Add(newRewards...)
	claim.RewardIndexes = claim.RewardIndexes.With(sourceID, globalRewardIndexes)

	return claim
}

// GetSynchronizedClaim fetches a claim from the store and syncs rewards for all
// rewarded sourceIDs.
func (k Keeper) GetSynchronizedClaim(
	ctx sdk.Context,
	claimType types.ClaimType,
	owner sdk.AccAddress,
) (types.Claim, bool) {
	claim, found := k.GetClaim(ctx, claimType, owner)
	if !found {
		return types.Claim{}, false
	}

	// Fetch all source IDs from indexes
	var sourceIDs []string
	k.IterateRewardIndexesByClaimType(ctx, claimType, func(rewardIndexes types.TypedRewardIndexes) bool {
		sourceIDs = append(sourceIDs, rewardIndexes.CollateralType)
		return false
	})

	adapter := k.GetSourceAdapter(claimType)
	accShares := adapter.OwnerSharesBySource(ctx, owner, sourceIDs)

	// Synchronize claim for each source ID
	for _, sourceID := range sourceIDs {
		claim = k.synchronizeClaim(ctx, claim, sourceID, owner, accShares[sourceID])
	}

	return claim, true
}
