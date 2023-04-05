package keeper_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/suite"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/cdp/keeper"
	"github.com/kava-labs/kava/x/cdp/types"
)

type InterestTestSuite struct {
	suite.Suite

	keeper keeper.Keeper
	app    app.TestApp
	ctx    sdk.Context
}

func (suite *InterestTestSuite) SetupTest() {
	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})
	cdc := tApp.AppCodec()
	tApp.InitializeFromGenesisStates(
		NewPricefeedGenStateMulti(cdc),
		NewCDPGenStateMulti(cdc),
	)
	keeper := tApp.GetCDPKeeper()
	suite.app = tApp
	suite.ctx = ctx
	suite.keeper = keeper
}

func (suite *InterestTestSuite) TestCalculateInterestFactor() {
	type args struct {
		perSecondInterestRate sdk.Dec
		timeElapsed           sdkmath.Int
		expectedValue         sdk.Dec
	}

	type test struct {
		name string
		args args
	}

	oneYearInSeconds := int64(31536000)

	testCases := []test{
		{
			"1 year",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000005555"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds),
				expectedValue:         sdk.MustNewDecFromStr("1.191463614477847370"),
			},
		},
		{
			"10 year",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000005555"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds * 10),
				expectedValue:         sdk.MustNewDecFromStr("5.765113233897391189"),
			},
		},
		{
			"1 month",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000005555"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds / 12),
				expectedValue:         sdk.MustNewDecFromStr("1.014705619075717373"),
			},
		},
		{
			"1 day",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000005555"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds / 365),
				expectedValue:         sdk.MustNewDecFromStr("1.000480067194057924"),
			},
		},
		{
			"1 year: low interest rate",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000000555"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds),
				expectedValue:         sdk.MustNewDecFromStr("1.017656545925063632"),
			},
		},
		{
			"1 year, lower interest rate",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000000055"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds),
				expectedValue:         sdk.MustNewDecFromStr("1.001735985079841390"),
			},
		},
		{
			"1 year, lowest interest rate",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000000005"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds),
				expectedValue:         sdk.MustNewDecFromStr("1.000157692432076670"),
			},
		},
		{
			"1 year: high interest rate",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000055555"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds),
				expectedValue:         sdk.MustNewDecFromStr("5.766022095987868825"),
			},
		},
		{
			"1 year: higher interest rate",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000000555555"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds),
				expectedValue:         sdk.MustNewDecFromStr("40628388.864535408465693310"),
			},
		},
		// If we raise the per second interest rate too much we'll cause an integer overflow.
		// For example, perSecondInterestRate: '1.000005555555' will cause a panic.
		{
			"1 year: highest interest rate",
			args{
				perSecondInterestRate: sdk.MustNewDecFromStr("1.000001555555"),
				timeElapsed:           sdkmath.NewInt(oneYearInSeconds),
				expectedValue:         sdk.MustNewDecFromStr("2017093013158200407564.613502861572552603"),
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			interestFactor := keeper.CalculateInterestFactor(tc.args.perSecondInterestRate, tc.args.timeElapsed)
			suite.Require().Equal(tc.args.expectedValue, interestFactor)
		})
	}
}

