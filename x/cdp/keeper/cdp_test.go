package keeper_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/cdp/keeper"
	"github.com/kava-labs/kava/x/cdp/types"
)

type CdpTestSuite struct {
	suite.Suite

	keeper keeper.Keeper
	app    app.TestApp
	ctx    sdk.Context
}

func (suite *CdpTestSuite) SetupTest() {
	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})
	tApp.InitializeFromGenesisStates(
		NewPricefeedGenStateMulti(tApp.AppCodec()),
		NewCDPGenStateMulti(tApp.AppCodec()),
	)
	keeper := tApp.GetCDPKeeper()
	suite.app = tApp
	suite.ctx = ctx
	suite.keeper = keeper
}

func (suite *CdpTestSuite) TestAddCdp() {
	_, addrs := app.GeneratePrivKeyAddressPairs(2)
	ak := suite.app.GetAccountKeeper()
	acc := ak.NewAccountWithAddress(suite.ctx, addrs[0])
	err := suite.app.FundAccount(suite.ctx, acc.GetAddress(), cs(c("xrp", 200000000), c("btc", 500000000)))
	suite.Require().NoError(err)

	ak.SetAccount(suite.ctx, acc)
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("xrp", 200000000), c("usdx", 10000000), "btc-a")
	suite.Require().True(errors.Is(err, types.ErrInvalidCollateral))
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("xrp", 200000000), c("usdx", 26000000), "xrp-a")
	suite.Require().True(errors.Is(err, types.ErrInvalidCollateralRatio))
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("xrp", 500000000), c("usdx", 26000000), "xrp-a")
	suite.Error(err) // insufficient balance
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("xrp", 200000000), c("xusd", 10000000), "xrp-a")
	suite.Require().True(errors.Is(err, types.ErrDebtNotSupported))

	acc2 := ak.NewAccountWithAddress(suite.ctx, addrs[1])
	err = suite.app.FundAccount(suite.ctx, acc2.GetAddress(), cs(c("btc", 500000000000)))
	suite.Require().NoError(err)

	ak.SetAccount(suite.ctx, acc2)
	err = suite.keeper.AddCdp(suite.ctx, addrs[1], c("btc", 500000000000), c("usdx", 500000000001), "btc-a")
	suite.Require().True(errors.Is(err, types.ErrExceedsDebtLimit))

	ctx := suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(time.Hour * 2))
	pk := suite.app.GetPriceFeedKeeper()
	err = pk.SetCurrentPrices(ctx, "xrp:usd")
	suite.Error(err)
	ok := suite.keeper.UpdatePricefeedStatus(ctx, "xrp:usd")
	suite.False(ok)
	err = suite.keeper.AddCdp(ctx, addrs[0], c("xrp", 100000000), c("usdx", 10000000), "xrp-a")
	suite.Require().True(errors.Is(err, types.ErrPricefeedDown))

	err = pk.SetCurrentPrices(suite.ctx, "xrp:usd")
	ok = suite.keeper.UpdatePricefeedStatus(suite.ctx, "xrp:usd")
	suite.True(ok)
	suite.NoError(err)
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("xrp", 100000000), c("usdx", 10000000), "xrp-a")
	suite.NoError(err)
	id := suite.keeper.GetNextCdpID(suite.ctx)
	suite.Equal(uint64(2), id)
	tp := suite.keeper.GetTotalPrincipal(suite.ctx, "xrp-a", "usdx")
	suite.Equal(i(10000000), tp)

	bk := suite.app.GetBankKeeper()

	macc := ak.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.Equal(cs(c("debt", 10000000), c("xrp", 100000000)), bk.GetAllBalances(suite.ctx, macc.GetAddress()))
	acc = ak.GetAccount(suite.ctx, addrs[0])
	suite.Equal(cs(c("usdx", 10000000), c("xrp", 100000000), c("btc", 500000000)), bk.GetAllBalances(suite.ctx, acc.GetAddress()))

	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("btc", 500000000), c("usdx", 26667000000), "btc-a")
	suite.Require().True(errors.Is(err, types.ErrInvalidCollateralRatio))

	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("btc", 500000000), c("usdx", 100000000), "btc-a")
	suite.NoError(err)
	id = suite.keeper.GetNextCdpID(suite.ctx)
	suite.Equal(uint64(3), id)
	tp = suite.keeper.GetTotalPrincipal(suite.ctx, "btc-a", "usdx")
	suite.Equal(i(100000000), tp)
	macc = ak.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.Equal(cs(c("debt", 110000000), c("xrp", 100000000), c("btc", 500000000)), bk.GetAllBalances(suite.ctx, macc.GetAddress()))
	acc = ak.GetAccount(suite.ctx, addrs[0])
	suite.Equal(cs(c("usdx", 110000000), c("xrp", 100000000)), bk.GetAllBalances(suite.ctx, acc.GetAddress()))

	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("lol", 100), c("usdx", 10), "lol-a")
	suite.Require().True(errors.Is(err, types.ErrCollateralNotSupported))
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], c("xrp", 100), c("usdx", 10), "xrp-a")
	suite.Require().True(errors.Is(err, types.ErrCdpAlreadyExists))
}

