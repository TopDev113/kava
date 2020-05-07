package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RewardPeriod stores the state of an ongoing reward
type RewardPeriod struct {
	Denom         string        `json:"denom" yaml:"denom"`
	Start         time.Time     `json:"start" yaml:"start"`
	End           time.Time     `json:"end" yaml:"end"`
	Reward        sdk.Coin      `json:"reward" yaml:"reward"` // per second reward payouts
	ClaimEnd      time.Time     `json:"claim_end" yaml:"claim_end"`
	ClaimTimeLock time.Duration `json:"claim_time_lock" yaml:"claim_time_lock"` // the amount of time rewards are timelocked once they are sent to users
}

// String implements fmt.Stringer
func (rp RewardPeriod) String() string {
	return fmt.Sprintf(`Reward Period:
	Denom: %s,
	Start: %s,
	End: %s,
	Reward: %s,
	Claim End: %s,
	Claim Time Lock: %s
	`, rp.Denom, rp.Start, rp.End, rp.Reward, rp.ClaimEnd, rp.ClaimTimeLock)
}

// NewRewardPeriod returns a new RewardPeriod
func NewRewardPeriod(denom string, start time.Time, end time.Time, reward sdk.Coin, claimEnd time.Time, claimTimeLock time.Duration) RewardPeriod {
	return RewardPeriod{
		Denom:         denom,
		Start:         start,
		End:           end,
		Reward:        reward,
		ClaimEnd:      claimEnd,
		ClaimTimeLock: claimTimeLock,
	}
}

// RewardPeriods array of RewardPeriod
type RewardPeriods []RewardPeriod

// ClaimPeriod stores the state of an ongoing claim period
type ClaimPeriod struct {
	Denom    string        `json:"denom" yaml:"denom"`
	ID       uint64        `json:"id" yaml:"id"`
	End      time.Time     `json:"end" yaml:"end"`
	TimeLock time.Duration `json:"time_lock" yaml:"time_lock"`
}

// String implements fmt.Stringer
func (cp ClaimPeriod) String() string {
	return fmt.Sprintf(`Claim Period:
	Denom: %s,
	ID: %d,
	End: %s,
	Claim Time Lock: %s
	`, cp.Denom, cp.ID, cp.End, cp.TimeLock)
}

// NewClaimPeriod returns a new ClaimPeriod
func NewClaimPeriod(denom string, id uint64, end time.Time, timeLock time.Duration) ClaimPeriod {
	return ClaimPeriod{
		Denom:    denom,
		ID:       id,
		End:      end,
		TimeLock: timeLock,
	}
}

// ClaimPeriods array of ClaimPeriod
type ClaimPeriods []ClaimPeriod

// Claim stores the rewards that can be claimed by owner
type Claim struct {
	Owner         sdk.AccAddress `json:"owner" yaml:"owner"`
	Reward        sdk.Coin       `json:"reward" yaml:"reward"`
	Denom         string         `json:"denom" yaml:"denom"`
	ClaimPeriodID uint64         `json:"claim_period_id" yaml:"claim_period_id"`
}

// NewClaim returns a new Claim
func NewClaim(owner sdk.AccAddress, reward sdk.Coin, denom string, claimPeriodID uint64) Claim {
	return Claim{
		Owner:         owner,
		Reward:        reward,
		Denom:         denom,
		ClaimPeriodID: claimPeriodID,
	}
}

// String implements fmt.Stringer
func (c Claim) String() string {
	return fmt.Sprintf(`Claim:
	Owner: %s,
	Denom: %s,
	Reward: %s,
	Claim Period ID: %d,
	`, c.Owner, c.Denom, c.Reward, c.ClaimPeriodID)
}

// Claims array of Claim
type Claims []Claim

// NewRewardPeriodFromReward returns a new reward period from the input reward and block time
func NewRewardPeriodFromReward(reward Reward, blockTime time.Time) RewardPeriod {
	// note: reward periods store the amount of rewards paid PER SECOND
	rewardsPerSecond := sdk.NewDecFromInt(reward.AvailableRewards.Amount).Quo(sdk.NewDecFromInt(sdk.NewInt(int64(reward.Duration.Seconds())))).TruncateInt()
	rewardCoinPerSecond := sdk.NewCoin(reward.AvailableRewards.Denom, rewardsPerSecond)
	return RewardPeriod{
		Denom:         reward.Denom,
		Start:         blockTime,
		End:           blockTime.Add(reward.Duration),
		Reward:        rewardCoinPerSecond,
		ClaimEnd:      blockTime.Add(reward.Duration).Add(reward.ClaimDuration),
		ClaimTimeLock: reward.TimeLock,
	}
}
