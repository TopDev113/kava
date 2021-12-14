package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/incentive/testutil"
	"github.com/kava-labs/kava/x/incentive/types"
	kavadisttypes "github.com/kava-labs/kava/x/kavadist/types"
)

const secondsPerDay = 24 * 60 * 60

// Test suite used for all keeper tests
type HandlerTestSuite struct {
	testutil.IntegrationTester

	genesisTime time.Time
	addrs       []sdk.AccAddress
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

// SetupTest is run automatically before each suite test
func (suite *HandlerTestSuite) SetupTest() {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)

	_, suite.addrs = app.GeneratePrivKeyAddressPairs(5)

	suite.genesisTime = time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC)
}

func (suite *HandlerTestSuite) SetupApp() {
	suite.App = app.NewTestApp()

	suite.Ctx = suite.App.NewContext(true, tmproto.Header{Height: 1, Time: suite.genesisTime})
}

type genesisBuilder interface {
	BuildMarshalled(cdc codec.JSONCodec) app.GenesisState
}

func (suite *HandlerTestSuite) SetupWithGenState(builders ...genesisBuilder) {
	suite.SetupApp()

	builtGenStates := []app.GenesisState{
		NewStakingGenesisState(suite.App.AppCodec()),
		NewPricefeedGenStateMultiFromTime(suite.App.AppCodec(), suite.genesisTime),
		NewCDPGenStateMulti(suite.App.AppCodec()),
		NewHardGenStateMulti(suite.genesisTime).BuildMarshalled(suite.App.AppCodec()),
		NewSwapGenesisState(suite.App.AppCodec()),
	}
	for _, builder := range builders {
		builtGenStates = append(builtGenStates, builder.BuildMarshalled(suite.App.AppCodec()))
	}

	suite.App.InitializeFromGenesisStatesWithTime(
		suite.genesisTime,
		builtGenStates...,
	)
}

// authBuilder returns a new auth genesis builder with a full kavadist module account.
func (suite *HandlerTestSuite) authBuilder() *app.AuthBankGenesisBuilder {
	return app.NewAuthBankGenesisBuilder().
		WithSimpleModuleAccount(kavadisttypes.ModuleName, cs(c(types.USDXMintingRewardDenom, 1e18), c("hard", 1e18), c("swap", 1e18)))
}

// incentiveBuilder returns a new incentive genesis builder with a genesis time and multipliers set
func (suite *HandlerTestSuite) incentiveBuilder() testutil.IncentiveGenesisBuilder {
	return testutil.NewIncentiveGenesisBuilder().
		WithGenesisTime(suite.genesisTime).
		WithMultipliers(types.MultipliersPerDenoms{
			{
				Denom: "hard",
				Multipliers: types.Multipliers{
					types.NewMultiplier("small", 1, d("0.2")),
					types.NewMultiplier("large", 12, d("1.0")),
				},
			},
			{
				Denom: "swap",
				Multipliers: types.Multipliers{
					types.NewMultiplier("medium", 6, d("0.5")),
					types.NewMultiplier("large", 12, d("1.0")),
				},
			},
			{
				Denom: "ukava",
				Multipliers: types.Multipliers{
					types.NewMultiplier("small", 1, d("0.2")),
					types.NewMultiplier("large", 12, d("1.0")),
				},
			},
		})
}

func (suite *HandlerTestSuite) TestPayoutSwapClaimMultiDenom() {
	userAddr := suite.addrs[0]

	authBulder := suite.authBuilder().
		WithSimpleAccount(userAddr, cs(c("ukava", 1e12), c("busd", 1e12)))

	incentBuilder := suite.incentiveBuilder().
		WithSimpleSwapRewardPeriod("busd:ukava", cs(c("hard", 1e6), c("swap", 1e6)))

	suite.SetupWithGenState(authBulder, incentBuilder)

	// deposit into a swap pool
	suite.NoError(
		suite.DeliverSwapMsgDeposit(userAddr, c("ukava", 1e9), c("busd", 1e9), d("1.0")),
	)
	// accumulate some swap rewards
	suite.NextBlockAfter(7 * time.Second)

	preClaimBal := suite.GetBalance(userAddr)

	msg := types.NewMsgClaimSwapReward(
		userAddr.String(),
		types.Selections{
			types.NewSelection("hard", "small"),
			types.NewSelection("swap", "medium"),
		},
	)

	// Claim rewards
	err := suite.DeliverIncentiveMsg(&msg)
	suite.NoError(err)

	// Check rewards were paid out
	expectedRewardsHard := c("hard", int64(0.2*float64(7*1e6)))
	expectedRewardsSwap := c("swap", int64(0.5*float64(7*1e6)))
	suite.BalanceEquals(userAddr, preClaimBal.Add(expectedRewardsHard, expectedRewardsSwap))

	suite.VestingPeriodsEqual(userAddr, []vestingtypes.Period{
		{Length: (17+31)*secondsPerDay - 7, Amount: cs(expectedRewardsHard)},
		{Length: (28 + 31 + 30 + 31 + 30) * secondsPerDay, Amount: cs(expectedRewardsSwap)}, // second length is stacked on top of the first
	})

	// Check that each claim reward coin's amount has been reset to 0
	suite.SwapRewardEquals(userAddr, nil)
}

func (suite *HandlerTestSuite) TestPayoutSwapClaimSingleDenom() {
	userAddr := suite.addrs[0]

	authBulder := suite.authBuilder().
		WithSimpleAccount(userAddr, cs(c("ukava", 1e12), c("busd", 1e12)))

	incentBuilder := suite.incentiveBuilder().
		WithSimpleSwapRewardPeriod("busd:ukava", cs(c("hard", 1e6), c("swap", 1e6)))

	suite.SetupWithGenState(authBulder, incentBuilder)

	// deposit into a swap pool
	suite.NoError(
		suite.DeliverSwapMsgDeposit(userAddr, c("ukava", 1e9), c("busd", 1e9), d("1.0")),
	)

	// accumulate some swap rewards
	suite.NextBlockAfter(7 * time.Second)

	preClaimBal := suite.GetBalance(userAddr)

	msg := types.NewMsgClaimSwapReward(
		userAddr.String(),
		types.Selections{
			types.NewSelection("swap", "large"),
		},
	)

	// Claim rewards
	err := suite.DeliverIncentiveMsg(&msg)
	suite.NoError(err)

	// Check rewards were paid out
	expectedRewards := c("swap", 7*1e6)
	suite.BalanceEquals(userAddr, preClaimBal.Add(expectedRewards))

	suite.VestingPeriodsEqual(userAddr, vestingtypes.Periods{
		{Length: (17+31+28+31+30+31+30+31+31+30+31+30+31)*secondsPerDay - 7, Amount: cs(expectedRewards)},
	})

	// Check that claimed coins have been removed from a claim's reward
	suite.SwapRewardEquals(userAddr, cs(c("hard", 7*1e6)))
}
