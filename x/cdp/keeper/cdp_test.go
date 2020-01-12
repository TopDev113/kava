package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/cdp/keeper"
	"github.com/kava-labs/kava/x/cdp/types"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
	tmtime "github.com/tendermint/tendermint/types/time"
)

type CdpTestSuite struct {
	suite.Suite

	keeper keeper.Keeper
	app    app.TestApp
	ctx    sdk.Context
}

func (suite *CdpTestSuite) SetupTest() {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, abci.Header{Height: 1, Time: tmtime.Now()})
	tApp.InitializeFromGenesisStates(
		NewPricefeedGenStateMulti(),
		NewCDPGenStateMulti(),
	)
	keeper := tApp.GetCDPKeeper()
	suite.app = tApp
	suite.ctx = ctx
	suite.keeper = keeper
	return
}

func (suite *CdpTestSuite) TestAddCdp() {
	_, addrs := app.GeneratePrivKeyAddressPairs(1)
	ak := suite.app.GetAccountKeeper()
	acc := ak.NewAccountWithAddress(suite.ctx, addrs[0])
	acc.SetCoins(cs(c("xrp", 200000000), c("btc", 500000000)))
	ak.SetAccount(suite.ctx, acc)
	err := suite.keeper.AddCdp(suite.ctx, addrs[0], cs(c("xrp", 200000000)), cs(c("usdx", 26000000)))
	suite.Equal(types.CodeInvalidCollateralRatio, err.Result().Code)
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], cs(c("xrp", 500000000)), cs(c("usdx", 26000000)))
	suite.Error(err)
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], cs(c("xrp", 200000000)), cs(c("xusd", 10000000)))
	suite.Equal(types.CodeDebtNotSupported, err.Result().Code)
	ctx := suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(time.Hour * 2))
	pk := suite.app.GetPriceFeedKeeper()
	_ = pk.SetCurrentPrices(ctx, "xrp:usd")
	err = suite.keeper.AddCdp(ctx, addrs[0], cs(c("xrp", 100000000)), cs(c("usdx", 10000000)))
	suite.Error(err)

	_ = pk.SetCurrentPrices(suite.ctx, "xrp:usd")
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], cs(c("xrp", 100000000)), cs(c("usdx", 10000000)))
	suite.NoError(err)
	id := suite.keeper.GetNextCdpID(suite.ctx)
	suite.Equal(uint64(2), id)
	tp := suite.keeper.GetTotalPrincipal(suite.ctx, "xrp", "usdx")
	suite.Equal(i(10000000), tp)
	sk := suite.app.GetSupplyKeeper()
	macc := sk.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.Equal(cs(c("debt", 10000000), c("xrp", 100000000)), macc.GetCoins())
	acc = ak.GetAccount(suite.ctx, addrs[0])
	suite.Equal(cs(c("usdx", 10000000), c("xrp", 100000000), c("btc", 500000000)), acc.GetCoins())

	err = suite.keeper.AddCdp(suite.ctx, addrs[0], cs(c("btc", 500000000)), cs(c("usdx", 26667000000)))
	suite.Equal(types.CodeInvalidCollateralRatio, err.Result().Code)

	err = suite.keeper.AddCdp(suite.ctx, addrs[0], cs(c("btc", 500000000)), cs(c("usdx", 100000000)))
	suite.NoError(err)
	id = suite.keeper.GetNextCdpID(suite.ctx)
	suite.Equal(uint64(3), id)
	tp = suite.keeper.GetTotalPrincipal(suite.ctx, "btc", "usdx")
	suite.Equal(i(100000000), tp)
	macc = sk.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.Equal(cs(c("debt", 110000000), c("xrp", 100000000), c("btc", 500000000)), macc.GetCoins())
	acc = ak.GetAccount(suite.ctx, addrs[0])
	suite.Equal(cs(c("usdx", 110000000), c("xrp", 100000000)), acc.GetCoins())

	err = suite.keeper.AddCdp(suite.ctx, addrs[0], cs(c("lol", 100)), cs(c("usdx", 10)))
	suite.Equal(types.CodeCollateralNotSupported, err.Result().Code)
	err = suite.keeper.AddCdp(suite.ctx, addrs[0], cs(c("xrp", 100)), cs(c("usdx", 10)))
	suite.Equal(types.CodeCdpAlreadyExists, err.Result().Code)
}