func (suite *CdpTestSuite) TestGetCollateral() {
	_, found := suite.keeper.GetCollateral(suite.ctx, "lol-a")
	suite.False(found)
	_, found = suite.keeper.GetCollateral(suite.ctx, "xrp-a")
	suite.True(found)
}

func (suite *CdpTestSuite) TestGetDebtDenom() {
	suite.Panics(func() { suite.keeper.SetDebtDenom(suite.ctx, "") })
	t := suite.keeper.GetDebtDenom(suite.ctx)
	suite.Equal("debt", t)
	suite.keeper.SetDebtDenom(suite.ctx, "lol")
	t = suite.keeper.GetDebtDenom(suite.ctx)
	suite.Equal("lol", t)
}

func (suite *CdpTestSuite) TestGetNextCdpID() {
	id := suite.keeper.GetNextCdpID(suite.ctx)
	suite.Equal(types.DefaultCdpStartingID, id)
}

func (suite *CdpTestSuite) TestGetSetCdp() {
	_, addrs := app.GeneratePrivKeyAddressPairs(1)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], c("xrp", 1), "xrp-a", c("usdx", 1), tmtime.Canonical(time.Now()), sdk.OneDec())
	err := suite.keeper.SetCDP(suite.ctx, cdp)
	suite.NoError(err)

	t, found := suite.keeper.GetCDP(suite.ctx, "xrp-a", types.DefaultCdpStartingID)
	suite.True(found)
	suite.Equal(cdp, t)
	_, found = suite.keeper.GetCDP(suite.ctx, "xrp-a", uint64(2))
	suite.False(found)
	suite.NoError(suite.keeper.DeleteCDP(suite.ctx, cdp))
	_, found = suite.keeper.GetCDP(suite.ctx, "btc-a", types.DefaultCdpStartingID)
	suite.False(found)
}

func (suite *CdpTestSuite) TestGetSetCdpId() {
	_, addrs := app.GeneratePrivKeyAddressPairs(2)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], c("xrp", 1), "xrp-a", c("usdx", 1), tmtime.Canonical(time.Now()), sdk.OneDec())
	err := suite.keeper.SetCDP(suite.ctx, cdp)
	suite.NoError(err)
	suite.keeper.IndexCdpByOwner(suite.ctx, cdp)
	id, found := suite.keeper.GetCdpID(suite.ctx, addrs[0], "xrp-a")
	suite.True(found)
	suite.Equal(types.DefaultCdpStartingID, id)
	_, found = suite.keeper.GetCdpID(suite.ctx, addrs[0], "lol-a")
	suite.False(found)
	_, found = suite.keeper.GetCdpID(suite.ctx, addrs[1], "xrp-a")
	suite.False(found)
}

func (suite *CdpTestSuite) TestGetSetCdpByOwnerAndCollateralType() {
	_, addrs := app.GeneratePrivKeyAddressPairs(2)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], c("xrp", 1), "xrp-a", c("usdx", 1), tmtime.Canonical(time.Now()), sdk.OneDec())
	err := suite.keeper.SetCDP(suite.ctx, cdp)
	suite.NoError(err)
	suite.keeper.IndexCdpByOwner(suite.ctx, cdp)
	t, found := suite.keeper.GetCdpByOwnerAndCollateralType(suite.ctx, addrs[0], "xrp-a")
	suite.True(found)
	suite.Equal(cdp, t)
	_, found = suite.keeper.GetCdpByOwnerAndCollateralType(suite.ctx, addrs[0], "lol-a")
	suite.False(found)
	_, found = suite.keeper.GetCdpByOwnerAndCollateralType(suite.ctx, addrs[1], "xrp-a")
	suite.False(found)
	suite.NotPanics(func() { suite.keeper.IndexCdpByOwner(suite.ctx, cdp) })
}

