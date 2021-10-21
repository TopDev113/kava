package incentive_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/hard"
	"github.com/kava-labs/kava/x/incentive"
	"github.com/kava-labs/kava/x/kavadist"
)

const (
	oneYear time.Duration = 365 * 24 * time.Hour
)

type GenesisTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	app    app.TestApp
	keeper incentive.Keeper
	addrs  []sdk.AccAddress

	genesisTime time.Time
}

func (suite *GenesisTestSuite) SetupTest() {
	tApp := app.NewTestApp()
	keeper := tApp.GetIncentiveKeeper()
	suite.genesisTime = time.Date(1998, 1, 1, 0, 0, 0, 0, time.UTC)

	_, addrs := app.GeneratePrivKeyAddressPairs(5)

	authBuilder := app.NewAuthGenesisBuilder().
		WithSimpleAccount(addrs[0], cs(c("bnb", 1e10), c("ukava", 1e10))).
		WithSimpleModuleAccount(kavadist.KavaDistMacc, cs(c("hard", 1e15), c("ukava", 1e15)))

	loanToValue, _ := sdk.NewDecFromStr("0.6")
	borrowLimit := sdk.NewDec(1000000000000000)
	hardGS := hard.NewGenesisState(
		hard.NewParams(
			hard.MoneyMarkets{
				hard.NewMoneyMarket("ukava", hard.NewBorrowLimit(false, borrowLimit, loanToValue), "kava:usd", sdk.NewInt(1000000), hard.NewInterestRateModel(sdk.MustNewDecFromStr("0.05"), sdk.MustNewDecFromStr("2"), sdk.MustNewDecFromStr("0.8"), sdk.MustNewDecFromStr("10")), sdk.MustNewDecFromStr("0.05"), sdk.ZeroDec()),
				hard.NewMoneyMarket("bnb", hard.NewBorrowLimit(false, borrowLimit, loanToValue), "bnb:usd", sdk.NewInt(1000000), hard.NewInterestRateModel(sdk.MustNewDecFromStr("0.05"), sdk.MustNewDecFromStr("2"), sdk.MustNewDecFromStr("0.8"), sdk.MustNewDecFromStr("10")), sdk.MustNewDecFromStr("0.05"), sdk.ZeroDec()),
			},
			sdk.NewDec(10),
		),
		hard.DefaultAccumulationTimes,
		hard.DefaultDeposits,
		hard.DefaultBorrows,
		hard.DefaultTotalSupplied,
		hard.DefaultTotalBorrowed,
		hard.DefaultTotalReserves,
	)
	incentiveGS := incentive.NewGenesisState(
		incentive.NewParams(
			incentive.RewardPeriods{incentive.NewRewardPeriod(true, "bnb-a", suite.genesisTime.Add(-1*oneYear), suite.genesisTime.Add(oneYear), c("ukava", 122354))},
			incentive.MultiRewardPeriods{incentive.NewMultiRewardPeriod(true, "bnb", suite.genesisTime.Add(-1*oneYear), suite.genesisTime.Add(oneYear), cs(c("hard", 122354)))},
			incentive.MultiRewardPeriods{incentive.NewMultiRewardPeriod(true, "bnb", suite.genesisTime.Add(-1*oneYear), suite.genesisTime.Add(oneYear), cs(c("hard", 122354)))},
			incentive.MultiRewardPeriods{incentive.NewMultiRewardPeriod(true, "ukava", suite.genesisTime.Add(-1*oneYear), suite.genesisTime.Add(oneYear), cs(c("hard", 122354)))},
			incentive.MultiRewardPeriods{incentive.NewMultiRewardPeriod(true, "btcb/usdx", suite.genesisTime.Add(-1*oneYear), suite.genesisTime.Add(oneYear), cs(c("swp", 122354)))},
			incentive.MultipliersPerDenom{
				{
					Denom: "ukava",
					Multipliers: incentive.Multipliers{
						incentive.NewMultiplier(incentive.Large, 12, d("1.0")),
					},
				},
				{
					Denom: "hard",
					Multipliers: incentive.Multipliers{
						incentive.NewMultiplier(incentive.Small, 1, d("0.25")),
						incentive.NewMultiplier(incentive.Large, 12, d("1.0")),
					},
				},
				{
					Denom: "swp",
					Multipliers: incentive.Multipliers{
						incentive.NewMultiplier(incentive.Small, 1, d("0.25")),
						incentive.NewMultiplier(incentive.Medium, 6, d("0.8")),
					},
				},
			},
			suite.genesisTime.Add(5*oneYear),
		),
		incentive.DefaultGenesisRewardState,
		incentive.DefaultGenesisRewardState,
		incentive.DefaultGenesisRewardState,
		incentive.DefaultGenesisRewardState,
		incentive.DefaultGenesisRewardState,
		incentive.DefaultUSDXClaims,
		incentive.DefaultHardClaims,
		incentive.DefaultDelegatorClaims,
		incentive.DefaultSwapClaims,
	)
	tApp.InitializeFromGenesisStatesWithTime(
		suite.genesisTime,
		authBuilder.BuildMarshalled(),
		app.GenesisState{incentive.ModuleName: incentive.ModuleCdc.MustMarshalJSON(incentiveGS)},
		app.GenesisState{hard.ModuleName: hard.ModuleCdc.MustMarshalJSON(hardGS)},
		NewCDPGenStateMulti(),
		NewPricefeedGenStateMultiFromTime(suite.genesisTime),
	)

	ctx := tApp.NewContext(true, abci.Header{Height: 1, Time: suite.genesisTime})

	suite.addrs = addrs
	suite.keeper = keeper
	suite.app = tApp
	suite.ctx = ctx
}

