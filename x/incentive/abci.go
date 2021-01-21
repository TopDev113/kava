package incentive

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/incentive/keeper"
)

// BeginBlocker runs at the start of every block
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	params := k.GetParams(ctx)
	for _, rp := range params.USDXMintingRewardPeriods {
		err := k.AccumulateUSDXMintingRewards(ctx, rp)
		if err != nil {
			panic(err)
		}
	}
	for _, rp := range params.HardSupplyRewardPeriods {
		err := k.AccumulateHardSupplyRewards(ctx, rp)
		if err != nil {
			panic(err)
		}
	}
	for _, rp := range params.HardBorrowRewardPeriods {
		err := k.AccumulateHardBorrowRewards(ctx, rp)
		if err != nil {
			panic(err)
		}
	}
}
