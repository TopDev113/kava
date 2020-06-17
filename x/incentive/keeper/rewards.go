package keeper

import (
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	"github.com/kava-labs/kava/x/incentive/types"
)

// HandleRewardPeriodExpiry deletes expired RewardPeriods from the store and creates a ClaimPeriod in the store for each expired RewardPeriod
func (k Keeper) HandleRewardPeriodExpiry(ctx sdk.Context, rp types.RewardPeriod) {
	k.CreateUniqueClaimPeriod(ctx, rp.Denom, rp.ClaimEnd, rp.ClaimTimeLock)
	store := prefix.NewStore(ctx.KVStore(k.key), types.RewardPeriodKeyPrefix)
	store.Delete([]byte(rp.Denom))
	return
}

// CreateNewRewardPeriod creates a new reward period from the input reward
func (k Keeper) CreateNewRewardPeriod(ctx sdk.Context, reward types.Reward) {
	rp := types.NewRewardPeriodFromReward(reward, ctx.BlockTime())
	k.SetRewardPeriod(ctx, rp)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewardPeriod,
			sdk.NewAttribute(types.AttributeKeyRewardPeriod, rp.String()),
		),
	)
}

// CreateAndDeleteRewardPeriods creates reward periods for active rewards that don't already have a reward period and deletes reward periods for inactive rewards that currently have a reward period
func (k Keeper) CreateAndDeleteRewardPeriods(ctx sdk.Context) {
	params := k.GetParams(ctx)

	for _, r := range params.Rewards {
		_, found := k.GetRewardPeriod(ctx, r.Denom)
		// if governance has made a reward inactive, delete the current period
		if found && !r.Active {
			k.DeleteRewardPeriod(ctx, r.Denom)
		}
		// if a reward period for an active reward is not found, create one
		if !found && r.Active {
			k.CreateNewRewardPeriod(ctx, r)
		}
	}
}

// ApplyRewardsToCdps iterates over the reward periods and creates a claim for each
// cdp owner that created usdx with the collateral specified in the reward period.
func (k Keeper) ApplyRewardsToCdps(ctx sdk.Context) {
	previousBlockTime, found := k.GetPreviousBlockTime(ctx)
	if !found {
		previousBlockTime = ctx.BlockTime()
		k.SetPreviousBlockTime(ctx, previousBlockTime)
		return
	}

	k.IterateRewardPeriods(ctx, func(rp types.RewardPeriod) bool {
		expired := false
		// the total amount of usdx created with the collateral type being incentivized
		totalPrincipal := k.cdpKeeper.GetTotalPrincipal(ctx, rp.Denom, types.PrincipalDenom)
		// the number of seconds since last payout
		timeElapsed := sdk.NewInt(ctx.BlockTime().Unix() - previousBlockTime.Unix())
		if rp.End.Before(ctx.BlockTime()) {
			timeElapsed = sdk.NewInt(rp.End.Unix() - previousBlockTime.Unix())
			expired = true
		}

		// the amount of rewards to pay (rewardAmount * timeElapsed)
		rewardsThisPeriod := rp.Reward.Amount.Mul(timeElapsed)
		id := k.GetNextClaimPeriodID(ctx, rp.Denom)
		k.cdpKeeper.IterateCdpsByDenom(ctx, rp.Denom, func(cdp cdptypes.CDP) bool {
			rewardsShare := sdk.NewDecFromInt(cdp.Principal.Amount.Add(cdp.AccumulatedFees.Amount)).Quo(sdk.NewDecFromInt(totalPrincipal))
			// sanity check - don't create zero claims
			if rewardsShare.IsZero() {
				return false
			}
			rewardsEarned := rewardsShare.Mul(sdk.NewDecFromInt(rewardsThisPeriod)).RoundInt()
			k.AddToClaim(ctx, cdp.Owner, rp.Denom, id, sdk.NewCoin(types.GovDenom, rewardsEarned))
			return false
		})
		if !expired {
			return false
		}
		k.HandleRewardPeriodExpiry(ctx, rp)
		return false
	})

	k.SetPreviousBlockTime(ctx, ctx.BlockTime())
}

// CreateUniqueClaimPeriod creates a new claim period in the store and updates the highest claim period id
func (k Keeper) CreateUniqueClaimPeriod(ctx sdk.Context, denom string, end time.Time, timeLock time.Duration) {
	id := k.GetNextClaimPeriodID(ctx, denom)
	claimPeriod := types.NewClaimPeriod(denom, id, end, timeLock)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeClaimPeriod,
			sdk.NewAttribute(types.AttributeKeyClaimPeriod, claimPeriod.String()),
		),
	)
	k.SetClaimPeriod(ctx, claimPeriod)
	k.SetNextClaimPeriodID(ctx, denom, id+1)
}

// AddToClaim adds the amount to an existing claim or creates a new one for that amount
func (k Keeper) AddToClaim(ctx sdk.Context, addr sdk.AccAddress, denom string, id uint64, amount sdk.Coin) {
	claim, found := k.GetClaim(ctx, addr, denom, id)
	if found {
		claim.Reward = claim.Reward.Add(amount)
	} else {
		claim = types.NewClaim(addr, amount, denom, id)
	}
	k.SetClaim(ctx, claim)
}
