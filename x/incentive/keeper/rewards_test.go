package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/kava-labs/kava/app"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	"github.com/kava-labs/kava/x/hard"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	"github.com/kava-labs/kava/x/incentive/types"
)

func (suite *KeeperTestSuite) TestAccumulateUSDXMintingRewards() {
	type args struct {
		ctype                 string
		rewardsPerSecond      sdk.Coin
		initialTime           time.Time
		initialTotalPrincipal sdk.Coin
		timeElapsed           int
		expectedRewardFactor  sdk.Dec
	}
	type test struct {
		name string
		args args
	}
	testCases := []test{
		{
			"7 seconds",
			args{
				ctype:                 "bnb-a",
				rewardsPerSecond:      c("ukava", 122354),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialTotalPrincipal: c("usdx", 1000000000000),
				timeElapsed:           7,
				expectedRewardFactor:  d("0.000000856478000000"),
			},
		},
		{
			"1 day",
			args{
				ctype:                 "bnb-a",
				rewardsPerSecond:      c("ukava", 122354),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialTotalPrincipal: c("usdx", 1000000000000),
				timeElapsed:           86400,
				expectedRewardFactor:  d("0.0105713856"),
			},
		},
		{
			"0 seconds",
			args{
				ctype:                 "bnb-a",
				rewardsPerSecond:      c("ukava", 122354),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialTotalPrincipal: c("usdx", 1000000000000),
				timeElapsed:           0,
				expectedRewardFactor:  d("0.0"),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// setup cdp state
			cdpKeeper := suite.app.GetCDPKeeper()
			cdpKeeper.SetTotalPrincipal(suite.ctx, tc.args.ctype, cdptypes.DefaultStableDenom, tc.args.initialTotalPrincipal.Amount)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), cs(tc.args.rewardsPerSecond))},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), cs(tc.args.rewardsPerSecond))},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousUSDXMintingAccrualTime(suite.ctx, tc.args.ctype, tc.args.initialTime)
			suite.keeper.SetUSDXMintingRewardFactor(suite.ctx, tc.args.ctype, sdk.ZeroDec())

			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)
			rewardPeriod, found := suite.keeper.GetUSDXMintingRewardPeriod(suite.ctx, tc.args.ctype)
			suite.Require().True(found)
			err := suite.keeper.AccumulateUSDXMintingRewards(suite.ctx, rewardPeriod)
			suite.Require().NoError(err)

			rewardFactor, found := suite.keeper.GetUSDXMintingRewardFactor(suite.ctx, tc.args.ctype)
			suite.Require().Equal(tc.args.expectedRewardFactor, rewardFactor)
		})
	}
}

func (suite *KeeperTestSuite) TestSynchronizeUSDXMintingReward() {
	type args struct {
		ctype                string
		rewardsPerSecond     sdk.Coin
		initialTime          time.Time
		initialCollateral    sdk.Coin
		initialPrincipal     sdk.Coin
		blockTimes           []int
		expectedRewardFactor sdk.Dec
		expectedRewards      sdk.Coin
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"10 blocks",
			args{
				ctype:                "bnb-a",
				rewardsPerSecond:     c("ukava", 122354),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:    c("bnb", 1000000000000),
				initialPrincipal:     c("usdx", 10000000000),
				blockTimes:           []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardFactor: d("0.001223540000000000"),
				expectedRewards:      c("ukava", 12235400),
			},
		},
		{
			"10 blocks - long block time",
			args{
				ctype:                "bnb-a",
				rewardsPerSecond:     c("ukava", 122354),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:    c("bnb", 1000000000000),
				initialPrincipal:     c("usdx", 10000000000),
				blockTimes:           []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardFactor: d("10.57138560000000000"),
				expectedRewards:      c("ukava", 105713856000),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), cs(tc.args.rewardsPerSecond))},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), cs(tc.args.rewardsPerSecond))},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousUSDXMintingAccrualTime(suite.ctx, tc.args.ctype, tc.args.initialTime)
			suite.keeper.SetUSDXMintingRewardFactor(suite.ctx, tc.args.ctype, sdk.ZeroDec())

			// setup account state
			sk := suite.app.GetSupplyKeeper()
			sk.MintCoins(suite.ctx, cdptypes.ModuleName, sdk.NewCoins(tc.args.initialCollateral))
			sk.SendCoinsFromModuleToAccount(suite.ctx, cdptypes.ModuleName, suite.addrs[0], sdk.NewCoins(tc.args.initialCollateral))

			// setup cdp state
			cdpKeeper := suite.app.GetCDPKeeper()
			err := cdpKeeper.AddCdp(suite.ctx, suite.addrs[0], tc.args.initialCollateral, tc.args.initialPrincipal, tc.args.ctype)
			suite.Require().NoError(err)

			claim, found := suite.keeper.GetUSDXMintingClaim(suite.ctx, suite.addrs[0])
			suite.Require().True(found)
			suite.Require().Equal(sdk.ZeroDec(), claim.RewardIndexes[0].RewardFactor)

			var timeElapsed int
			previousBlockTime := suite.ctx.BlockTime()
			for _, t := range tc.args.blockTimes {
				timeElapsed += t
				updatedBlockTime := previousBlockTime.Add(time.Duration(int(time.Second) * t))
				previousBlockTime = updatedBlockTime
				blockCtx := suite.ctx.WithBlockTime(updatedBlockTime)
				rewardPeriod, found := suite.keeper.GetUSDXMintingRewardPeriod(blockCtx, tc.args.ctype)
				suite.Require().True(found)
				err := suite.keeper.AccumulateUSDXMintingRewards(blockCtx, rewardPeriod)
				suite.Require().NoError(err)
			}
			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)
			cdp, found := cdpKeeper.GetCdpByOwnerAndCollateralType(suite.ctx, suite.addrs[0], tc.args.ctype)
			suite.Require().True(found)
			suite.Require().NotPanics(func() {
				suite.keeper.SynchronizeUSDXMintingReward(suite.ctx, cdp)
			})

			rewardFactor, found := suite.keeper.GetUSDXMintingRewardFactor(suite.ctx, tc.args.ctype)
			suite.Require().Equal(tc.args.expectedRewardFactor, rewardFactor)

			claim, found = suite.keeper.GetUSDXMintingClaim(suite.ctx, suite.addrs[0])
			suite.Require().True(found)
			suite.Require().Equal(tc.args.expectedRewardFactor, claim.RewardIndexes[0].RewardFactor)
			suite.Require().Equal(tc.args.expectedRewards, claim.Reward)
		})
	}
}

