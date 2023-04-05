package keeper_test

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"

	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/kava-labs/kava/app"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	"github.com/kava-labs/kava/x/cdp/keeper"
	"github.com/kava-labs/kava/x/cdp/types"
)

type SeizeTestSuite struct {
	suite.Suite

	keeper       keeper.Keeper
	addrs        []sdk.AccAddress
	app          app.TestApp
	cdps         types.CDPs
	ctx          sdk.Context
	liquidations liquidationTracker
}

type liquidationTracker struct {
	xrp  []uint64
	btc  []uint64
	debt int64
}

func (suite *SeizeTestSuite) SetupTest() {
	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now(), ChainID: "kavatest_1-1"})
	tracker := liquidationTracker{}
	coins := cs(c("btc", 100000000), c("xrp", 10000000000))
	_, addrs := app.GeneratePrivKeyAddressPairs(100)

	authGS := app.NewFundedGenStateWithSameCoins(tApp.AppCodec(), coins, addrs)
	tApp.InitializeFromGenesisStates(
		authGS,
		NewPricefeedGenStateMulti(tApp.AppCodec()),
		NewCDPGenStateMulti(tApp.AppCodec()),
	)
	suite.ctx = ctx
	suite.app = tApp
	suite.keeper = tApp.GetCDPKeeper()
	suite.cdps = types.CDPs{}
	suite.addrs = addrs
	suite.liquidations = tracker
}

func (suite *SeizeTestSuite) createCdps() {
	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})
	cdps := make(types.CDPs, 100)
	_, addrs := app.GeneratePrivKeyAddressPairs(100)
	tracker := liquidationTracker{}
	coins := cs(c("btc", 100000000), c("xrp", 10000000000))

	authGS := app.NewFundedGenStateWithSameCoins(tApp.AppCodec(), coins, addrs)
	tApp.InitializeFromGenesisStates(
		authGS,
		NewPricefeedGenStateMulti(tApp.AppCodec()),
		NewCDPGenStateMulti(tApp.AppCodec()),
	)

	suite.ctx = ctx
	suite.app = tApp
	suite.keeper = tApp.GetCDPKeeper()
	randSource := rand.New(rand.NewSource(int64(777)))
	for j := 0; j < 100; j++ {
		collateral := "xrp"
		amount := 10000000000
		debt := simulation.RandIntBetween(randSource, 750000000, 1249000000)
		if j%2 == 0 {
			collateral = "btc"
			amount = 100000000
			debt = simulation.RandIntBetween(randSource, 2700000000, 5332000000)
			if debt >= 4000000000 {
				tracker.btc = append(tracker.btc, uint64(j+1))
				tracker.debt += int64(debt)
			}
		} else {
			if debt >= 1000000000 {
				tracker.xrp = append(tracker.xrp, uint64(j+1))
				tracker.debt += int64(debt)
			}
		}
		err := suite.keeper.AddCdp(suite.ctx, addrs[j], c(collateral, int64(amount)), c("usdx", int64(debt)), collateral+"-a")
		suite.NoError(err)
		c, f := suite.keeper.GetCDP(suite.ctx, collateral+"-a", uint64(j+1))
		suite.True(f)
		cdps[j] = c
	}

	suite.cdps = cdps
	suite.addrs = addrs
	suite.liquidations = tracker
}

func (suite *SeizeTestSuite) setPrice(price sdk.Dec, market string) {
	pfKeeper := suite.app.GetPriceFeedKeeper()

	_, err := pfKeeper.SetPrice(suite.ctx, sdk.AccAddress{}, market, price, suite.ctx.BlockTime().Add(time.Hour*3))
	suite.NoError(err)
	err = pfKeeper.SetCurrentPrices(suite.ctx, market)
	suite.NoError(err)
	pp, err := pfKeeper.GetCurrentPrice(suite.ctx, market)
	suite.NoError(err)
	suite.Equal(price, pp.Price)
}