func (suite *InterestTestSuite) TestAccumulateInterest() {
	type args struct {
		ctype                   string
		initialTime             time.Time
		totalPrincipal          sdkmath.Int
		timeElapsed             int
		expectedTotalPrincipal  sdkmath.Int
		expectedLastAccrualTime time.Time
	}

	type test struct {
		name string
		args args
	}
	oneYearInSeconds := 31536000

	testCases := []test{
		{
			"1 year",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				totalPrincipal:          sdkmath.NewInt(100000000000000),
				timeElapsed:             oneYearInSeconds,
				expectedTotalPrincipal:  sdkmath.NewInt(105000000000012),
				expectedLastAccrualTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * oneYearInSeconds)),
			},
		},
		{
			"1 year - zero principal",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				totalPrincipal:          sdk.ZeroInt(),
				timeElapsed:             oneYearInSeconds,
				expectedTotalPrincipal:  sdk.ZeroInt(),
				expectedLastAccrualTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * oneYearInSeconds)),
			},
		},
		{
			"1 month",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				totalPrincipal:          sdkmath.NewInt(100000000000000),
				timeElapsed:             86400 * 30,
				expectedTotalPrincipal:  sdkmath.NewInt(100401820189198),
				expectedLastAccrualTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 86400 * 30)),
			},
		},
		{
			"1 month - interest rounds to zero",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				totalPrincipal:          sdkmath.NewInt(10),
				timeElapsed:             86400 * 30,
				expectedTotalPrincipal:  sdkmath.NewInt(10),
				expectedLastAccrualTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
			},
		},
		{
			"7 seconds",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				totalPrincipal:          sdkmath.NewInt(100000000000000),
				timeElapsed:             7,
				expectedTotalPrincipal:  sdkmath.NewInt(100000001082988),
				expectedLastAccrualTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 7)),
			},
		},
		{
			"7 seconds - interest rounds to zero",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				totalPrincipal:          sdkmath.NewInt(30000000),
				timeElapsed:             7,
				expectedTotalPrincipal:  sdkmath.NewInt(30000000),
				expectedLastAccrualTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
			},
		},
		{
			"7 seconds - zero interest",
			args{
				ctype:                   "busd-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				totalPrincipal:          sdkmath.NewInt(100000000000000),
				timeElapsed:             7,
				expectedTotalPrincipal:  sdkmath.NewInt(100000000000000),
				expectedLastAccrualTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 7)),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)
			suite.keeper.SetTotalPrincipal(suite.ctx, tc.args.ctype, types.DefaultStableDenom, tc.args.totalPrincipal)
			suite.keeper.SetPreviousAccrualTime(suite.ctx, tc.args.ctype, suite.ctx.BlockTime())
			suite.keeper.SetInterestFactor(suite.ctx, tc.args.ctype, sdk.OneDec())

			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)
			err := suite.keeper.AccumulateInterest(suite.ctx, tc.args.ctype)
			suite.Require().NoError(err)

			actualTotalPrincipal := suite.keeper.GetTotalPrincipal(suite.ctx, tc.args.ctype, types.DefaultStableDenom)
			suite.Require().Equal(tc.args.expectedTotalPrincipal, actualTotalPrincipal)
			actualAccrualTime, _ := suite.keeper.GetPreviousAccrualTime(suite.ctx, tc.args.ctype)
			suite.Require().Equal(tc.args.expectedLastAccrualTime, actualAccrualTime)
		})
	}
}