func (suite *KeeperTestSuite) TestAccumulateHardBorrowRewards() {
	type args struct {
		borrow                sdk.Coin
		rewardsPerSecond      sdk.Coins
		initialTime           time.Time
		timeElapsed           int
		expectedRewardIndexes types.RewardIndexes
	}
	type test struct {
		name string
		args args
	}
	testCases := []test{
		{
			"single reward denom: 7 seconds",
			args{
				borrow:                c("bnb", 1000000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:           7,
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.000000856478000001"))},
			},
		},
		{
			"single reward denom: 1 day",
			args{
				borrow:                c("bnb", 1000000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:           86400,
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.010571385600010177"))},
			},
		},
		{
			"single reward denom: 0 seconds",
			args{
				borrow:                c("bnb", 1000000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:           0,
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.0"))},
			},
		},
		{
			"multiple reward denoms: 7 seconds",
			args{
				borrow:           c("bnb", 1000000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:      7,
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.000000856478000001")),
					types.NewRewardIndex("ukava", d("0.000000856478000001")),
				},
			},
		},
		{
			"multiple reward denoms: 1 day",
			args{
				borrow:           c("bnb", 1000000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:      86400,
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.010571385600010177")),
					types.NewRewardIndex("ukava", d("0.010571385600010177")),
				},
			},
		},
		{
			"multiple reward denoms: 0 seconds",
			args{
				borrow:           c("bnb", 1000000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:      0,
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.0")),
					types.NewRewardIndex("ukava", d("0.0")),
				},
			},
		},
		{
			"multiple reward denoms with different rewards per second: 1 day",
			args{
				borrow:           c("bnb", 1000000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 555555)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:      86400,
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.010571385600010177")),
					types.NewRewardIndex("ukava", d("0.047999952000046210")),
				},
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardBorrowRewardAccrualTime(suite.ctx, tc.args.borrow.Denom, tc.args.initialTime)
			var rewardIndexes types.RewardIndexes
			for _, rewardCoin := range tc.args.rewardsPerSecond {
				rewardIndex := types.NewRewardIndex(rewardCoin.Denom, sdk.ZeroDec())
				rewardIndexes = append(rewardIndexes, rewardIndex)
			}
			suite.keeper.SetHardBorrowRewardIndexes(suite.ctx, tc.args.borrow.Denom, rewardIndexes)

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetSupplyInterestFactor(suite.ctx, tc.args.borrow.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetBorrowInterestFactor(suite.ctx, tc.args.borrow.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.borrow.Denom, tc.args.initialTime)

			// User deposits and borrows to increase total borrowed amount
			hardKeeper := suite.app.GetHardKeeper()
			userAddr := suite.addrs[3]
			err := hardKeeper.Deposit(suite.ctx, userAddr, sdk.NewCoins(sdk.NewCoin(tc.args.borrow.Denom, tc.args.borrow.Amount.Mul(sdk.NewInt(2)))))
			suite.Require().NoError(err)
			err = hardKeeper.Borrow(suite.ctx, userAddr, sdk.NewCoins(tc.args.borrow))
			suite.Require().NoError(err)

			// Set up chain context at future time
			runAtTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.timeElapsed))
			runCtx := suite.ctx.WithBlockTime(runAtTime)

			// Run Hard begin blocker in order to update the denom's index factor
			hard.BeginBlocker(runCtx, suite.hardKeeper)

			// Accumulate hard borrow rewards for the deposit denom
			multiRewardPeriod, found := suite.keeper.GetHardBorrowRewardPeriods(runCtx, tc.args.borrow.Denom)
			suite.Require().True(found)
			err = suite.keeper.AccumulateHardBorrowRewards(runCtx, multiRewardPeriod)
			suite.Require().NoError(err)

			// Check that each expected reward index matches the current stored reward index for the denom
			globalRewardIndexes, found := suite.keeper.GetHardBorrowRewardIndexes(runCtx, tc.args.borrow.Denom)
			suite.Require().True(found)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				globalRewardIndex, found := globalRewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, globalRewardIndex)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSynchronizeHardBorrowReward() {
	type args struct {
		borrow                sdk.Coin
		rewardsPerSecond      sdk.Coins
		initialTime           time.Time
		blockTimes            []int
		expectedRewardIndexes types.RewardIndexes
		expectedRewards       sdk.Coins
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"10 blocks",
			args{
				borrow:                c("bnb", 10000000000), // TODO: 2 decimal diff from TestAccumulateHardBorrowRewards's borrow
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.001223540000173228"))},
				expectedRewards:       cs(c("hard", 12235400)),
			},
		},
		{
			"10 blocks - long block time",
			args{
				borrow:                c("bnb", 10000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("10.571385603126235340"))},
				expectedRewards:       cs(c("hard", 105713856031)),
			},
		},
		{
			"multiple reward denoms: 10 blocks",
			args{
				borrow:           c("bnb", 10000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:       []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.001223540000173228")),
					types.NewRewardIndex("ukava", d("0.001223540000173228")),
				},
				expectedRewards: cs(c("hard", 12235400), c("ukava", 12235400)),
			},
		},
		{
			"multiple reward denoms: 10 blocks - long block time",
			args{
				borrow:           c("bnb", 10000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:       []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("10.571385603126235340")),
					types.NewRewardIndex("ukava", d("10.571385603126235340")),
				},
				expectedRewards: cs(c("hard", 105713856031), c("ukava", 105713856031)),
			},
		},
		{
			"multiple reward denoms with different rewards per second: 10 blocks",
			args{
				borrow:           c("bnb", 10000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 555555)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:       []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.001223540000173228")),
					types.NewRewardIndex("ukava", d("0.005555550000786558")),
				},
				expectedRewards: cs(c("hard", 12235400), c("ukava", 55555500)),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardBorrowRewardAccrualTime(suite.ctx, tc.args.borrow.Denom, tc.args.initialTime)
			var rewardIndexes types.RewardIndexes
			for _, rewardCoin := range tc.args.rewardsPerSecond {
				rewardIndex := types.NewRewardIndex(rewardCoin.Denom, sdk.ZeroDec())
				rewardIndexes = append(rewardIndexes, rewardIndex)
			}
			suite.keeper.SetHardBorrowRewardIndexes(suite.ctx, tc.args.borrow.Denom, rewardIndexes)

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetSupplyInterestFactor(suite.ctx, tc.args.borrow.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetBorrowInterestFactor(suite.ctx, tc.args.borrow.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.borrow.Denom, tc.args.initialTime)

			// User deposits and borrows to increase total borrowed amount
			hardKeeper := suite.app.GetHardKeeper()
			userAddr := suite.addrs[3]
			err := hardKeeper.Deposit(suite.ctx, userAddr, sdk.NewCoins(sdk.NewCoin(tc.args.borrow.Denom, tc.args.borrow.Amount.Mul(sdk.NewInt(2)))))
			suite.Require().NoError(err)
			err = hardKeeper.Borrow(suite.ctx, userAddr, sdk.NewCoins(tc.args.borrow))
			suite.Require().NoError(err)

			// Check that Hard hooks initialized a HardLiquidityProviderClaim
			claim, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			multiRewardIndex, _ := claim.BorrowRewardIndexes.GetRewardIndex(tc.args.borrow.Denom)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				currRewardIndex, found := multiRewardIndex.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(sdk.ZeroDec(), currRewardIndex.RewardFactor)
			}

			// Run accumulator at several intervals
			var timeElapsed int
			previousBlockTime := suite.ctx.BlockTime()
			for _, t := range tc.args.blockTimes {
				timeElapsed += t
				updatedBlockTime := previousBlockTime.Add(time.Duration(int(time.Second) * t))
				previousBlockTime = updatedBlockTime
				blockCtx := suite.ctx.WithBlockTime(updatedBlockTime)

				// Run Hard begin blocker for each block ctx to update denom's interest factor
				hard.BeginBlocker(blockCtx, suite.hardKeeper)

				// Accumulate hard borrow-side rewards
				multiRewardPeriod, found := suite.keeper.GetHardBorrowRewardPeriods(blockCtx, tc.args.borrow.Denom)
				suite.Require().True(found)
				err := suite.keeper.AccumulateHardBorrowRewards(blockCtx, multiRewardPeriod)
				suite.Require().NoError(err)
			}
			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)

			// After we've accumulated, run synchronize
			borrow, found := hardKeeper.GetBorrow(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			suite.Require().NotPanics(func() {
				suite.keeper.SynchronizeHardBorrowReward(suite.ctx, borrow)
			})

			// Check that the global reward index's reward factor and user's claim have been updated as expected
			globalRewardIndexes, found := suite.keeper.GetHardBorrowRewardIndexes(suite.ctx, tc.args.borrow.Denom)
			suite.Require().True(found)
			claim, found = suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				// Check that global reward index has been updated as expected
				globalRewardIndex, found := globalRewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, globalRewardIndex)

				// Check that the user's claim's reward index matches the corresponding global reward index
				multiRewardIndex, found := claim.BorrowRewardIndexes.GetRewardIndex(tc.args.borrow.Denom)
				suite.Require().True(found)
				rewardIndex, found := multiRewardIndex.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, rewardIndex)

				// Check that the user's claim holds the expected amount of reward coins
				suite.Require().Equal(
					tc.args.expectedRewards.AmountOf(expectedRewardIndex.CollateralType),
					claim.Reward.AmountOf(expectedRewardIndex.CollateralType),
				)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestAccumulateHardDelegatorRewards() {
	type args struct {
		delegation           sdk.Coin
		rewardsPerSecond     sdk.Coin
		initialTime          time.Time
		timeElapsed          int
		expectedRewardFactor sdk.Dec
	}
	type test struct {
		name string
		args args
	}
	testCases := []test{
		{
			"7 seconds",
			args{
				delegation:           c("ukava", 1_000_000),
				rewardsPerSecond:     c("hard", 122354),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:          7,
				expectedRewardFactor: d("0.428239000000000000"),
			},
		},
		{
			"1 day",
			args{
				delegation:           c("ukava", 1_000_000),
				rewardsPerSecond:     c("hard", 122354),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:          86400,
				expectedRewardFactor: d("5285.692800000000000000"),
			},
		},
		{
			"0 seconds",
			args{
				delegation:           c("ukava", 1_000_000),
				rewardsPerSecond:     c("hard", 122354),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:          0,
				expectedRewardFactor: d("0.0"),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// Set up incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), cs(tc.args.rewardsPerSecond))},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), cs(tc.args.rewardsPerSecond))},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardDelegatorRewardAccrualTime(suite.ctx, tc.args.delegation.Denom, tc.args.initialTime)
			suite.keeper.SetHardDelegatorRewardFactor(suite.ctx, tc.args.delegation.Denom, sdk.ZeroDec())

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.delegation.Denom, tc.args.initialTime)

			err := suite.deliverMsgCreateValidator(suite.ctx, suite.validatorAddrs[0], tc.args.delegation)
			suite.Require().NoError(err)
			suite.deliverMsgDelegate(suite.ctx, suite.addrs[0], suite.validatorAddrs[0], tc.args.delegation)
			suite.Require().NoError(err)

			staking.EndBlocker(suite.ctx, suite.stakingKeeper)

			// Set up chain context at future time
			runAtTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.timeElapsed))
			runCtx := suite.ctx.WithBlockTime(runAtTime)

			// Run Hard begin blocker in order to update the denom's index factor
			hard.BeginBlocker(runCtx, suite.hardKeeper)

			rewardPeriod, found := suite.keeper.GetHardDelegatorRewardPeriod(runCtx, tc.args.delegation.Denom)
			suite.Require().True(found)
			err = suite.keeper.AccumulateHardDelegatorRewards(runCtx, rewardPeriod)
			suite.Require().NoError(err)

			rewardFactor, found := suite.keeper.GetHardDelegatorRewardFactor(runCtx, tc.args.delegation.Denom)
			suite.Require().Equal(tc.args.expectedRewardFactor, rewardFactor)
		})
	}
}