func (suite *GenesisTestSuite) TestExportedGenesisMatchesImported() {
	genesisTime := time.Date(1998, 1, 1, 0, 0, 0, 0, time.UTC)
	genesisState := incentive.NewGenesisState(
		incentive.NewParams(
			incentive.RewardPeriods{incentive.NewRewardPeriod(true, "bnb-a", genesisTime.Add(-1*oneYear), genesisTime.Add(oneYear), c("ukava", 122354))},
			incentive.MultiRewardPeriods{incentive.NewMultiRewardPeriod(true, "bnb", genesisTime.Add(-1*oneYear), genesisTime.Add(oneYear), cs(c("hard", 122354)))},
			incentive.MultiRewardPeriods{incentive.NewMultiRewardPeriod(true, "bnb", genesisTime.Add(-1*oneYear), genesisTime.Add(oneYear), cs(c("hard", 122354)))},
			incentive.MultiRewardPeriods{incentive.NewMultiRewardPeriod(true, "ukava", genesisTime.Add(-1*oneYear), genesisTime.Add(oneYear), cs(c("hard", 122354)))},
			incentive.MultiRewardPeriods{incentive.NewMultiRewardPeriod(true, "btcb/usdx", genesisTime.Add(-1*oneYear), genesisTime.Add(oneYear), cs(c("swp", 122354)))},
			incentive.MultipliersPerDenom{
				{
					Denom: "ukava",
					Multipliers: incentive.Multipliers{
						incentive.NewMultiplier(incentive.Large, 12, d("1.0")),
					},
				},
				{
					Denom: "hard",
					Multipliers: incentive.Multipliers{
						incentive.NewMultiplier(incentive.Small, 1, d("0.25")),
						incentive.NewMultiplier(incentive.Large, 12, d("1.0")),
					},
				},
				{
					Denom: "swp",
					Multipliers: incentive.Multipliers{
						incentive.NewMultiplier(incentive.Small, 1, d("0.25")),
						incentive.NewMultiplier(incentive.Medium, 6, d("0.8")),
					},
				},
			},
			genesisTime.Add(5*oneYear),
		),
		incentive.NewGenesisRewardState(
			incentive.AccumulationTimes{
				incentive.NewAccumulationTime("bnb-a", genesisTime),
			},
			incentive.MultiRewardIndexes{
				incentive.NewMultiRewardIndex("bnb-a", incentive.RewardIndexes{{CollateralType: "ukava", RewardFactor: d("0.3")}}),
			},
		),
		incentive.NewGenesisRewardState(
			incentive.AccumulationTimes{
				incentive.NewAccumulationTime("bnb", genesisTime.Add(-1*time.Hour)),
			},
			incentive.MultiRewardIndexes{
				incentive.NewMultiRewardIndex("bnb", incentive.RewardIndexes{{CollateralType: "hard", RewardFactor: d("0.1")}}),
			},
		),
		incentive.NewGenesisRewardState(
			incentive.AccumulationTimes{
				incentive.NewAccumulationTime("bnb", genesisTime.Add(-2*time.Hour)),
			},
			incentive.MultiRewardIndexes{
				incentive.NewMultiRewardIndex("bnb", incentive.RewardIndexes{{CollateralType: "hard", RewardFactor: d("0.05")}}),
			},
		),
		incentive.NewGenesisRewardState(
			incentive.AccumulationTimes{
				incentive.NewAccumulationTime("ukava", genesisTime.Add(-3*time.Hour)),
			},
			incentive.MultiRewardIndexes{
				incentive.NewMultiRewardIndex("ukava", incentive.RewardIndexes{{CollateralType: "hard", RewardFactor: d("0.2")}}),
			},
		),
		incentive.NewGenesisRewardState(
			incentive.AccumulationTimes{
				incentive.NewAccumulationTime("bctb/usdx", genesisTime.Add(-4*time.Hour)),
			},
			incentive.MultiRewardIndexes{
				incentive.NewMultiRewardIndex("btcb/usdx", incentive.RewardIndexes{{CollateralType: "swap", RewardFactor: d("0.001")}}),
			},
		),
		incentive.USDXMintingClaims{
			incentive.NewUSDXMintingClaim(
				suite.addrs[0],
				c("ukava", 1e9),
				incentive.RewardIndexes{{CollateralType: "bnb-a", RewardFactor: d("0.3")}},
			),
			incentive.NewUSDXMintingClaim(
				suite.addrs[1],
				c("ukava", 1),
				incentive.RewardIndexes{{CollateralType: "bnb-a", RewardFactor: d("0.001")}},
			),
		},
		incentive.HardLiquidityProviderClaims{
			incentive.NewHardLiquidityProviderClaim(
				suite.addrs[0],
				cs(c("ukava", 1e9), c("hard", 1e9)),
				incentive.MultiRewardIndexes{{CollateralType: "bnb", RewardIndexes: incentive.RewardIndexes{{CollateralType: "hard", RewardFactor: d("0.01")}}}},
				incentive.MultiRewardIndexes{{CollateralType: "bnb", RewardIndexes: incentive.RewardIndexes{{CollateralType: "hard", RewardFactor: d("0.0")}}}},
			),
			incentive.NewHardLiquidityProviderClaim(
				suite.addrs[1],
				cs(c("hard", 1)),
				incentive.MultiRewardIndexes{{CollateralType: "bnb", RewardIndexes: incentive.RewardIndexes{{CollateralType: "hard", RewardFactor: d("0.1")}}}},
				incentive.MultiRewardIndexes{{CollateralType: "bnb", RewardIndexes: incentive.RewardIndexes{{CollateralType: "hard", RewardFactor: d("0.0")}}}},
			),
		},
		incentive.DelegatorClaims{
			incentive.NewDelegatorClaim(
				suite.addrs[2],
				cs(c("hard", 5)),
				incentive.MultiRewardIndexes{{CollateralType: "ukava", RewardIndexes: incentive.RewardIndexes{{CollateralType: "hard", RewardFactor: d("0.2")}}}},
			),
		},
		incentive.SwapClaims{
			incentive.NewSwapClaim(
				suite.addrs[3],
				nil,
				incentive.MultiRewardIndexes{{CollateralType: "btcb/usdx", RewardIndexes: incentive.RewardIndexes{{CollateralType: "swap", RewardFactor: d("0.0")}}}},
			),
		},
	)

	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, abci.Header{Height: 0, Time: genesisTime})

	// Incentive init genesis reads from the cdp keeper to check params are ok. So it needs to be initialized first.
	// Then the cdp keeper reads from pricefeed keeper to check its params are ok. So it also need initialization.
	tApp.InitializeFromGenesisStates(
		NewCDPGenStateMulti(),
		NewPricefeedGenStateMultiFromTime(genesisTime),
	)

	incentive.InitGenesis(ctx, tApp.GetIncentiveKeeper(), tApp.GetSupplyKeeper(), tApp.GetCDPKeeper(), genesisState)

	exportedGenesisState := incentive.ExportGenesis(ctx, tApp.GetIncentiveKeeper())

	suite.Equal(genesisState, exportedGenesisState)
}