func (suite *SeizeTestSuite) TestSeizeCollateral() {
	suite.createCdps()
	ak := suite.app.GetAccountKeeper()
	bk := suite.app.GetBankKeeper()

	cdp, found := suite.keeper.GetCDP(suite.ctx, "xrp-a", uint64(2))
	suite.True(found)

	p := cdp.Principal.Amount
	cl := cdp.Collateral.Amount

	tpb := suite.keeper.GetTotalPrincipal(suite.ctx, "xrp-a", "usdx")
	err := suite.keeper.SeizeCollateral(suite.ctx, cdp)
	suite.NoError(err)

	tpa := suite.keeper.GetTotalPrincipal(suite.ctx, "xrp-a", "usdx")
	suite.Equal(tpb.Sub(tpa), p)

	auctionKeeper := suite.app.GetAuctionKeeper()

	_, found = auctionKeeper.GetAuction(suite.ctx, auctiontypes.DefaultNextAuctionID)
	suite.True(found)

	auctionMacc := ak.GetModuleAccount(suite.ctx, auctiontypes.ModuleName)
	suite.Equal(cs(c("debt", p.Int64()), c("xrp", cl.Int64())), bk.GetAllBalances(suite.ctx, auctionMacc.GetAddress()))

	acc := ak.GetAccount(suite.ctx, suite.addrs[1])
	suite.Equal(p.Int64(), bk.GetBalance(suite.ctx, acc.GetAddress(), "usdx").Amount.Int64())
	err = suite.keeper.WithdrawCollateral(suite.ctx, suite.addrs[1], suite.addrs[1], c("xrp", 10), "xrp-a")
	suite.Require().True(errors.Is(err, types.ErrCdpNotFound))
}

func (suite *SeizeTestSuite) TestSeizeCollateralMultiDeposit() {
	suite.createCdps()
	ak := suite.app.GetAccountKeeper()
	bk := suite.app.GetBankKeeper()

	_, found := suite.keeper.GetCDP(suite.ctx, "xrp-a", uint64(2))
	suite.True(found)

	err := suite.keeper.DepositCollateral(suite.ctx, suite.addrs[1], suite.addrs[0], c("xrp", 6999000000), "xrp-a")
	suite.NoError(err)

	cdp, found := suite.keeper.GetCDP(suite.ctx, "xrp-a", uint64(2))
	suite.True(found)

	deposits := suite.keeper.GetDeposits(suite.ctx, cdp.ID)
	suite.Equal(2, len(deposits))

	p := cdp.Principal.Amount
	cl := cdp.Collateral.Amount
	tpb := suite.keeper.GetTotalPrincipal(suite.ctx, "xrp-a", "usdx")
	err = suite.keeper.SeizeCollateral(suite.ctx, cdp)
	suite.NoError(err)

	tpa := suite.keeper.GetTotalPrincipal(suite.ctx, "xrp-a", "usdx")
	suite.Equal(tpb.Sub(tpa), p)

	auctionMacc := ak.GetModuleAccount(suite.ctx, auctiontypes.ModuleName)
	suite.Equal(cs(c("debt", p.Int64()), c("xrp", cl.Int64())), bk.GetAllBalances(suite.ctx, auctionMacc.GetAddress()))

	acc := ak.GetAccount(suite.ctx, suite.addrs[1])
	suite.Equal(p.Int64(), bk.GetBalance(suite.ctx, acc.GetAddress(), "usdx").Amount.Int64())
	err = suite.keeper.WithdrawCollateral(suite.ctx, suite.addrs[1], suite.addrs[1], c("xrp", 10), "xrp-a")
	suite.Require().True(errors.Is(err, types.ErrCdpNotFound))
}

func (suite *SeizeTestSuite) TestLiquidateCdps() {
	suite.createCdps()
	ak := suite.app.GetAccountKeeper()
	bk := suite.app.GetBankKeeper()
	acc := ak.GetModuleAccount(suite.ctx, types.ModuleName)

	originalXrpCollateral := bk.GetBalance(suite.ctx, acc.GetAddress(), "xrp").Amount
	suite.setPrice(d("0.2"), "xrp:usd")
	p, found := suite.keeper.GetCollateral(suite.ctx, "xrp-a")
	suite.True(found)

	err := suite.keeper.LiquidateCdps(suite.ctx, "xrp:usd", "xrp-a", p.LiquidationRatio, p.CheckCollateralizationIndexCount)
	suite.NoError(err)

	acc = ak.GetModuleAccount(suite.ctx, types.ModuleName)
	finalXrpCollateral := bk.GetBalance(suite.ctx, acc.GetAddress(), "xrp").Amount
	seizedXrpCollateral := originalXrpCollateral.Sub(finalXrpCollateral)
	xrpLiquidations := int(seizedXrpCollateral.Quo(i(10000000000)).Int64())
	suite.Equal(10, xrpLiquidations)
}