func (suite *KeeperTestSuite) TestAccumulateHardSupplyRewards() {
	type args struct {
		deposit               sdk.Coin
		rewardsPerSecond      sdk.Coins
		initialTime           time.Time
		timeElapsed           int
		expectedRewardIndexes types.RewardIndexes
	}
	type test struct {
		name string
		args args
	}
	testCases := []test{
		{
			"single reward denom: 7 seconds",
			args{
				deposit:               c("bnb", 1000000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:           7,
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.000000856478000000"))},
			},
		},
		{
			"single reward denom: 1 day",
			args{
				deposit:               c("bnb", 1000000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:           86400,
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.010571385600000000"))},
			},
		},
		{
			"single reward denom: 0 seconds",
			args{
				deposit:               c("bnb", 1000000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:           0,
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.0"))},
			},
		},
		{
			"multiple reward denoms: 7 seconds",
			args{
				deposit:          c("bnb", 1000000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:      7,
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.000000856478000000")),
					types.NewRewardIndex("ukava", d("0.000000856478000000")),
				},
			},
		},
		{
			"multiple reward denoms: 1 day",
			args{
				deposit:          c("bnb", 1000000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:      86400,
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.010571385600000000")),
					types.NewRewardIndex("ukava", d("0.010571385600000000")),
				},
			},
		},
		{
			"multiple reward denoms: 0 seconds",
			args{
				deposit:          c("bnb", 1000000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:      0,
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.0")),
					types.NewRewardIndex("ukava", d("0.0")),
				},
			},
		},
		{
			"multiple reward denoms with different rewards per second: 1 day",
			args{
				deposit:          c("bnb", 1000000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 555555)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:      86400,
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.010571385600000000")),
					types.NewRewardIndex("ukava", d("0.047999952000000000")),
				},
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// Set up incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardSupplyRewardAccrualTime(suite.ctx, tc.args.deposit.Denom, tc.args.initialTime)
			var rewardIndexes types.RewardIndexes
			for _, rewardCoin := range tc.args.rewardsPerSecond {
				rewardIndex := types.NewRewardIndex(rewardCoin.Denom, sdk.ZeroDec())
				rewardIndexes = append(rewardIndexes, rewardIndex)
			}
			suite.keeper.SetHardSupplyRewardIndexes(suite.ctx, tc.args.deposit.Denom, rewardIndexes)

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetSupplyInterestFactor(suite.ctx, tc.args.deposit.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.deposit.Denom, tc.args.initialTime)

			// User deposits to increase total supplied amount
			hardKeeper := suite.app.GetHardKeeper()
			userAddr := suite.addrs[3]
			err := hardKeeper.Deposit(suite.ctx, userAddr, sdk.NewCoins(tc.args.deposit))
			suite.Require().NoError(err)

			// Set up chain context at future time
			runAtTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.timeElapsed))
			runCtx := suite.ctx.WithBlockTime(runAtTime)

			// Run Hard begin blocker in order to update the denom's index factor
			hard.BeginBlocker(runCtx, suite.hardKeeper)

			// Accumulate hard supply rewards for the deposit denom
			multiRewardPeriod, found := suite.keeper.GetHardSupplyRewardPeriods(runCtx, tc.args.deposit.Denom)
			suite.Require().True(found)
			err = suite.keeper.AccumulateHardSupplyRewards(runCtx, multiRewardPeriod)
			suite.Require().NoError(err)

			// Check that each expected reward index matches the current stored reward index for theh denom
			globalRewardIndexes, found := suite.keeper.GetHardSupplyRewardIndexes(runCtx, tc.args.deposit.Denom)
			suite.Require().True(found)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				globalRewardIndex, found := globalRewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, globalRewardIndex)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSynchronizeHardSupplyReward() {
	type args struct {
		deposit               sdk.Coin
		rewardsPerSecond      sdk.Coins
		initialTime           time.Time
		blockTimes            []int
		expectedRewardIndexes types.RewardIndexes
		expectedRewards       sdk.Coins
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"single reward denom: 10 blocks",
			args{
				deposit:               c("bnb", 10000000000), // TODO: 2 decimal diff
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.001223540000000000"))},
				expectedRewards:       cs(c("hard", 12235400)),
			},
		},
		{
			"single reward denom: 10 blocks - long block time",
			args{
				deposit:               c("bnb", 10000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("10.571385600000000000"))},
				expectedRewards:       cs(c("hard", 105713856000)),
			},
		},
		{
			"multiple reward denoms: 10 blocks",
			args{
				deposit:          c("bnb", 10000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:       []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.001223540000000000")),
					types.NewRewardIndex("ukava", d("0.001223540000000000")),
				},
				expectedRewards: cs(c("hard", 12235400), c("ukava", 12235400)),
			},
		},
		{
			"multiple reward denoms: 10 blocks - long block time",
			args{
				deposit:          c("bnb", 10000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:       []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("10.571385600000000000")),
					types.NewRewardIndex("ukava", d("10.571385600000000000")),
				},
				expectedRewards: cs(c("hard", 105713856000), c("ukava", 105713856000)),
			},
		},
		{
			"multiple reward denoms with different rewards per second: 10 blocks",
			args{
				deposit:          c("bnb", 10000000000),
				rewardsPerSecond: cs(c("hard", 122354), c("ukava", 555555)),
				initialTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:       []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{
					types.NewRewardIndex("hard", d("0.001223540000000000")),
					types.NewRewardIndex("ukava", d("0.005555550000000000")),
				},
				expectedRewards: cs(c("hard", 12235400), c("ukava", 55555500)),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardSupplyRewardAccrualTime(suite.ctx, tc.args.deposit.Denom, tc.args.initialTime)
			var rewardIndexes types.RewardIndexes
			for _, rewardCoin := range tc.args.rewardsPerSecond {
				rewardIndex := types.NewRewardIndex(rewardCoin.Denom, sdk.ZeroDec())
				rewardIndexes = append(rewardIndexes, rewardIndex)
			}
			suite.keeper.SetHardSupplyRewardIndexes(suite.ctx, tc.args.deposit.Denom, rewardIndexes)

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetSupplyInterestFactor(suite.ctx, tc.args.deposit.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetBorrowInterestFactor(suite.ctx, tc.args.deposit.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.deposit.Denom, tc.args.initialTime)

			// User deposits and borrows to increase total borrowed amount
			hardKeeper := suite.app.GetHardKeeper()
			userAddr := suite.addrs[3]
			err := hardKeeper.Deposit(suite.ctx, userAddr, sdk.NewCoins(tc.args.deposit))
			suite.Require().NoError(err)

			// Check that Hard hooks initialized a HardLiquidityProviderClaim with 0 reward indexes
			claim, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			multiRewardIndex, _ := claim.SupplyRewardIndexes.GetRewardIndex(tc.args.deposit.Denom)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				currRewardIndex, found := multiRewardIndex.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(sdk.ZeroDec(), currRewardIndex.RewardFactor)
			}

			// Run accumulator at several intervals
			var timeElapsed int
			previousBlockTime := suite.ctx.BlockTime()
			for _, t := range tc.args.blockTimes {
				timeElapsed += t
				updatedBlockTime := previousBlockTime.Add(time.Duration(int(time.Second) * t))
				previousBlockTime = updatedBlockTime
				blockCtx := suite.ctx.WithBlockTime(updatedBlockTime)

				// Run Hard begin blocker for each block ctx to update denom's interest factor
				hard.BeginBlocker(blockCtx, suite.hardKeeper)

				// Accumulate hard supply-side rewards
				multiRewardPeriod, found := suite.keeper.GetHardSupplyRewardPeriods(blockCtx, tc.args.deposit.Denom)
				suite.Require().True(found)
				err := suite.keeper.AccumulateHardSupplyRewards(blockCtx, multiRewardPeriod)
				suite.Require().NoError(err)
			}
			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)

			// After we've accumulated, run synchronize
			deposit, found := hardKeeper.GetDeposit(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			suite.Require().NotPanics(func() {
				suite.keeper.SynchronizeHardSupplyReward(suite.ctx, deposit)
			})

			// Check that the global reward index's reward factor and user's claim have been updated as expected
			globalRewardIndexes, found := suite.keeper.GetHardSupplyRewardIndexes(suite.ctx, tc.args.deposit.Denom)
			suite.Require().True(found)
			claim, found = suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				// Check that global reward index has been updated as expected
				globalRewardIndex, found := globalRewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, globalRewardIndex)

				// Check that the user's claim's reward index matches the corresponding global reward index
				multiRewardIndex, found := claim.SupplyRewardIndexes.GetRewardIndex(tc.args.deposit.Denom)
				suite.Require().True(found)
				rewardIndex, found := multiRewardIndex.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, rewardIndex)

				// Check that the user's claim holds the expected amount of reward coins
				suite.Require().Equal(
					tc.args.expectedRewards.AmountOf(expectedRewardIndex.CollateralType),
					claim.Reward.AmountOf(expectedRewardIndex.CollateralType),
				)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateHardSupplyIndexDenoms() {
	type args struct {
		firstDeposit              sdk.Coins
		secondDeposit             sdk.Coins
		rewardsPerSecond          sdk.Coins
		initialTime               time.Time
		expectedSupplyIndexDenoms []string
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"single reward denom: update adds one supply reward index",
			args{
				firstDeposit:              cs(c("bnb", 10000000000)),
				secondDeposit:             cs(c("ukava", 10000000000)),
				rewardsPerSecond:          cs(c("hard", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedSupplyIndexDenoms: []string{"bnb", "ukava"},
			},
		},
		{
			"single reward denom: update adds multiple supply reward indexes",
			args{
				firstDeposit:              cs(c("bnb", 10000000000)),
				secondDeposit:             cs(c("ukava", 10000000000), c("btcb", 10000000000), c("xrp", 10000000000)),
				rewardsPerSecond:          cs(c("hard", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedSupplyIndexDenoms: []string{"bnb", "ukava", "btcb", "xrp"},
			},
		},
		{
			"single reward denom: update doesn't add duplicate supply reward index for same denom",
			args{
				firstDeposit:              cs(c("bnb", 10000000000)),
				secondDeposit:             cs(c("bnb", 5000000000)),
				rewardsPerSecond:          cs(c("hard", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedSupplyIndexDenoms: []string{"bnb"},
			},
		},
		{
			"multiple reward denoms: update adds one supply reward index",
			args{
				firstDeposit:              cs(c("bnb", 10000000000)),
				secondDeposit:             cs(c("ukava", 10000000000)),
				rewardsPerSecond:          cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedSupplyIndexDenoms: []string{"bnb", "ukava"},
			},
		},
		{
			"multiple reward denoms: update adds multiple supply reward indexes",
			args{
				firstDeposit:              cs(c("bnb", 10000000000)),
				secondDeposit:             cs(c("ukava", 10000000000), c("btcb", 10000000000), c("xrp", 10000000000)),
				rewardsPerSecond:          cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedSupplyIndexDenoms: []string{"bnb", "ukava", "btcb", "xrp"},
			},
		},
		{
			"multiple reward denoms: update doesn't add duplicate supply reward index for same denom",
			args{
				firstDeposit:              cs(c("bnb", 10000000000)),
				secondDeposit:             cs(c("bnb", 5000000000)),
				rewardsPerSecond:          cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedSupplyIndexDenoms: []string{"bnb"},
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// Set up generic reward periods
			var multiRewardPeriods types.MultiRewardPeriods
			var rewardPeriods types.RewardPeriods
			for i, denom := range tc.args.expectedSupplyIndexDenoms {
				// Create just one reward period for USDX Minting / Hard Delegator reward periods (otherwise params will panic on duplicate)
				if i == 0 {
					rewardPeriod := types.NewRewardPeriod(true, denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[i])
					rewardPeriods = append(rewardPeriods, rewardPeriod)
				}
				multiRewardPeriod := types.NewMultiRewardPeriod(true, denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)
				multiRewardPeriods = append(multiRewardPeriods, multiRewardPeriod)
			}

			// Setup incentive state
			params := types.NewParams(
				rewardPeriods, multiRewardPeriods, multiRewardPeriods, rewardPeriods,
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)

			// Set each denom's previous accrual time and supply reward factor
			var rewardIndexes types.RewardIndexes
			for _, rewardCoin := range tc.args.rewardsPerSecond {
				rewardIndex := types.NewRewardIndex(rewardCoin.Denom, sdk.ZeroDec())
				rewardIndexes = append(rewardIndexes, rewardIndex)
			}
			for _, denom := range tc.args.expectedSupplyIndexDenoms {
				suite.keeper.SetPreviousHardSupplyRewardAccrualTime(suite.ctx, denom, tc.args.initialTime)
				suite.keeper.SetHardSupplyRewardIndexes(suite.ctx, denom, rewardIndexes)
			}

			// User deposits (first time)
			hardKeeper := suite.app.GetHardKeeper()
			userAddr := suite.addrs[3]
			err := hardKeeper.Deposit(suite.ctx, userAddr, tc.args.firstDeposit)
			suite.Require().NoError(err)

			// Confirm that a claim was created and populated with the correct supply indexes
			claimAfterFirstDeposit, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			for _, coin := range tc.args.firstDeposit {
				_, hasIndex := claimAfterFirstDeposit.HasSupplyRewardIndex(coin.Denom)
				suite.Require().True(hasIndex)
			}
			suite.Require().True(len(claimAfterFirstDeposit.SupplyRewardIndexes) == len(tc.args.firstDeposit))

			// User deposits (second time)
			err = hardKeeper.Deposit(suite.ctx, userAddr, tc.args.secondDeposit)
			suite.Require().NoError(err)

			// Confirm that the claim contains all expected supply indexes
			claimAfterSecondDeposit, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			for _, denom := range tc.args.expectedSupplyIndexDenoms {
				_, hasIndex := claimAfterSecondDeposit.HasSupplyRewardIndex(denom)
				suite.Require().True(hasIndex)
			}
			suite.Require().True(len(claimAfterSecondDeposit.SupplyRewardIndexes) == len(tc.args.expectedSupplyIndexDenoms))
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateHardBorrowIndexDenoms() {
	type args struct {
		initialDeposit            sdk.Coins
		firstBorrow               sdk.Coins
		secondBorrow              sdk.Coins
		rewardsPerSecond          sdk.Coins
		initialTime               time.Time
		expectedBorrowIndexDenoms []string
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"single reward denom: update adds one borrow reward index",
			args{
				initialDeposit:            cs(c("bnb", 10000000000)),
				firstBorrow:               cs(c("bnb", 50000000)),
				secondBorrow:              cs(c("ukava", 500000000)),
				rewardsPerSecond:          cs(c("hard", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedBorrowIndexDenoms: []string{"bnb", "ukava"},
			},
		},
		{
			"single reward denom: update adds multiple borrow supply reward indexes",
			args{
				initialDeposit:            cs(c("btcb", 10000000000)),
				firstBorrow:               cs(c("btcb", 50000000)),
				secondBorrow:              cs(c("ukava", 500000000), c("bnb", 50000000000), c("xrp", 50000000000)),
				rewardsPerSecond:          cs(c("hard", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedBorrowIndexDenoms: []string{"btcb", "ukava", "bnb", "xrp"},
			},
		},
		{
			"single reward denom: update doesn't add duplicate borrow reward index for same denom",
			args{
				initialDeposit:            cs(c("bnb", 100000000000)),
				firstBorrow:               cs(c("bnb", 50000000)),
				secondBorrow:              cs(c("bnb", 50000000000)),
				rewardsPerSecond:          cs(c("hard", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedBorrowIndexDenoms: []string{"bnb"},
			},
		},
		{
			"multiple reward denoms: update adds one borrow reward index",
			args{
				initialDeposit:            cs(c("bnb", 10000000000)),
				firstBorrow:               cs(c("bnb", 50000000)),
				secondBorrow:              cs(c("ukava", 500000000)),
				rewardsPerSecond:          cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedBorrowIndexDenoms: []string{"bnb", "ukava"},
			},
		},
		{
			"multiple reward denoms: update adds multiple borrow supply reward indexes",
			args{
				initialDeposit:            cs(c("btcb", 10000000000)),
				firstBorrow:               cs(c("btcb", 50000000)),
				secondBorrow:              cs(c("ukava", 500000000), c("bnb", 50000000000), c("xrp", 50000000000)),
				rewardsPerSecond:          cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedBorrowIndexDenoms: []string{"btcb", "ukava", "bnb", "xrp"},
			},
		},
		{
			"multiple reward denoms: update doesn't add duplicate borrow reward index for same denom",
			args{
				initialDeposit:            cs(c("bnb", 100000000000)),
				firstBorrow:               cs(c("bnb", 50000000)),
				secondBorrow:              cs(c("bnb", 50000000000)),
				rewardsPerSecond:          cs(c("hard", 122354), c("ukava", 122354)),
				initialTime:               time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				expectedBorrowIndexDenoms: []string{"bnb"},
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account so it can service borrow requests
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := tc.args.firstBorrow.Add(tc.args.secondBorrow...)
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// Set up generic reward periods
			var multiRewardPeriods types.MultiRewardPeriods
			var rewardPeriods types.RewardPeriods
			for i, denom := range tc.args.expectedBorrowIndexDenoms {
				// Create just one reward period for USDX Minting / Hard Delegator reward periods (otherwise params will panic on duplicate)
				if i == 0 {
					rewardPeriod := types.NewRewardPeriod(true, denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[i])
					rewardPeriods = append(rewardPeriods, rewardPeriod)
				}
				multiRewardPeriod := types.NewMultiRewardPeriod(true, denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)
				multiRewardPeriods = append(multiRewardPeriods, multiRewardPeriod)
			}

			// Setup incentive state
			params := types.NewParams(
				rewardPeriods, multiRewardPeriods, multiRewardPeriods, rewardPeriods,
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)

			// Set each expected borrow denom's previous accrual time and borrow reward factor
			var rewardIndexes types.RewardIndexes
			for _, rewardCoin := range tc.args.rewardsPerSecond {
				rewardIndex := types.NewRewardIndex(rewardCoin.Denom, sdk.ZeroDec())
				rewardIndexes = append(rewardIndexes, rewardIndex)
			}
			for _, denom := range tc.args.expectedBorrowIndexDenoms {
				suite.keeper.SetPreviousHardSupplyRewardAccrualTime(suite.ctx, denom, tc.args.initialTime)
				suite.keeper.SetHardBorrowRewardIndexes(suite.ctx, denom, rewardIndexes)
			}

			// User deposits initial funds (so that user can borrow)
			hardKeeper := suite.app.GetHardKeeper()
			userAddr := suite.addrs[3]
			err := hardKeeper.Deposit(suite.ctx, userAddr, tc.args.initialDeposit)
			suite.Require().NoError(err)

			// Confirm that claim exists but no borrow reward indexes have been added
			claimAfterDeposit, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			suite.Require().Equal(0, len(claimAfterDeposit.BorrowRewardIndexes))

			// User borrows (first time)
			err = hardKeeper.Borrow(suite.ctx, userAddr, tc.args.firstBorrow)
			suite.Require().NoError(err)

			// Confirm that claim's borrow reward indexes have been updated
			claimAfterFirstBorrow, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			for _, coin := range tc.args.firstBorrow {
				_, hasIndex := claimAfterFirstBorrow.HasBorrowRewardIndex(coin.Denom)
				suite.Require().True(hasIndex)
			}
			suite.Require().True(len(claimAfterFirstBorrow.BorrowRewardIndexes) == len(tc.args.firstBorrow))

			// User borrows (second time)
			err = hardKeeper.Borrow(suite.ctx, userAddr, tc.args.secondBorrow)
			suite.Require().NoError(err)

			// Confirm that claim's borrow reward indexes contain expected values
			claimAfterSecondBorrow, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			for _, coin := range tc.args.secondBorrow {
				_, hasIndex := claimAfterSecondBorrow.HasBorrowRewardIndex(coin.Denom)
				suite.Require().True(hasIndex)
			}
			suite.Require().True(len(claimAfterSecondBorrow.BorrowRewardIndexes) == len(tc.args.expectedBorrowIndexDenoms))
		})
	}
}

func (suite *KeeperTestSuite) TestSynchronizeHardDelegatorReward() {
	type args struct {
		delegation           sdk.Coin
		rewardsPerSecond     sdk.Coin
		initialTime          time.Time
		blockTimes           []int
		expectedRewardFactor sdk.Dec
		expectedRewards      sdk.Coins
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"10 blocks",
			args{
				delegation:           c("ukava", 1_000_000),
				rewardsPerSecond:     c("hard", 122354),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:           []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardFactor: d("6.117700000000000000"),
				expectedRewards:      cs(c("hard", 6117700)),
			},
		},
		{
			"10 blocks - long block time",
			args{
				delegation:           c("ukava", 1_000_000),
				rewardsPerSecond:     c("hard", 122354),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:           []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardFactor: d("52856.928000000000000000"),
				expectedRewards:      cs(c("hard", 52856928000)),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), cs(tc.args.rewardsPerSecond))},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), cs(tc.args.rewardsPerSecond))},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardDelegatorRewardAccrualTime(suite.ctx, tc.args.delegation.Denom, tc.args.initialTime)
			suite.keeper.SetHardDelegatorRewardFactor(suite.ctx, tc.args.delegation.Denom, sdk.ZeroDec())

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.delegation.Denom, tc.args.initialTime)

			// Delegator delegates
			err := suite.deliverMsgCreateValidator(suite.ctx, suite.validatorAddrs[0], tc.args.delegation)
			suite.Require().NoError(err)
			suite.deliverMsgDelegate(suite.ctx, suite.addrs[0], suite.validatorAddrs[0], tc.args.delegation)
			suite.Require().NoError(err)

			staking.EndBlocker(suite.ctx, suite.stakingKeeper)

			// Check that Staking hooks initialized a HardLiquidityProviderClaim
			claim, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[0])
			suite.Require().True(found)
			suite.Require().Equal(sdk.ZeroDec(), claim.DelegatorRewardIndexes[0].RewardFactor)

			// Run accumulator at several intervals
			var timeElapsed int
			previousBlockTime := suite.ctx.BlockTime()
			for _, t := range tc.args.blockTimes {
				timeElapsed += t
				updatedBlockTime := previousBlockTime.Add(time.Duration(int(time.Second) * t))
				previousBlockTime = updatedBlockTime
				blockCtx := suite.ctx.WithBlockTime(updatedBlockTime)

				// Run Hard begin blocker for each block ctx to update denom's interest factor
				hard.BeginBlocker(blockCtx, suite.hardKeeper)

				rewardPeriod, found := suite.keeper.GetHardDelegatorRewardPeriod(blockCtx, tc.args.delegation.Denom)
				suite.Require().True(found)

				err := suite.keeper.AccumulateHardDelegatorRewards(blockCtx, rewardPeriod)
				suite.Require().NoError(err)
			}
			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)

			// After we've accumulated, run synchronize
			suite.Require().NotPanics(func() {
				suite.keeper.SynchronizeHardDelegatorRewards(suite.ctx, suite.addrs[0])
			})

			// Check that reward factor and claim have been updated as expected
			rewardFactor, found := suite.keeper.GetHardDelegatorRewardFactor(suite.ctx, tc.args.delegation.Denom)
			suite.Require().Equal(tc.args.expectedRewardFactor, rewardFactor)

			claim, found = suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[0])
			suite.Require().True(found)
			suite.Require().Equal(tc.args.expectedRewardFactor, claim.DelegatorRewardIndexes[0].RewardFactor)
			suite.Require().Equal(tc.args.expectedRewards, claim.Reward)
		})
	}
}

func (suite *KeeperTestSuite) TestSimulateHardSupplyRewardSynchronization() {
	type args struct {
		deposit               sdk.Coin
		rewardsPerSecond      sdk.Coins
		initialTime           time.Time
		blockTimes            []int
		expectedRewardIndexes types.RewardIndexes
		expectedRewards       sdk.Coins
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"10 blocks",
			args{
				deposit:               c("bnb", 10000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.001223540000000000"))},
				expectedRewards:       cs(c("hard", 12235400)),
			},
		},
		{
			"10 blocks - long block time",
			args{
				deposit:               c("bnb", 10000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("10.571385600000000000"))},
				expectedRewards:       cs(c("hard", 105713856000)),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// Set up incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.deposit.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardSupplyRewardAccrualTime(suite.ctx, tc.args.deposit.Denom, tc.args.initialTime)
			var rewardIndexes types.RewardIndexes
			for _, rewardCoin := range tc.args.rewardsPerSecond {
				rewardIndex := types.NewRewardIndex(rewardCoin.Denom, sdk.ZeroDec())
				rewardIndexes = append(rewardIndexes, rewardIndex)
			}
			suite.keeper.SetHardSupplyRewardIndexes(suite.ctx, tc.args.deposit.Denom, rewardIndexes)

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetSupplyInterestFactor(suite.ctx, tc.args.deposit.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.deposit.Denom, tc.args.initialTime)

			// User deposits and borrows to increase total borrowed amount
			hardKeeper := suite.app.GetHardKeeper()
			userAddr := suite.addrs[3]
			err := hardKeeper.Deposit(suite.ctx, userAddr, sdk.NewCoins(tc.args.deposit))
			suite.Require().NoError(err)

			// Check that Hard hooks initialized a HardLiquidityProviderClaim
			claim, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			multiRewardIndex, _ := claim.SupplyRewardIndexes.GetRewardIndex(tc.args.deposit.Denom)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				currRewardIndex, found := multiRewardIndex.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(sdk.ZeroDec(), currRewardIndex.RewardFactor)
			}

			// Run accumulator at several intervals
			var timeElapsed int
			previousBlockTime := suite.ctx.BlockTime()
			for _, t := range tc.args.blockTimes {
				timeElapsed += t
				updatedBlockTime := previousBlockTime.Add(time.Duration(int(time.Second) * t))
				previousBlockTime = updatedBlockTime
				blockCtx := suite.ctx.WithBlockTime(updatedBlockTime)

				// Run Hard begin blocker for each block ctx to update denom's interest factor
				hard.BeginBlocker(blockCtx, suite.hardKeeper)

				// Accumulate hard supply-side rewards
				multiRewardPeriod, found := suite.keeper.GetHardSupplyRewardPeriods(blockCtx, tc.args.deposit.Denom)
				suite.Require().True(found)
				err := suite.keeper.AccumulateHardSupplyRewards(blockCtx, multiRewardPeriod)
				suite.Require().NoError(err)
			}
			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)

			// Confirm that the user's claim hasn't been synced
			claimPre, foundPre := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(foundPre)
			multiRewardIndexPre, _ := claimPre.SupplyRewardIndexes.GetRewardIndex(tc.args.deposit.Denom)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				currRewardIndex, found := multiRewardIndexPre.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(sdk.ZeroDec(), currRewardIndex.RewardFactor)
			}

			// Check that the synced claim held in memory has properly simulated syncing
			syncedClaim := suite.keeper.SimulateHardSynchronization(suite.ctx, claimPre)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				// Check that the user's claim's reward index matches the expected reward index
				multiRewardIndex, found := syncedClaim.SupplyRewardIndexes.GetRewardIndex(tc.args.deposit.Denom)
				suite.Require().True(found)
				rewardIndex, found := multiRewardIndex.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, rewardIndex)

				// Check that the user's claim holds the expected amount of reward coins
				suite.Require().Equal(
					tc.args.expectedRewards.AmountOf(expectedRewardIndex.CollateralType),
					syncedClaim.Reward.AmountOf(expectedRewardIndex.CollateralType),
				)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSimulateHardBorrowRewardSynchronization() {
	type args struct {
		borrow                sdk.Coin
		rewardsPerSecond      sdk.Coins
		initialTime           time.Time
		blockTimes            []int
		expectedRewardIndexes types.RewardIndexes
		expectedRewards       sdk.Coins
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"10 blocks",
			args{
				borrow:                c("bnb", 10000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("0.001223540000173228"))},
				expectedRewards:       cs(c("hard", 12235400)),
			},
		},
		{
			"10 blocks - long block time",
			args{
				borrow:                c("bnb", 10000000000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("hard", d("10.571385603126235340"))},
				expectedRewards:       cs(c("hard", 105713856031)),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.borrow.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardBorrowRewardAccrualTime(suite.ctx, tc.args.borrow.Denom, tc.args.initialTime)
			var rewardIndexes types.RewardIndexes
			for _, rewardCoin := range tc.args.rewardsPerSecond {
				rewardIndex := types.NewRewardIndex(rewardCoin.Denom, sdk.ZeroDec())
				rewardIndexes = append(rewardIndexes, rewardIndex)
			}
			suite.keeper.SetHardBorrowRewardIndexes(suite.ctx, tc.args.borrow.Denom, rewardIndexes)

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetSupplyInterestFactor(suite.ctx, tc.args.borrow.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetBorrowInterestFactor(suite.ctx, tc.args.borrow.Denom, sdk.MustNewDecFromStr("1.0"))
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.borrow.Denom, tc.args.initialTime)

			// User deposits and borrows to increase total borrowed amount
			hardKeeper := suite.app.GetHardKeeper()
			userAddr := suite.addrs[3]
			err := hardKeeper.Deposit(suite.ctx, userAddr, sdk.NewCoins(sdk.NewCoin(tc.args.borrow.Denom, tc.args.borrow.Amount.Mul(sdk.NewInt(2)))))
			suite.Require().NoError(err)
			err = hardKeeper.Borrow(suite.ctx, userAddr, sdk.NewCoins(tc.args.borrow))
			suite.Require().NoError(err)

			// Check that Hard hooks initialized a HardLiquidityProviderClaim
			claim, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(found)
			multiRewardIndex, _ := claim.BorrowRewardIndexes.GetRewardIndex(tc.args.borrow.Denom)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				currRewardIndex, found := multiRewardIndex.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(sdk.ZeroDec(), currRewardIndex.RewardFactor)
			}

			// Run accumulator at several intervals
			var timeElapsed int
			previousBlockTime := suite.ctx.BlockTime()
			for _, t := range tc.args.blockTimes {
				timeElapsed += t
				updatedBlockTime := previousBlockTime.Add(time.Duration(int(time.Second) * t))
				previousBlockTime = updatedBlockTime
				blockCtx := suite.ctx.WithBlockTime(updatedBlockTime)

				// Run Hard begin blocker for each block ctx to update denom's interest factor
				hard.BeginBlocker(blockCtx, suite.hardKeeper)

				// Accumulate hard borrow-side rewards
				multiRewardPeriod, found := suite.keeper.GetHardBorrowRewardPeriods(blockCtx, tc.args.borrow.Denom)
				suite.Require().True(found)
				err := suite.keeper.AccumulateHardBorrowRewards(blockCtx, multiRewardPeriod)
				suite.Require().NoError(err)
			}
			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)

			// Confirm that the user's claim hasn't been synced
			claimPre, foundPre := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[3])
			suite.Require().True(foundPre)
			multiRewardIndexPre, _ := claimPre.BorrowRewardIndexes.GetRewardIndex(tc.args.borrow.Denom)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				currRewardIndex, found := multiRewardIndexPre.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(sdk.ZeroDec(), currRewardIndex.RewardFactor)
			}

			// Check that the synced claim held in memory has properly simulated syncing
			syncedClaim := suite.keeper.SimulateHardSynchronization(suite.ctx, claim)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				// Check that the user's claim's reward index matches the expected reward index
				multiRewardIndex, found := syncedClaim.BorrowRewardIndexes.GetRewardIndex(tc.args.borrow.Denom)
				suite.Require().True(found)
				rewardIndex, found := multiRewardIndex.RewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, rewardIndex)

				// Check that the user's claim holds the expected amount of reward coins
				suite.Require().Equal(
					tc.args.expectedRewards.AmountOf(expectedRewardIndex.CollateralType),
					syncedClaim.Reward.AmountOf(expectedRewardIndex.CollateralType),
				)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSimulateHardDelegatorRewardSynchronization() {
	type args struct {
		delegation            sdk.Coin
		rewardsPerSecond      sdk.Coins
		initialTime           time.Time
		blockTimes            []int
		expectedRewardIndexes types.RewardIndexes
		expectedRewards       sdk.Coins
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"10 blocks",
			args{
				delegation:            c("ukava", 1_000_000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("ukava", d("6.117700000000000000"))}, // Here the reward index stores data differently than inside a MultiRewardIndex
				expectedRewards:       cs(c("hard", 6117700)),
			},
		},
		{
			"10 blocks - long block time",
			args{
				delegation:            c("ukava", 1_000_000),
				rewardsPerSecond:      cs(c("hard", 122354)),
				initialTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockTimes:            []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardIndexes: types.RewardIndexes{types.NewRewardIndex("ukava", d("52856.928000000000000000"))},
				expectedRewards:       cs(c("hard", 52856928000)),
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// Mint coins to hard module account
			supplyKeeper := suite.app.GetSupplyKeeper()
			hardMaccCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(200000000)))
			supplyKeeper.MintCoins(suite.ctx, hardtypes.ModuleAccountName, hardMaccCoins)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.delegation.Denom, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousHardDelegatorRewardAccrualTime(suite.ctx, tc.args.delegation.Denom, tc.args.initialTime)
			suite.keeper.SetHardDelegatorRewardFactor(suite.ctx, tc.args.delegation.Denom, sdk.ZeroDec())

			// Set up hard state (interest factor for the relevant denom)
			suite.hardKeeper.SetPreviousAccrualTime(suite.ctx, tc.args.delegation.Denom, tc.args.initialTime)

			// Delegator delegates
			err := suite.deliverMsgCreateValidator(suite.ctx, suite.validatorAddrs[0], tc.args.delegation)
			suite.Require().NoError(err)
			suite.deliverMsgDelegate(suite.ctx, suite.addrs[0], suite.validatorAddrs[0], tc.args.delegation)
			suite.Require().NoError(err)

			staking.EndBlocker(suite.ctx, suite.stakingKeeper)

			// Check that Staking hooks initialized a HardLiquidityProviderClaim
			claim, found := suite.keeper.GetHardLiquidityProviderClaim(suite.ctx, suite.addrs[0])
			suite.Require().True(found)
			suite.Require().Equal(sdk.ZeroDec(), claim.DelegatorRewardIndexes[0].RewardFactor)

			// Run accumulator at several intervals
			var timeElapsed int
			previousBlockTime := suite.ctx.BlockTime()
			for _, t := range tc.args.blockTimes {
				timeElapsed += t
				updatedBlockTime := previousBlockTime.Add(time.Duration(int(time.Second) * t))
				previousBlockTime = updatedBlockTime
				blockCtx := suite.ctx.WithBlockTime(updatedBlockTime)

				// Run Hard begin blocker for each block ctx to update denom's interest factor
				hard.BeginBlocker(blockCtx, suite.hardKeeper)

				// Accumulate hard delegator rewards
				rewardPeriod, found := suite.keeper.GetHardDelegatorRewardPeriod(blockCtx, tc.args.delegation.Denom)
				suite.Require().True(found)
				err := suite.keeper.AccumulateHardDelegatorRewards(blockCtx, rewardPeriod)
				suite.Require().NoError(err)
			}
			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)

			// Check that the synced claim held in memory has properly simulated syncing
			syncedClaim := suite.keeper.SimulateHardSynchronization(suite.ctx, claim)
			for _, expectedRewardIndex := range tc.args.expectedRewardIndexes {
				// Check that the user's claim's reward index matches the expected reward index
				rewardIndex, found := syncedClaim.DelegatorRewardIndexes.GetRewardIndex(expectedRewardIndex.CollateralType)
				suite.Require().True(found)
				suite.Require().Equal(expectedRewardIndex, rewardIndex)

				// Check that the user's claim holds the expected amount of reward coins
				suite.Require().Equal(
					tc.args.expectedRewards.AmountOf(expectedRewardIndex.CollateralType),
					syncedClaim.Reward.AmountOf(expectedRewardIndex.CollateralType),
				)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSimulateUSDXMintingRewardSynchronization() {
	type args struct {
		ctype                string
		rewardsPerSecond     sdk.Coins
		initialTime          time.Time
		initialCollateral    sdk.Coin
		initialPrincipal     sdk.Coin
		blockTimes           []int
		expectedRewardFactor sdk.Dec
		expectedRewards      sdk.Coin
	}
	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"10 blocks",
			args{
				ctype:                "bnb-a",
				rewardsPerSecond:     cs(c("ukava", 122354)),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:    c("bnb", 1000000000000),
				initialPrincipal:     c("usdx", 10000000000),
				blockTimes:           []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
				expectedRewardFactor: d("0.001223540000000000"),
				expectedRewards:      c("ukava", 12235400),
			},
		},
		{
			"10 blocks - long block time",
			args{
				ctype:                "bnb-a",
				rewardsPerSecond:     cs(c("ukava", 122354)),
				initialTime:          time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:    c("bnb", 1000000000000),
				initialPrincipal:     c("usdx", 10000000000),
				blockTimes:           []int{86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400, 86400},
				expectedRewardFactor: d("10.57138560000000000"),
				expectedRewards:      c("ukava", 105713856000),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWithGenState()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// setup incentive state
			params := types.NewParams(
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.MultiRewardPeriods{types.NewMultiRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond)},
				types.RewardPeriods{types.NewRewardPeriod(true, tc.args.ctype, tc.args.initialTime, tc.args.initialTime.Add(time.Hour*24*365*4), tc.args.rewardsPerSecond[0])},
				types.Multipliers{types.NewMultiplier(types.MultiplierName("small"), 1, d("0.25")), types.NewMultiplier(types.MultiplierName("large"), 12, d("1.0"))},
				tc.args.initialTime.Add(time.Hour*24*365*5),
			)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetParams(suite.ctx, params)
			suite.keeper.SetPreviousUSDXMintingAccrualTime(suite.ctx, tc.args.ctype, tc.args.initialTime)
			suite.keeper.SetUSDXMintingRewardFactor(suite.ctx, tc.args.ctype, sdk.ZeroDec())

			// setup account state
			sk := suite.app.GetSupplyKeeper()
			sk.MintCoins(suite.ctx, cdptypes.ModuleName, sdk.NewCoins(tc.args.initialCollateral))
			sk.SendCoinsFromModuleToAccount(suite.ctx, cdptypes.ModuleName, suite.addrs[0], sdk.NewCoins(tc.args.initialCollateral))

			// setup cdp state
			cdpKeeper := suite.app.GetCDPKeeper()
			err := cdpKeeper.AddCdp(suite.ctx, suite.addrs[0], tc.args.initialCollateral, tc.args.initialPrincipal, tc.args.ctype)
			suite.Require().NoError(err)

			claim, found := suite.keeper.GetUSDXMintingClaim(suite.ctx, suite.addrs[0])
			suite.Require().True(found)
			suite.Require().Equal(sdk.ZeroDec(), claim.RewardIndexes[0].RewardFactor)

			var timeElapsed int
			previousBlockTime := suite.ctx.BlockTime()
			for _, t := range tc.args.blockTimes {
				timeElapsed += t
				updatedBlockTime := previousBlockTime.Add(time.Duration(int(time.Second) * t))
				previousBlockTime = updatedBlockTime
				blockCtx := suite.ctx.WithBlockTime(updatedBlockTime)
				rewardPeriod, found := suite.keeper.GetUSDXMintingRewardPeriod(blockCtx, tc.args.ctype)
				suite.Require().True(found)
				err := suite.keeper.AccumulateUSDXMintingRewards(blockCtx, rewardPeriod)
				suite.Require().NoError(err)
			}
			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)

			claim, found = suite.keeper.GetUSDXMintingClaim(suite.ctx, suite.addrs[0])
			suite.Require().True(found)
			suite.Require().Equal(claim.RewardIndexes[0].RewardFactor, sdk.ZeroDec())
			suite.Require().Equal(claim.Reward, sdk.NewCoin("ukava", sdk.ZeroInt()))

			updatedClaim := suite.keeper.SimulateUSDXMintingSynchronization(suite.ctx, claim)
			suite.Require().Equal(tc.args.expectedRewardFactor, updatedClaim.RewardIndexes[0].RewardFactor)
			suite.Require().Equal(tc.args.expectedRewards, updatedClaim.Reward)
		})
	}
}