// TestSynchronizeInterest tests the functionality of synchronizing the accumulated interest for CDPs
func (suite *InterestTestSuite) TestSynchronizeInterest() {
	type args struct {
		ctype                   string
		initialTime             time.Time
		initialCollateral       sdk.Coin
		initialPrincipal        sdk.Coin
		timeElapsed             int
		expectedFees            sdk.Coin
		expectedFeesUpdatedTime time.Time
	}

	type test struct {
		name string
		args args
	}

	oneYearInSeconds := 31536000
	testCases := []test{
		{
			"1 year",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:       c("bnb", 1000000000000),
				initialPrincipal:        c("usdx", 100000000000),
				timeElapsed:             oneYearInSeconds,
				expectedFees:            c("usdx", 5000000000),
				expectedFeesUpdatedTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * oneYearInSeconds)),
			},
		},
		{
			"1 month",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:       c("bnb", 1000000000000),
				initialPrincipal:        c("usdx", 100000000000),
				timeElapsed:             86400 * 30,
				expectedFees:            c("usdx", 401820189),
				expectedFeesUpdatedTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 86400 * 30)),
			},
		},
		{
			"7 seconds",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:       c("bnb", 1000000000000),
				initialPrincipal:        c("usdx", 100000000000),
				timeElapsed:             7,
				expectedFees:            c("usdx", 1083),
				expectedFeesUpdatedTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 7)),
			},
		},
		{
			"7 seconds - zero apy",
			args{
				ctype:                   "busd-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:       c("busd", 10000000000000),
				initialPrincipal:        c("usdx", 10000000000),
				timeElapsed:             7,
				expectedFees:            c("usdx", 0),
				expectedFeesUpdatedTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 7)),
			},
		},
		{
			"7 seconds - fees round to zero",
			args{
				ctype:                   "bnb-a",
				initialTime:             time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral:       c("bnb", 1000000000),
				initialPrincipal:        c("usdx", 10000000),
				timeElapsed:             7,
				expectedFees:            c("usdx", 0),
				expectedFeesUpdatedTime: time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// setup account state
			_, addrs := app.GeneratePrivKeyAddressPairs(1)
			ak := suite.app.GetAccountKeeper()
			// setup the first account
			acc := ak.NewAccountWithAddress(suite.ctx, addrs[0])
			ak.SetAccount(suite.ctx, acc)
			bk := suite.app.GetBankKeeper()

			err := bk.MintCoins(suite.ctx, types.ModuleName, cs(tc.args.initialCollateral))
			suite.Require().NoError(err)
			err = bk.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addrs[0], cs(tc.args.initialCollateral))
			suite.Require().NoError(err)

			// setup pricefeed
			pk := suite.app.GetPriceFeedKeeper()
			_, err = pk.SetPrice(suite.ctx, sdk.AccAddress{}, "bnb:usd", d("17.25"), tc.args.expectedFeesUpdatedTime.Add(time.Second))
			suite.NoError(err)
			_, err = pk.SetPrice(suite.ctx, sdk.AccAddress{}, "busd:usd", d("1"), tc.args.expectedFeesUpdatedTime.Add(time.Second))
			suite.NoError(err)

			// setup cdp state
			suite.keeper.SetPreviousAccrualTime(suite.ctx, tc.args.ctype, suite.ctx.BlockTime())
			suite.keeper.SetInterestFactor(suite.ctx, tc.args.ctype, sdk.OneDec())
			err = suite.keeper.AddCdp(suite.ctx, addrs[0], tc.args.initialCollateral, tc.args.initialPrincipal, tc.args.ctype)
			suite.Require().NoError(err)

			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)
			err = suite.keeper.AccumulateInterest(suite.ctx, tc.args.ctype)
			suite.Require().NoError(err)

			cdp, found := suite.keeper.GetCDP(suite.ctx, tc.args.ctype, 1)
			suite.Require().True(found)

			cdp = suite.keeper.SynchronizeInterest(suite.ctx, cdp)

			suite.Require().Equal(tc.args.expectedFees, cdp.AccumulatedFees)
			suite.Require().Equal(tc.args.expectedFeesUpdatedTime, cdp.FeesUpdated)
		})
	}
}