func (suite *GenesisTestSuite) TestInitGenesisPanicsWhenAccumulationTimesToLongAgo() {
	genesisTime := time.Date(1998, 1, 1, 0, 0, 0, 0, time.UTC)
	invalidRewardState := incentive.NewGenesisRewardState(
		incentive.AccumulationTimes{
			incentive.NewAccumulationTime("bnb", genesisTime.Add(-23*incentive.EarliestValidAccumulationTime).Add(-time.Nanosecond)),
		},
		incentive.MultiRewardIndexes{},
	)
	minimalParams := incentive.Params{
		ClaimEnd: genesisTime.Add(5 * oneYear),
	}

	testCases := []struct {
		genesisState incentive.GenesisState
	}{
		{
			incentive.GenesisState{
				Params:          minimalParams,
				USDXRewardState: invalidRewardState,
			},
		},
		{
			incentive.GenesisState{
				Params:                minimalParams,
				HardSupplyRewardState: invalidRewardState,
			},
		},
		{
			incentive.GenesisState{
				Params:                minimalParams,
				HardBorrowRewardState: invalidRewardState,
			},
		},
		{
			incentive.GenesisState{
				Params:               minimalParams,
				DelegatorRewardState: invalidRewardState,
			},
		},
		{
			incentive.GenesisState{
				Params:          minimalParams,
				SwapRewardState: invalidRewardState,
			},
		},
	}

	for _, tc := range testCases {

		tApp := app.NewTestApp()
		ctx := tApp.NewContext(true, abci.Header{Height: 0, Time: genesisTime})

		// Incentive init genesis reads from the cdp keeper to check params are ok. So it needs to be initialized first.
		// Then the cdp keeper reads from pricefeed keeper to check its params are ok. So it also need initialization.
		tApp.InitializeFromGenesisStates(
			NewCDPGenStateMulti(),
			NewPricefeedGenStateMultiFromTime(genesisTime),
		)

		suite.PanicsWithValue(
			"found accumulation time '1975-01-06 23:59:59.999999999 +0000 UTC' more than '8760h0m0s' behind genesis time '1998-01-01 00:00:00 +0000 UTC'",
			func() {
				incentive.InitGenesis(ctx, tApp.GetIncentiveKeeper(), tApp.GetSupplyKeeper(), tApp.GetCDPKeeper(), tc.genesisState)
			},
		)
	}
}

func (suite *GenesisTestSuite) TestValidateAccumulationTime() {
	genTime := time.Date(1998, 1, 1, 0, 0, 0, 0, time.UTC)

	err := incentive.ValidateAccumulationTime(
		genTime.Add(-incentive.EarliestValidAccumulationTime).Add(-time.Nanosecond),
		genTime,
	)
	suite.Error(err)
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}