func (suite *SeizeTestSuite) TestApplyLiquidationPenalty() {
	penalty := suite.keeper.ApplyLiquidationPenalty(suite.ctx, "xrp-a", i(1000))
	suite.Equal(i(50), penalty)
	penalty = suite.keeper.ApplyLiquidationPenalty(suite.ctx, "btc-a", i(1000))
	suite.Equal(i(25), penalty)
	penalty = suite.keeper.ApplyLiquidationPenalty(suite.ctx, "xrp-a", i(675760172))
	suite.Equal(i(33788009), penalty)
	suite.Panics(func() { suite.keeper.ApplyLiquidationPenalty(suite.ctx, "lol-a", i(1000)) })
}

func (suite *SeizeTestSuite) TestKeeperLiquidation() {
	type args struct {
		ctype               string
		blockTime           time.Time
		initialPrice        sdk.Dec
		finalPrice          sdk.Dec
		finalTwapPrice      sdk.Dec
		collateral          sdk.Coin
		principal           sdk.Coin
		expectedKeeperCoins sdk.Coins              // additional coins (if any) the borrower address should have after successfully liquidating position
		expectedAuctions    []auctiontypes.Auction // the auctions we should expect to find have been started
	}

	type errArgs struct {
		expectLiquidate bool
		contains        string
	}

	type test struct {
		name    string
		args    args
		errArgs errArgs
	}

	// Set up auction constants
	layout := "2006-01-02T15:04:05.000Z"
	endTimeStr := "9000-01-01T00:00:00.000Z"
	endTime, _ := time.Parse(layout, endTimeStr)
	addr, _ := sdk.AccAddressFromBech32("kava1ze7y9qwdddejmy7jlw4cymqqlt2wh05yhwmrv2")

	testCases := []test{
		{
			"valid liquidation",
			args{
				ctype:               "btc-a",
				blockTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialPrice:        d("20000.00"),
				finalPrice:          d("19000.0"),
				finalTwapPrice:      d("19000.0"),
				collateral:          c("btc", 10000000),
				principal:           c("usdx", 1333330000),
				expectedKeeperCoins: cs(c("btc", 100100000), c("xrp", 10000000000)),
				expectedAuctions: []auctiontypes.Auction{
					&auctiontypes.CollateralAuction{
						BaseAuction: auctiontypes.BaseAuction{
							ID:              1,
							Initiator:       "liquidator",
							Lot:             c("btc", 9900000),
							Bidder:          nil,
							Bid:             c("usdx", 0),
							HasReceivedBids: false,
							EndTime:         endTime,
							MaxEndTime:      endTime,
						},
						CorrespondingDebt: c("debt", 1333330000),
						MaxBid:            c("usdx", 1366663250),
						LotReturns: auctiontypes.WeightedAddresses{
							Addresses: []sdk.AccAddress{addr},
							Weights:   []sdkmath.Int{sdkmath.NewInt(9900000)},
						},
					},
				},
			},
			errArgs{
				true,
				"",
			},
		},
		{
			"valid liquidation - twap market liquidateable but not spot",
			args{
				ctype:        "btc-a",
				blockTime:    time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialPrice: d("20000.00"),
				// spot price does not liquidates
				finalPrice: d("21000.0"),
				// twap / liquidation price does liquidate
				finalTwapPrice:      d("19000.0"),
				collateral:          c("btc", 10000000),
				principal:           c("usdx", 1333330000),
				expectedKeeperCoins: cs(c("btc", 100100000), c("xrp", 10000000000)),
				expectedAuctions: []auctiontypes.Auction{
					&auctiontypes.CollateralAuction{
						BaseAuction: auctiontypes.BaseAuction{
							ID:              1,
							Initiator:       "liquidator",
							Lot:             c("btc", 9900000),
							Bidder:          nil,
							Bid:             c("usdx", 0),
							HasReceivedBids: false,
							EndTime:         endTime,
							MaxEndTime:      endTime,
						},
						CorrespondingDebt: c("debt", 1333330000),
						MaxBid:            c("usdx", 1366663250),
						LotReturns: auctiontypes.WeightedAddresses{
							Addresses: []sdk.AccAddress{addr},
							Weights:   []sdkmath.Int{sdkmath.NewInt(9900000)},
						},
					},
				},
			},
			errArgs{
				true,
				"",
			},
		},
		{
			"invalid - not below collateralization ratio",
			args{
				ctype:               "btc-a",
				blockTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialPrice:        d("20000.00"),
				finalPrice:          d("21000.0"),
				finalTwapPrice:      d("21000.0"),
				collateral:          c("btc", 10000000),
				principal:           c("usdx", 1333330000),
				expectedKeeperCoins: cs(),
				expectedAuctions:    []auctiontypes.Auction{},
			},
			errArgs{
				false,
				"collateral ratio not below liquidation ratio",
			},
		},
		{
			"invalid - spot market liquidateable but not twap",
			args{
				ctype:        "btc-a",
				blockTime:    time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialPrice: d("20000.00"),
				// spot price liquidates
				finalPrice: d("19000.0"),
				// twap / liquidation price does not liquidate
				finalTwapPrice:      d("21000.0"),
				collateral:          c("btc", 10000000),
				principal:           c("usdx", 1333330000),
				expectedKeeperCoins: cs(),
				expectedAuctions:    []auctiontypes.Auction{},
			},
			errArgs{
				false,
				"collateral ratio not below liquidation ratio",
			},
		},
		{
			"invalid - collateralization ratio equal to liquidation ratio",
			args{
				ctype:               "xrp-a",
				blockTime:           time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				initialPrice:        d("1.00"), // we are allowed to create a cdp with an exact ratio
				finalPrice:          d("1.00"),
				finalTwapPrice:      d("1.00"), // and it should not be able to be liquidated
				collateral:          c("xrp", 100000000),
				principal:           c("usdx", 50000000),
				expectedKeeperCoins: cs(),
				expectedAuctions:    []auctiontypes.Auction{},
			},
			errArgs{
				false,
				"collateral ratio not below liquidation ratio",
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			spotMarket := fmt.Sprintf("%s:usd", tc.args.collateral.Denom)
			liquidationMarket := fmt.Sprintf("%s:30", spotMarket)

			// setup pricefeed
			pk := suite.app.GetPriceFeedKeeper()
			_, err := pk.SetPrice(suite.ctx, sdk.AccAddress{}, spotMarket, tc.args.initialPrice, suite.ctx.BlockTime().Add(time.Hour*24))
			suite.Require().NoError(err)
			err = pk.SetCurrentPrices(suite.ctx, spotMarket)
			suite.Require().NoError(err)

			// setup cdp state
			suite.keeper.SetPreviousAccrualTime(suite.ctx, tc.args.ctype, suite.ctx.BlockTime())
			suite.keeper.SetInterestFactor(suite.ctx, tc.args.ctype, sdk.OneDec())
			err = suite.keeper.AddCdp(suite.ctx, suite.addrs[0], tc.args.collateral, tc.args.principal, tc.args.ctype)
			suite.Require().NoError(err)

			// update pricefeed
			// spot market
			_, err = pk.SetPrice(suite.ctx, sdk.AccAddress{}, spotMarket, tc.args.finalPrice, suite.ctx.BlockTime().Add(time.Hour*24))
			suite.Require().NoError(err)
			// liquidate market
			_, err = pk.SetPrice(suite.ctx, sdk.AccAddress{}, liquidationMarket, tc.args.finalTwapPrice, suite.ctx.BlockTime().Add(time.Hour*24))
			suite.Require().NoError(err)

			err = pk.SetCurrentPrices(suite.ctx, spotMarket)
			suite.Require().NoError(err)
			err = pk.SetCurrentPrices(suite.ctx, liquidationMarket)
			suite.Require().NoError(err)

			_, found := suite.keeper.GetCdpByOwnerAndCollateralType(suite.ctx, suite.addrs[0], tc.args.ctype)
			suite.Require().True(found)

			err = suite.keeper.AttemptKeeperLiquidation(suite.ctx, suite.addrs[1], suite.addrs[0], tc.args.ctype)

			if tc.errArgs.expectLiquidate {
				suite.Require().NoError(err)

				_, found = suite.keeper.GetCdpByOwnerAndCollateralType(suite.ctx, suite.addrs[0], tc.args.ctype)
				suite.Require().False(found)

				ak := suite.app.GetAuctionKeeper()
				auctions := ak.GetAllAuctions(suite.ctx)
				suite.Require().Equal(tc.args.expectedAuctions, auctions)

				ack := suite.app.GetAccountKeeper()
				bk := suite.app.GetBankKeeper()
				keeper := ack.GetAccount(suite.ctx, suite.addrs[1])
				suite.Require().Equal(tc.args.expectedKeeperCoins, bk.GetAllBalances(suite.ctx, keeper.GetAddress()))
			} else {
				suite.Require().Error(err)
				suite.Require().True(strings.Contains(err.Error(), tc.errArgs.contains))
			}
		})
	}
}