func (suite *CdpTestSuite) TestGetSetDenomByte() {
	_, found := suite.keeper.GetDenomPrefix(suite.ctx, "lol")
	suite.False(found)
	db, found := suite.keeper.GetDenomPrefix(suite.ctx, "xrp")
	suite.True(found)
	suite.Equal(byte(0x20), db)
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
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], cs(c("xrp", 1)), cs(c("usdx", 1)), tmtime.Canonical(time.Now()))
	suite.keeper.SetCDP(suite.ctx, cdp)
	t, found := suite.keeper.GetCDP(suite.ctx, "xrp", types.DefaultCdpStartingID)
	suite.True(found)
	suite.Equal(cdp, t)
	_, found = suite.keeper.GetCDP(suite.ctx, "xrp", uint64(2))
	suite.False(found)
	suite.keeper.DeleteCDP(suite.ctx, cdp)
	_, found = suite.keeper.GetCDP(suite.ctx, "btc", types.DefaultCdpStartingID)
	suite.False(found)
}

func (suite *CdpTestSuite) TestGetSetCdpId() {
	_, addrs := app.GeneratePrivKeyAddressPairs(2)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], cs(c("xrp", 1)), cs(c("usdx", 1)), tmtime.Canonical(time.Now()))
	suite.keeper.SetCDP(suite.ctx, cdp)
	suite.keeper.IndexCdpByOwner(suite.ctx, cdp)
	id, found := suite.keeper.GetCdpID(suite.ctx, addrs[0], "xrp")
	suite.True(found)
	suite.Equal(types.DefaultCdpStartingID, id)
	_, found = suite.keeper.GetCdpID(suite.ctx, addrs[0], "lol")
	suite.False(found)
	_, found = suite.keeper.GetCdpID(suite.ctx, addrs[1], "xrp")
	suite.False(found)
}

func (suite *CdpTestSuite) TestGetSetCdpByOwnerAndDenom() {
	_, addrs := app.GeneratePrivKeyAddressPairs(2)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], cs(c("xrp", 1)), cs(c("usdx", 1)), tmtime.Canonical(time.Now()))
	suite.keeper.SetCDP(suite.ctx, cdp)
	suite.keeper.IndexCdpByOwner(suite.ctx, cdp)
	t, found := suite.keeper.GetCdpByOwnerAndDenom(suite.ctx, addrs[0], "xrp")
	suite.True(found)
	suite.Equal(cdp, t)
	_, found = suite.keeper.GetCdpByOwnerAndDenom(suite.ctx, addrs[0], "lol")
	suite.False(found)
	_, found = suite.keeper.GetCdpByOwnerAndDenom(suite.ctx, addrs[1], "xrp")
	suite.False(found)
	suite.NotPanics(func() { suite.keeper.IndexCdpByOwner(suite.ctx, cdp) })
}

func (suite *CdpTestSuite) TestCalculateCollateralToDebtRatio() {
	_, addrs := app.GeneratePrivKeyAddressPairs(1)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], cs(c("xrp", 3)), cs(c("usdx", 1)), tmtime.Canonical(time.Now()))
	cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdp.Collateral, cdp.Principal)
	suite.Equal(sdk.MustNewDecFromStr("3.0"), cr)
	cdp = types.NewCDP(types.DefaultCdpStartingID, addrs[0], cs(c("xrp", 1)), cs(c("usdx", 2)), tmtime.Canonical(time.Now()))
	cr = suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdp.Collateral, cdp.Principal)
	suite.Equal(sdk.MustNewDecFromStr("0.5"), cr)
	cdp = types.NewCDP(types.DefaultCdpStartingID, addrs[0], cs(c("xrp", 3)), cs(c("usdx", 1), c("susd", 2)), tmtime.Canonical(time.Now()))
	cr = suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdp.Collateral, cdp.Principal)
	suite.Equal(sdk.MustNewDecFromStr("1"), cr)
}