func (suite *InterestTestSuite) TestMultipleCDPInterest() {
	type args struct {
		ctype                        string
		initialTime                  time.Time
		blockInterval                int
		numberOfBlocks               int
		initialCDPCollateral         sdk.Coin
		initialCDPPrincipal          sdk.Coin
		numberOfCdps                 int
		expectedFeesPerCDP           sdk.Coin
		expectedTotalPrincipalPerCDP sdk.Coin
		expectedFeesUpdatedTime      time.Time
		expectedTotalPrincipal       sdkmath.Int
		expectedDebtBalance          sdkmath.Int
		expectedStableBalance        sdkmath.Int
		expectedSumOfCDPPrincipal    sdkmath.Int
	}

	type test struct {
		name string
		args args
	}

	testCases := []test{
		{
			"1 block",
			args{
				ctype:                        "bnb-a",
				initialTime:                  time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockInterval:                7,
				numberOfBlocks:               1,
				initialCDPCollateral:         c("bnb", 10000000000),
				initialCDPPrincipal:          c("usdx", 500000000),
				numberOfCdps:                 100,
				expectedFeesPerCDP:           c("usdx", 5),
				expectedTotalPrincipalPerCDP: c("usdx", 500000005),
				expectedFeesUpdatedTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 7)),
				expectedTotalPrincipal:       i(50000000541),
				expectedDebtBalance:          i(50000000541),
				expectedStableBalance:        i(50000000541),
				expectedSumOfCDPPrincipal:    i(50000000500),
			},
		},
		{
			"100 blocks",
			args{
				ctype:                        "bnb-a",
				initialTime:                  time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockInterval:                7,
				numberOfBlocks:               100,
				initialCDPCollateral:         c("bnb", 10000000000),
				initialCDPPrincipal:          c("usdx", 500000000),
				numberOfCdps:                 100,
				expectedFeesPerCDP:           c("usdx", 541),
				expectedTotalPrincipalPerCDP: c("usdx", 500000541),
				expectedFeesUpdatedTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 7 * 100)),
				expectedTotalPrincipal:       i(50000054100),
				expectedDebtBalance:          i(50000054100),
				expectedStableBalance:        i(50000054100),
				expectedSumOfCDPPrincipal:    i(50000054100),
			},
		},
		{
			"10000 blocks",
			args{
				ctype:                        "bnb-a",
				initialTime:                  time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				blockInterval:                7,
				numberOfBlocks:               10000,
				initialCDPCollateral:         c("bnb", 10000000000),
				initialCDPPrincipal:          c("usdx", 500000000),
				numberOfCdps:                 100,
				expectedFeesPerCDP:           c("usdx", 54152),
				expectedTotalPrincipalPerCDP: c("usdx", 500054152),
				expectedFeesUpdatedTime:      time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC).Add(time.Duration(int(time.Second) * 7 * 10000)),
				expectedTotalPrincipal:       i(50005418990),
				expectedDebtBalance:          i(50005418990),
				expectedStableBalance:        i(50005418990),
				expectedSumOfCDPPrincipal:    i(50005415200),
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// setup pricefeed
			pk := suite.app.GetPriceFeedKeeper()
			_, err := pk.SetPrice(suite.ctx, sdk.AccAddress{}, "bnb:usd", d("17.25"), tc.args.expectedFeesUpdatedTime.Add(time.Second))
			suite.NoError(err)

			// setup cdp state
			suite.keeper.SetPreviousAccrualTime(suite.ctx, tc.args.ctype, suite.ctx.BlockTime())
			suite.keeper.SetInterestFactor(suite.ctx, tc.args.ctype, sdk.OneDec())

			// setup account state
			_, addrs := app.GeneratePrivKeyAddressPairs(tc.args.numberOfCdps)
			for j := 0; j < tc.args.numberOfCdps; j++ {
				ak := suite.app.GetAccountKeeper()
				// setup the first account
				acc := ak.NewAccountWithAddress(suite.ctx, addrs[j])
				ak.SetAccount(suite.ctx, acc)
				bk := suite.app.GetBankKeeper()
				err := bk.MintCoins(suite.ctx, types.ModuleName, cs(tc.args.initialCDPCollateral))
				suite.Require().NoError(err)
				err = bk.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addrs[j], cs(tc.args.initialCDPCollateral))
				suite.Require().NoError(err)
				err = suite.keeper.AddCdp(suite.ctx, addrs[j], tc.args.initialCDPCollateral, tc.args.initialCDPPrincipal, tc.args.ctype)
				suite.Require().NoError(err)
			}

			// run a number of blocks where CDPs are not synchronized
			for j := 0; j < tc.args.numberOfBlocks; j++ {
				updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.blockInterval))
				suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)
				err := suite.keeper.AccumulateInterest(suite.ctx, tc.args.ctype)
				suite.Require().NoError(err)
			}

			bk := suite.app.GetBankKeeper()
			debtSupply := bk.GetSupply(suite.ctx, types.DefaultDebtDenom)
			usdxSupply := bk.GetSupply(suite.ctx, types.DefaultStableDenom)
			totalPrincipal := suite.keeper.GetTotalPrincipal(suite.ctx, tc.args.ctype, types.DefaultStableDenom)

			suite.Require().Equal(tc.args.expectedDebtBalance, debtSupply.Amount)
			suite.Require().Equal(tc.args.expectedStableBalance, usdxSupply.Amount)
			suite.Require().Equal(tc.args.expectedTotalPrincipal, totalPrincipal)

			sumOfCDPPrincipal := sdk.ZeroInt()

			for j := 0; j < tc.args.numberOfCdps; j++ {
				cdp, found := suite.keeper.GetCDP(suite.ctx, tc.args.ctype, uint64(j+1))
				suite.Require().True(found)
				cdp = suite.keeper.SynchronizeInterest(suite.ctx, cdp)
				suite.Require().Equal(tc.args.expectedFeesPerCDP, cdp.AccumulatedFees)
				suite.Require().Equal(tc.args.expectedTotalPrincipalPerCDP, cdp.GetTotalPrincipal())
				suite.Require().Equal(tc.args.expectedFeesUpdatedTime, cdp.FeesUpdated)
				sumOfCDPPrincipal = sumOfCDPPrincipal.Add(cdp.GetTotalPrincipal().Amount)
			}

			suite.Require().Equal(tc.args.expectedSumOfCDPPrincipal, sumOfCDPPrincipal)
		})
	}
}