func (suite *CdpTestSuite) TestCalculateCollateralToDebtRatio() {
	_, addrs := app.GeneratePrivKeyAddressPairs(1)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], c("xrp", 3), "xrp-a", c("usdx", 1), tmtime.Canonical(time.Now()), sdk.OneDec())
	cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdp.Collateral, cdp.Type, cdp.Principal)
	suite.Equal(sdk.MustNewDecFromStr("3.0"), cr)
	cdp = types.NewCDP(types.DefaultCdpStartingID, addrs[0], c("xrp", 1), "xrp-a", c("usdx", 2), tmtime.Canonical(time.Now()), sdk.OneDec())
	cr = suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdp.Collateral, cdp.Type, cdp.Principal)
	suite.Equal(sdk.MustNewDecFromStr("0.5"), cr)
}

func (suite *CdpTestSuite) TestSetCdpByCollateralRatio() {
	_, addrs := app.GeneratePrivKeyAddressPairs(1)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], c("xrp", 3), "xrp-a", c("usdx", 1), tmtime.Canonical(time.Now()), sdk.OneDec())
	cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdp.Collateral, cdp.Type, cdp.Principal)
	suite.NotPanics(func() { suite.keeper.IndexCdpByCollateralRatio(suite.ctx, cdp.Type, cdp.ID, cr) })
}

func (suite *CdpTestSuite) TestIterateCdps() {
	cdps := cdps()
	for _, c := range cdps {
		err := suite.keeper.SetCDP(suite.ctx, c)
		suite.NoError(err)
		suite.keeper.IndexCdpByOwner(suite.ctx, c)
		cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, c.Collateral, c.Type, c.Principal)
		suite.keeper.IndexCdpByCollateralRatio(suite.ctx, c.Type, c.ID, cr)
	}
	t := suite.keeper.GetAllCdps(suite.ctx)
	suite.Equal(4, len(t))
}

func (suite *CdpTestSuite) TestIterateCdpsByCollateralType() {
	cdps := cdps()
	for _, c := range cdps {
		err := suite.keeper.SetCDP(suite.ctx, c)
		suite.NoError(err)
		suite.keeper.IndexCdpByOwner(suite.ctx, c)
		cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, c.Collateral, c.Type, c.Principal)
		suite.keeper.IndexCdpByCollateralRatio(suite.ctx, c.Type, c.ID, cr)
	}
	xrpCdps := suite.keeper.GetAllCdpsByCollateralType(suite.ctx, "xrp-a")
	suite.Equal(3, len(xrpCdps))
	btcCdps := suite.keeper.GetAllCdpsByCollateralType(suite.ctx, "btc-a")
	suite.Equal(1, len(btcCdps))
	suite.NoError(suite.keeper.DeleteCDP(suite.ctx, cdps[0]))
	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[0])
	xrpCdps = suite.keeper.GetAllCdpsByCollateralType(suite.ctx, "xrp-a")
	suite.Equal(2, len(xrpCdps))
	suite.NoError(suite.keeper.DeleteCDP(suite.ctx, cdps[1]))
	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[1])
	ids, found := suite.keeper.GetCdpIdsByOwner(suite.ctx, cdps[1].Owner)
	suite.True(found)
	suite.Equal(1, len(ids))
	suite.Equal(uint64(3), ids[0])
}

