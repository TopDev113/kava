package v3

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/x/incentive/types"
)

// MigrateStore performs in-place migrations from incentive ConsensusVersion 2 to 3.
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)

	if err := MigrateEarnClaims(store, cdc); err != nil {
		return err
	}

	if err := MigrateAccrualTimes(store, cdc, types.CLAIM_TYPE_EARN); err != nil {
		return err
	}

	if err := MigrateRewardIndexes(store, cdc, types.CLAIM_TYPE_EARN); err != nil {
		return err
	}

	return nil
}

// MigrateEarnClaims migrates earn claims from v2 to v3
func MigrateEarnClaims(store sdk.KVStore, cdc codec.BinaryCodec) error {
	newStore := prefix.NewStore(store, types.GetClaimKeyPrefix(types.CLAIM_TYPE_EARN))

	oldStore := prefix.NewStore(store, EarnClaimKeyPrefix)
	iterator := sdk.KVStorePrefixIterator(oldStore, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var c types.EarnClaim
		cdc.MustUnmarshal(iterator.Value(), &c)

		if err := c.Validate(); err != nil {
			return fmt.Errorf("invalid v2 EarnClaim: %w", err)
		}

		// Convert to the new Claim type
		newClaim := types.NewClaim(
			types.CLAIM_TYPE_EARN,
			c.Owner,
			c.Reward,
			c.RewardIndexes,
		)

		if err := newClaim.Validate(); err != nil {
			return fmt.Errorf("invalid v3 EarnClaim: %w", err)
		}

		// Set in the **newStore** for the new store prefix
		newStore.Set(c.Owner, cdc.MustMarshal(&newClaim))

		// Remove the old claim in the old store
		oldStore.Delete(iterator.Key())
	}

	return nil
}

// MigrateAccrualTimes migrates accrual times from v2 to v3
func MigrateAccrualTimes(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	claimType types.ClaimType,
) error {
	newStore := prefix.NewStore(store, types.GetPreviousRewardAccrualTimeKeyPrefix(claimType))

	// Need prefix.NewStore instead of using it directly in the iterator, as
	// there would be an extra space in the key
	legacyPrefix := LegacyAccrualTimeKeyFromClaimType(claimType)
	oldStore := prefix.NewStore(store, legacyPrefix)
	iterator := sdk.KVStorePrefixIterator(oldStore, []byte{})

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var blockTime time.Time
		if err := blockTime.UnmarshalBinary(iterator.Value()); err != nil {
			panic(err)
		}

		sourceID := string(iterator.Key())

		at := types.NewAccrualTime(claimType, sourceID, blockTime)
		if err := at.Validate(); err != nil {
			return fmt.Errorf("invalid v3 accrual time for claim type %s: %w", claimType, err)
		}

		// Set in the **newStore** for the new store prefix
		bz := cdc.MustMarshal(&at)
		newStore.Set(types.GetKeyFromSourceID(sourceID), bz)

		// Remove the old accrual time in the old store
		oldStore.Delete(iterator.Key())
	}

	return nil
}

// MigrateRewardIndexes migrates reward indexes from v2 to v3
func MigrateRewardIndexes(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	claimType types.ClaimType,
) error {
	newStore := prefix.NewStore(store, types.GetRewardIndexesKeyPrefix(claimType))

	legacyPrefix := LegacyRewardIndexesKeyFromClaimType(claimType)
	oldStore := prefix.NewStore(store, legacyPrefix)
	iterator := sdk.KVStorePrefixIterator(oldStore, []byte{})

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var proto types.RewardIndexesProto
		cdc.MustUnmarshal(iterator.Value(), &proto)

		sourceID := string(iterator.Key())

		rewardIndex := types.NewTypedRewardIndexes(
			claimType,
			sourceID,
			proto.RewardIndexes,
		)

		bz := cdc.MustMarshal(&rewardIndex)
		newStore.Set(types.GetKeyFromSourceID(sourceID), bz)

		// Remove the old reward indexes in the old store
		oldStore.Delete(iterator.Key())
	}

	return nil
}