// TestSynchronizeInterest tests the functionality of synchronizing the accumulated interest for CDPs
func (suite *InterestTestSuite) TestCalculateCDPInterest() {
	type args struct {
		ctype             string
		initialTime       time.Time
		initialCollateral sdk.Coin
		initialPrincipal  sdk.Coin
		timeElapsed       int
		expectedFees      sdk.Coin
	}

	type test struct {
		name string
		args args
	}

	oneYearInSeconds := 31536000
	testCases := []test{
		{
			"1 year",
			args{
				ctype:             "bnb-a",
				initialTime:       time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral: c("bnb", 1000000000000),
				initialPrincipal:  c("usdx", 100000000000),
				timeElapsed:       oneYearInSeconds,
				expectedFees:      c("usdx", 5000000000),
			},
		},
		{
			"1 month",
			args{
				ctype:             "bnb-a",
				initialTime:       time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral: c("bnb", 1000000000000),
				initialPrincipal:  c("usdx", 100000000000),
				timeElapsed:       86400 * 30,
				expectedFees:      c("usdx", 401820189),
			},
		},
		{
			"7 seconds",
			args{
				ctype:             "bnb-a",
				initialTime:       time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral: c("bnb", 1000000000000),
				initialPrincipal:  c("usdx", 100000000000),
				timeElapsed:       7,
				expectedFees:      c("usdx", 1083),
			},
		},
		{
			"7 seconds - fees round to zero",
			args{
				ctype:             "bnb-a",
				initialTime:       time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialCollateral: c("bnb", 1000000000),
				initialPrincipal:  c("usdx", 10000000),
				timeElapsed:       7,
				expectedFees:      c("usdx", 0),
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)

			// setup account state
			_, addrs := app.GeneratePrivKeyAddressPairs(1)
			ak := suite.app.GetAccountKeeper()
			// setup the first account
			acc := ak.NewAccountWithAddress(suite.ctx, addrs[0])
			ak.SetAccount(suite.ctx, acc)
			bk := suite.app.GetBankKeeper()
			err := bk.MintCoins(suite.ctx, types.ModuleName, cs(tc.args.initialCollateral))
			suite.Require().NoError(err)
			err = bk.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addrs[0], cs(tc.args.initialCollateral))
			suite.Require().NoError(err)

			// setup pricefeed
			pk := suite.app.GetPriceFeedKeeper()
			_, err = pk.SetPrice(suite.ctx, sdk.AccAddress{}, "bnb:usd", d("17.25"), tc.args.initialTime.Add(time.Duration(int(time.Second)*tc.args.timeElapsed)))
			suite.Require().NoError(err)

			// setup cdp state
			suite.keeper.SetPreviousAccrualTime(suite.ctx, tc.args.ctype, suite.ctx.BlockTime())
			suite.keeper.SetInterestFactor(suite.ctx, tc.args.ctype, sdk.OneDec())
			err = suite.keeper.AddCdp(suite.ctx, addrs[0], tc.args.initialCollateral, tc.args.initialPrincipal, tc.args.ctype)
			suite.Require().NoError(err)

			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)
			err = suite.keeper.AccumulateInterest(suite.ctx, tc.args.ctype)
			suite.Require().NoError(err)

			cdp, found := suite.keeper.GetCDP(suite.ctx, tc.args.ctype, 1)
			suite.Require().True(found)

			newInterest := suite.keeper.CalculateNewInterest(suite.ctx, cdp)

			suite.Require().Equal(tc.args.expectedFees, newInterest)
		})
	}
}