func (suite *CdpTestSuite) TestSetCdpByCollateralRatio() {
	_, addrs := app.GeneratePrivKeyAddressPairs(1)
	cdp := types.NewCDP(types.DefaultCdpStartingID, addrs[0], cs(c("xrp", 3)), cs(c("usdx", 1)), tmtime.Canonical(time.Now()))
	cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdp.Collateral, cdp.Principal)
	suite.NotPanics(func() { suite.keeper.IndexCdpByCollateralRatio(suite.ctx, cdp.Collateral[0].Denom, cdp.ID, cr) })
}

func (suite *CdpTestSuite) TestIterateCdps() {
	cdps := cdps()
	for _, c := range cdps {
		suite.keeper.SetCDP(suite.ctx, c)
		suite.keeper.IndexCdpByOwner(suite.ctx, c)
		cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, c.Collateral, c.Principal)
		suite.keeper.IndexCdpByCollateralRatio(suite.ctx, c.Collateral[0].Denom, c.ID, cr)
	}
	t := suite.keeper.GetAllCdps(suite.ctx)
	suite.Equal(4, len(t))
}

func (suite *CdpTestSuite) TestIterateCdpsByDenom() {
	cdps := cdps()
	for _, c := range cdps {
		suite.keeper.SetCDP(suite.ctx, c)
		suite.keeper.IndexCdpByOwner(suite.ctx, c)
		cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, c.Collateral, c.Principal)
		suite.keeper.IndexCdpByCollateralRatio(suite.ctx, c.Collateral[0].Denom, c.ID, cr)
	}
	xrpCdps := suite.keeper.GetAllCdpsByDenom(suite.ctx, "xrp")
	suite.Equal(3, len(xrpCdps))
	btcCdps := suite.keeper.GetAllCdpsByDenom(suite.ctx, "btc")
	suite.Equal(1, len(btcCdps))
	suite.keeper.DeleteCDP(suite.ctx, cdps[0])
	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[0])
	xrpCdps = suite.keeper.GetAllCdpsByDenom(suite.ctx, "xrp")
	suite.Equal(2, len(xrpCdps))
	suite.keeper.DeleteCDP(suite.ctx, cdps[1])
	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[1])
	ids, found := suite.keeper.GetCdpIdsByOwner(suite.ctx, cdps[1].Owner)
	suite.True(found)
	suite.Equal(1, len(ids))
	suite.Equal(uint64(3), ids[0])
}

func (suite *CdpTestSuite) TestIterateCdpsByCollateralRatio() {
	cdps := cdps()
	for _, c := range cdps {
		suite.keeper.SetCDP(suite.ctx, c)
		suite.keeper.IndexCdpByOwner(suite.ctx, c)
		cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, c.Collateral, c.Principal)
		suite.keeper.IndexCdpByCollateralRatio(suite.ctx, c.Collateral[0].Denom, c.ID, cr)
	}
	xrpCdps := suite.keeper.GetAllCdpsByDenomAndRatio(suite.ctx, "xrp", d("1.25"))
	suite.Equal(0, len(xrpCdps))
	xrpCdps = suite.keeper.GetAllCdpsByDenomAndRatio(suite.ctx, "xrp", d("1.25").Add(sdk.SmallestDec()))
	suite.Equal(1, len(xrpCdps))
	xrpCdps = suite.keeper.GetAllCdpsByDenomAndRatio(suite.ctx, "xrp", d("2.0").Add(sdk.SmallestDec()))
	suite.Equal(2, len(xrpCdps))
	xrpCdps = suite.keeper.GetAllCdpsByDenomAndRatio(suite.ctx, "xrp", d("100.0").Add(sdk.SmallestDec()))
	suite.Equal(3, len(xrpCdps))
	suite.keeper.DeleteCDP(suite.ctx, cdps[0])
	suite.keeper.RemoveCdpOwnerIndex(suite.ctx, cdps[0])
	cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, cdps[0].Collateral, cdps[0].Principal)
	suite.keeper.RemoveCdpCollateralRatioIndex(suite.ctx, cdps[0].Collateral[0].Denom, cdps[0].ID, cr)
	xrpCdps = suite.keeper.GetAllCdpsByDenomAndRatio(suite.ctx, "xrp", d("2.0").Add(sdk.SmallestDec()))
	suite.Equal(1, len(xrpCdps))
}