func (suite *CdpTestSuite) TestIterateCdpsByCollateralRatio() {
	cdps := cdps()
	for _, c := range cdps {
		err := suite.keeper.SetCDP(suite.ctx, c)
		suite.NoError(err)
		suite.keeper.IndexCdpByOwner(suite.ctx, c)
		cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, c.Collateral, c.Type, c.Principal)
		suite.keeper.IndexCdpByCollateralRatio(suite.ctx, c.Type, c.ID, cr)
	}
	xrpCdps := suite.keeper.GetAllCdpsByCollateralTypeAndRatio(suite.ctx, "xrp-a", d("1.25"))
	suite.Equal(0, len(xrpCdps))
	xrpCdps = suite.keeper.GetAllCdpsByCollateralTypeAndRatio(suite.ctx, "xrp-a", d("1.25").Add(sdk.SmallestDec()))
	suite.Equal(1, len(xrpCdps))
	xrpCdps = suite.keeper.GetAllCdpsByCollateralTypeAndRatio(suite.ctx, "xrp-a", d("2.0").Add(sdk.SmallestDec()))
	suite.Equal(2, len(xrpCdps))
	xrpCdps = suite.keeper.GetAllCdpsByCollateralTypeAndRatio(suite.ctx, "xrp-a", d("100.0").Add(sdk.SmallestDec()))
	suite.Equal(3, len(xrpCdps))

	suite.NoError(suite.keeper.DeleteCDP(suite.ctx, cdps[0]))

	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[0])
	cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdps[0].Collateral, cdps[0].Type, cdps[0].Principal)
	suite.keeper.RemoveCdpCollateralRatioIndex(suite.ctx, cdps[0].Type, cdps[0].ID, cr)
	xrpCdps = suite.keeper.GetAllCdpsByCollateralTypeAndRatio(suite.ctx, "xrp-a", d("2.0").Add(sdk.SmallestDec()))
	suite.Equal(1, len(xrpCdps))
}

func (suite *CdpTestSuite) TestValidateCollateral() {
	c := sdk.NewCoin("xrp", sdkmath.NewInt(1))
	err := suite.keeper.ValidateCollateral(suite.ctx, c, "xrp-a")
	suite.NoError(err)
	c = sdk.NewCoin("lol", sdkmath.NewInt(1))
	err = suite.keeper.ValidateCollateral(suite.ctx, c, "lol-a")
	suite.Require().True(errors.Is(err, types.ErrCollateralNotSupported))
}

func (suite *CdpTestSuite) TestValidatePrincipal() {
	d := sdk.NewCoin("usdx", sdkmath.NewInt(10000000))
	err := suite.keeper.ValidatePrincipalAdd(suite.ctx, d)
	suite.NoError(err)
	d = sdk.NewCoin("xusd", sdkmath.NewInt(1))
	err = suite.keeper.ValidatePrincipalAdd(suite.ctx, d)
	suite.Require().True(errors.Is(err, types.ErrDebtNotSupported))
	d = sdk.NewCoin("usdx", sdkmath.NewInt(1000000000001))
	err = suite.keeper.ValidateDebtLimit(suite.ctx, "xrp-a", d)
	suite.Require().True(errors.Is(err, types.ErrExceedsDebtLimit))
	d = sdk.NewCoin("usdx", sdkmath.NewInt(100000000))
	err = suite.keeper.ValidateDebtLimit(suite.ctx, "xrp-a", d)
	suite.NoError(err)
}

func (suite *CdpTestSuite) TestCalculateCollateralizationRatio() {
	c := cdps()[1]
	err := suite.keeper.SetCDP(suite.ctx, c)
	suite.NoError(err)
	suite.keeper.IndexCdpByOwner(suite.ctx, c)
	cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, c.Collateral, c.Type, c.Principal)
	suite.keeper.IndexCdpByCollateralRatio(suite.ctx, c.Type, c.ID, cr)
	cr, err = suite.keeper.CalculateCollateralizationRatio(suite.ctx, c.Collateral, c.Type, c.Principal, c.AccumulatedFees, "spot")
	suite.NoError(err)
	suite.Equal(d("2.5"), cr)
	c.AccumulatedFees = sdk.NewCoin("usdx", i(10000000))
	cr, err = suite.keeper.CalculateCollateralizationRatio(suite.ctx, c.Collateral, c.Type, c.Principal, c.AccumulatedFees, "spot")
	suite.NoError(err)
	suite.Equal(d("1.25"), cr)
}