func (suite *SeizeTestSuite) TestBeginBlockerLiquidation() {
	type args struct {
		ctype            string
		blockTime        time.Time
		initialPrice     sdk.Dec
		finalPrice       sdk.Dec
		collaterals      sdk.Coins
		principals       sdk.Coins
		expectedAuctions []auctiontypes.Auction // the auctions we should expect to find have been started
	}
	type errArgs struct {
		expectLiquidate bool
		contains        string
	}
	type test struct {
		name    string
		args    args
		errArgs errArgs
	}
	// Set up auction constants
	layout := "2006-01-02T15:04:05.000Z"
	endTimeStr := "9000-01-01T00:00:00.000Z"
	endTime, _ := time.Parse(layout, endTimeStr)
	addr, _ := sdk.AccAddressFromBech32("kava1ze7y9qwdddejmy7jlw4cymqqlt2wh05yhwmrv2")

	testCases := []test{
		{
			"1 liquidation",
			args{
				"btc-a",
				time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				d("20000.00"),
				d("10000.00"),
				sdk.Coins{c("btc", 10000000), c("btc", 10000000)},
				sdk.Coins{c("usdx", 1000000000), c("usdx", 500000000)},
				[]auctiontypes.Auction{
					&auctiontypes.CollateralAuction{
						BaseAuction: auctiontypes.BaseAuction{
							ID:              1,
							Initiator:       "liquidator",
							Lot:             c("btc", 10000000),
							Bidder:          nil,
							Bid:             c("usdx", 0),
							HasReceivedBids: false,
							EndTime:         endTime,
							MaxEndTime:      endTime,
						},
						CorrespondingDebt: c("debt", 1000000000),
						MaxBid:            c("usdx", 1025000000),
						LotReturns: auctiontypes.WeightedAddresses{
							Addresses: []sdk.AccAddress{addr},
							Weights:   []sdkmath.Int{sdkmath.NewInt(10000000)},
						},
					},
				},
			},
			errArgs{
				true,
				"",
			},
		},
		{
			"no liquidation",
			args{
				"btc-a",
				time.Date(2020, 12, 15, 14, 0, 0, 0, time.UTC),
				d("20000.00"),
				d("10000.00"),
				sdk.Coins{c("btc", 10000000), c("btc", 10000000)},
				sdk.Coins{c("usdx", 500000000), c("usdx", 500000000)},
				[]auctiontypes.Auction{},
			},
			errArgs{
				false,
				"collateral ratio not below liquidation ratio",
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			// setup pricefeed
			pk := suite.app.GetPriceFeedKeeper()
			_, err := pk.SetPrice(suite.ctx, sdk.AccAddress{}, "btc:usd", tc.args.initialPrice, suite.ctx.BlockTime().Add(time.Hour*24))
			suite.Require().NoError(err)
			err = pk.SetCurrentPrices(suite.ctx, "btc:usd")
			suite.Require().NoError(err)

			// setup cdp state
			suite.keeper.SetPreviousAccrualTime(suite.ctx, tc.args.ctype, suite.ctx.BlockTime())
			suite.keeper.SetInterestFactor(suite.ctx, tc.args.ctype, sdk.OneDec())

			for idx, col := range tc.args.collaterals {
				err := suite.keeper.AddCdp(suite.ctx, suite.addrs[idx], col, tc.args.principals[idx], tc.args.ctype)
				suite.Require().NoError(err)
			}

			// update pricefeed
			_, err = pk.SetPrice(suite.ctx, sdk.AccAddress{}, "btc:usd", tc.args.finalPrice, suite.ctx.BlockTime().Add(time.Hour*24))
			suite.Require().NoError(err)
			err = pk.SetCurrentPrices(suite.ctx, "btc:usd")
			suite.Require().NoError(err)

			_ = suite.app.BeginBlocker(suite.ctx, abci.RequestBeginBlock{Header: suite.ctx.BlockHeader()})
			ak := suite.app.GetAuctionKeeper()
			auctions := ak.GetAllAuctions(suite.ctx)
			if tc.errArgs.expectLiquidate {
				suite.Require().Equal(tc.args.expectedAuctions, auctions)
				for _, a := range auctions {
					ca := a.(*auctiontypes.CollateralAuction)
					_, found := suite.keeper.GetCdpByOwnerAndCollateralType(suite.ctx, ca.LotReturns.Addresses[0], tc.args.ctype)
					suite.Require().False(found)
				}
			} else {
				suite.Require().Equal(0, len(auctions))
				for idx := range tc.args.collaterals {
					_, found := suite.keeper.GetCdpByOwnerAndCollateralType(suite.ctx, suite.addrs[idx], tc.args.ctype)
					suite.Require().True(found)
				}
			}
		})
	}
}

func TestSeizeTestSuite(t *testing.T) {
	suite.Run(t, new(SeizeTestSuite))
}