func (suite *KeeperTestSuite) SetupWithGenState() {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)

	_, allAddrs := app.GeneratePrivKeyAddressPairs(10)
	suite.addrs = allAddrs[:5]
	for _, a := range allAddrs[5:] {
		suite.validatorAddrs = append(suite.validatorAddrs, sdk.ValAddress(a))
	}

	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, abci.Header{Height: 1, Time: tmtime.Now()})

	tApp.InitializeFromGenesisStates(
		coinsAuthGenState(allAddrs, cs(c("ukava", 5_000_000))),
		stakingGenesisState(),
		NewPricefeedGenStateMulti(),
		NewCDPGenStateMulti(),
		NewHardGenStateMulti(),
	)

	suite.app = tApp
	suite.ctx = ctx
	suite.keeper = tApp.GetIncentiveKeeper()
	suite.hardKeeper = tApp.GetHardKeeper()
	suite.stakingKeeper = tApp.GetStakingKeeper()
}

func coinsAuthGenState(addresses []sdk.AccAddress, coins sdk.Coins) app.GenesisState {
	coinsList := []sdk.Coins{}
	for range addresses {
		coinsList = append(coinsList, coins)
	}

	// Load up our primary user address
	if len(addresses) >= 4 {
		coinsList[3] = sdk.NewCoins(
			sdk.NewCoin("bnb", sdk.NewInt(1000000000000000)),
			sdk.NewCoin("ukava", sdk.NewInt(1000000000000000)),
			sdk.NewCoin("btcb", sdk.NewInt(1000000000000000)),
			sdk.NewCoin("xrp", sdk.NewInt(1000000000000000)),
		)
	}

	return app.NewAuthGenState(addresses, coinsList)
}

func stakingGenesisState() app.GenesisState {
	genState := staking.DefaultGenesisState()
	genState.Params.BondDenom = "ukava"
	return app.GenesisState{
		staking.ModuleName: staking.ModuleCdc.MustMarshalJSON(genState),
	}
}

func (suite *KeeperTestSuite) deliverMsgCreateValidator(ctx sdk.Context, address sdk.ValAddress, selfDelegation sdk.Coin) error {
	msg := staking.NewMsgCreateValidator(
		address,
		ed25519.GenPrivKey().PubKey(),
		selfDelegation,
		staking.Description{},
		staking.NewCommissionRates(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
		sdk.NewInt(1_000_000),
	)
	handleStakingMsg := staking.NewHandler(suite.stakingKeeper)
	_, err := handleStakingMsg(ctx, msg)
	return err
}

func (suite *KeeperTestSuite) deliverMsgDelegate(ctx sdk.Context, delegator sdk.AccAddress, validator sdk.ValAddress, amount sdk.Coin) error {
	msg := staking.NewMsgDelegate(
		delegator,
		validator,
		amount,
	)
	handleStakingMsg := staking.NewHandler(suite.stakingKeeper)
	_, err := handleStakingMsg(ctx, msg)
	return err
}