func (suite *CdpTestSuite) TestValidateCollateral() {
	c := sdk.NewCoins(sdk.NewCoin("xrp", sdk.NewInt(1)))
	err := suite.keeper.ValidateCollateral(suite.ctx, c)
	suite.NoError(err)
	c = sdk.NewCoins(sdk.NewCoin("lol", sdk.NewInt(1)))
	err = suite.keeper.ValidateCollateral(suite.ctx, c)
	suite.Equal(types.CodeCollateralNotSupported, err.Result().Code)
	c = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)), sdk.NewCoin("xrp", sdk.NewInt(1)))
	err = suite.keeper.ValidateCollateral(suite.ctx, c)
	suite.Equal(types.CodeCollateralLengthInvalid, err.Result().Code)
}

func (suite *CdpTestSuite) TestValidatePrincipal() {
	d := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(10000000)))
	err := suite.keeper.ValidatePrincipalAdd(suite.ctx, d)
	suite.NoError(err)
	d = sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(10000000)), sdk.NewCoin("susd", sdk.NewInt(10000000)))
	err = suite.keeper.ValidatePrincipalAdd(suite.ctx, d)
	suite.NoError(err)
	d = sdk.NewCoins(sdk.NewCoin("xusd", sdk.NewInt(1)))
	err = suite.keeper.ValidatePrincipalAdd(suite.ctx, d)
	suite.Equal(types.CodeDebtNotSupported, err.Result().Code)
	d = sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(1000000000001)))
	err = suite.keeper.ValidatePrincipalAdd(suite.ctx, d)
	suite.Equal(types.CodeExceedsDebtLimit, err.Result().Code)
}

func (suite *CdpTestSuite) TestCalculateCollateralizationRatio() {
	c := cdps()[1]
	suite.keeper.SetCDP(suite.ctx, c)
	suite.keeper.IndexCdpByOwner(suite.ctx, c)
	cr := suite.keeper.CalculateCollateralToDebtRatio(suite.ctx, c.Collateral, c.Principal)
	suite.keeper.IndexCdpByCollateralRatio(suite.ctx, c.Collateral[0].Denom, c.ID, cr)
	cr, err := suite.keeper.CalculateCollateralizationRatio(suite.ctx, c.Collateral, c.Principal, c.AccumulatedFees)
	suite.NoError(err)
	suite.Equal(d("2.5"), cr)
	c.AccumulatedFees = sdk.NewCoins(sdk.NewCoin("usdx", i(10000000)))
	cr, err = suite.keeper.CalculateCollateralizationRatio(suite.ctx, c.Collateral, c.Principal, c.AccumulatedFees)
	suite.NoError(err)
	suite.Equal(d("1.25"), cr)
}

func (suite *CdpTestSuite) TestMintBurnDebtCoins() {
	cd := cdps()[1]
	err := suite.keeper.MintDebtCoins(suite.ctx, types.ModuleName, suite.keeper.GetDebtDenom(suite.ctx), cd.Principal)
	suite.NoError(err)
	err = suite.keeper.MintDebtCoins(suite.ctx, "notamodule", suite.keeper.GetDebtDenom(suite.ctx), cd.Principal)
	suite.Error(err)
	sk := suite.app.GetSupplyKeeper()
	acc := sk.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.Equal(cs(c("debt", 10000000)), acc.GetCoins())

	err = suite.keeper.BurnDebtCoins(suite.ctx, types.ModuleName, suite.keeper.GetDebtDenom(suite.ctx), cd.Principal)
	suite.NoError(err)
	err = suite.keeper.BurnDebtCoins(suite.ctx, "notamodule", suite.keeper.GetDebtDenom(suite.ctx), cd.Principal)
	suite.Error(err)
	sk = suite.app.GetSupplyKeeper()
	acc = sk.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.Equal(sdk.Coins(nil), acc.GetCoins())
}

func TestCdpTestSuite(t *testing.T) {
	suite.Run(t, new(CdpTestSuite))
}