func (suite *InterestTestSuite) TestSyncInterestForRiskyCDPs() {
	type args struct {
		ctype              string
		numberCdps         int
		slice              int
		initialCollateral  sdk.Coin
		minPrincipal       sdk.Coin
		principalIncrement sdk.Coin
		initialTime        time.Time
		timeElapsed        int
		expectedCDPs       int
	}

	type test struct {
		name string
		args args
	}

	oneYearInSeconds := 31536000
	testCases := []test{
		{
			"1 year",
			args{
				ctype:              "bnb-a",
				numberCdps:         20,
				slice:              10,
				initialCollateral:  c("bnb", 100000000000),
				minPrincipal:       c("usdx", 100000000),
				principalIncrement: c("usdx", 10000000),
				initialTime:        time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				timeElapsed:        oneYearInSeconds,
				expectedCDPs:       10,
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.ctx = suite.ctx.WithBlockTime(tc.args.initialTime)
			// setup account state
			_, addrs := app.GeneratePrivKeyAddressPairs(tc.args.numberCdps)
			ak := suite.app.GetAccountKeeper()
			bk := suite.app.GetBankKeeper()
			for _, addr := range addrs {
				acc := ak.NewAccountWithAddress(suite.ctx, addr)
				ak.SetAccount(suite.ctx, acc)
				err := bk.MintCoins(suite.ctx, types.ModuleName, cs(tc.args.initialCollateral))
				suite.Require().NoError(err)
				err = bk.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, cs(tc.args.initialCollateral))
				suite.Require().NoError(err)
			}
			// setup pricefeed
			pk := suite.app.GetPriceFeedKeeper()
			_, err := pk.SetPrice(suite.ctx, sdk.AccAddress{}, "bnb:usd", d("20.0"), tc.args.initialTime.Add(time.Duration(int(time.Second)*tc.args.timeElapsed)))
			suite.Require().NoError(err)

			// setup cdp state
			suite.keeper.SetPreviousAccrualTime(suite.ctx, tc.args.ctype, suite.ctx.BlockTime())
			suite.keeper.SetInterestFactor(suite.ctx, tc.args.ctype, sdk.OneDec())
			for j, addr := range addrs {
				initialPrincipal := tc.args.minPrincipal.Add(c("usdx", int64(j)*tc.args.principalIncrement.Amount.Int64()))
				err := suite.keeper.AddCdp(suite.ctx, addr, tc.args.initialCollateral, initialPrincipal, tc.args.ctype)
				suite.Require().NoError(err)
			}

			updatedBlockTime := suite.ctx.BlockTime().Add(time.Duration(int(time.Second) * tc.args.timeElapsed))
			suite.ctx = suite.ctx.WithBlockTime(updatedBlockTime)
			err = suite.keeper.AccumulateInterest(suite.ctx, tc.args.ctype)
			suite.Require().NoError(err)

			err = suite.keeper.SynchronizeInterestForRiskyCDPs(suite.ctx, i(int64(tc.args.slice)), sdk.MaxSortableDec, tc.args.ctype)
			suite.Require().NoError(err)

			cdpsUpdatedCount := 0

			for _, addr := range addrs {
				cdp, found := suite.keeper.GetCdpByOwnerAndCollateralType(suite.ctx, addr, tc.args.ctype)
				suite.Require().True(found)
				if cdp.FeesUpdated.Equal(suite.ctx.BlockTime()) {
					cdpsUpdatedCount += 1
				}
			}
			suite.Require().Equal(tc.args.expectedCDPs, cdpsUpdatedCount)
		})
	}
}

func TestInterestTestSuite(t *testing.T) {
	suite.Run(t, new(InterestTestSuite))
}