func (suite *CdpTestSuite) TestMintBurnDebtCoins() {
	cd := cdps()[1]
	err := suite.keeper.MintDebtCoins(suite.ctx, types.ModuleName, suite.keeper.GetDebtDenom(suite.ctx), cd.Principal)
	suite.NoError(err)
	suite.Require().Panics(func() {
		_ = suite.keeper.MintDebtCoins(suite.ctx, "notamodule", suite.keeper.GetDebtDenom(suite.ctx), cd.Principal)
	})

	ak := suite.app.GetAccountKeeper()
	bk := suite.app.GetBankKeeper()
	acc := ak.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.Equal(cs(c("debt", 10000000)), bk.GetAllBalances(suite.ctx, acc.GetAddress()))

	err = suite.keeper.BurnDebtCoins(suite.ctx, types.ModuleName, suite.keeper.GetDebtDenom(suite.ctx), cd.Principal)
	suite.NoError(err)
	suite.Require().Panics(func() {
		_ = suite.keeper.BurnDebtCoins(suite.ctx, "notamodule", suite.keeper.GetDebtDenom(suite.ctx), cd.Principal)
	})

	acc = ak.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.Equal(sdk.Coins{}, bk.GetAllBalances(suite.ctx, acc.GetAddress()))
}

func (suite *CdpTestSuite) TestCdpOwnerIndex() {
	cdps := cdps()

	owner_1 := cdps[0].Owner
	owner_2 := cdps[1].Owner

	cdpIds_1, found := suite.keeper.GetCdpIdsByOwner(suite.ctx, owner_1)
	suite.Require().False(found)
	cdpIds_2, found := suite.keeper.GetCdpIdsByOwner(suite.ctx, owner_2)
	suite.Require().False(found)

	suite.Require().Equal(0, len(cdpIds_1))
	suite.Require().Equal(0, len(cdpIds_2))

	suite.keeper.IndexCdpByOwner(suite.ctx, cdps[2])
	suite.keeper.IndexCdpByOwner(suite.ctx, cdps[1])
	suite.keeper.IndexCdpByOwner(suite.ctx, cdps[0])

	expectedCdpIds, found := suite.keeper.GetCdpIdsByOwner(suite.ctx, owner_1)
	suite.Require().True(found)
	suite.Require().Equal([]uint64{cdps[0].ID}, expectedCdpIds)

	expectedCdpIds, found = suite.keeper.GetCdpIdsByOwner(suite.ctx, owner_2)
	suite.Require().True(found)
	suite.Require().Equal([]uint64{cdps[1].ID, cdps[2].ID}, expectedCdpIds)

	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[0])
	expectedCdpIds, found = suite.keeper.GetCdpIdsByOwner(suite.ctx, owner_1)
	suite.Require().False(found)
	suite.Require().Equal([]uint64{}, expectedCdpIds)

	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[1])
	expectedCdpIds, found = suite.keeper.GetCdpIdsByOwner(suite.ctx, owner_2)
	suite.Require().True(found)
	suite.Require().Equal([]uint64{cdps[2].ID}, expectedCdpIds)

	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[2])
	expectedCdpIds, found = suite.keeper.GetCdpIdsByOwner(suite.ctx, owner_2)
	suite.Require().False(found)
	suite.Require().Equal([]uint64{}, expectedCdpIds)
}

func (suite *CdpTestSuite) TestMarketStatus() {
	suite.keeper.SetMarketStatus(suite.ctx, "ukava:usd", true)
	status := suite.keeper.GetMarketStatus(suite.ctx, "ukava:usd")
	suite.Require().True(status)
	suite.keeper.SetMarketStatus(suite.ctx, "ukava:usd", false)
	status = suite.keeper.GetMarketStatus(suite.ctx, "ukava:usd")
	suite.Require().False(status)
	suite.keeper.SetMarketStatus(suite.ctx, "ukava:usd", true)
	status = suite.keeper.GetMarketStatus(suite.ctx, "ukava:usd")
	suite.Require().True(status)

	status = suite.keeper.GetMarketStatus(suite.ctx, "unknown:usd")
	suite.Require().False(status)

	suite.keeper.SetMarketStatus(suite.ctx, "btc:usd", false)
	status = suite.keeper.GetMarketStatus(suite.ctx, "btc:usd")
	suite.Require().False(status)
	suite.keeper.SetMarketStatus(suite.ctx, "btc:usd", true)
	status = suite.keeper.GetMarketStatus(suite.ctx, "btc:usd")
	suite.Require().True(status)
	suite.keeper.SetMarketStatus(suite.ctx, "btc:usd", false)
	status = suite.keeper.GetMarketStatus(suite.ctx, "btc:usd")
	suite.Require().False(status)
}

func TestCdpTestSuite(t *testing.T) {
	suite.Run(t, new(CdpTestSuite))
}
