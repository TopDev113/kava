package testutil

import (
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/auction/keeper"
	"github.com/kava-labs/kava/x/auction/types"
)

// Suite implements a test suite for the kavadist module integration tests
type Suite struct {
	suite.Suite

	Keeper        keeper.Keeper
	BankKeeper    bankkeeper.Keeper
	AccountKeeper authkeeper.AccountKeeper
	App           app.TestApp
	Ctx           sdk.Context
	Addrs         []sdk.AccAddress
	ModAcc        *authtypes.ModuleAccount
}

// SetupTest instantiates a new app, keepers, and sets suite state
func (suite *Suite) SetupTest(numAddrs int) {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	tApp := app.NewTestApp()

	_, addrs := app.GeneratePrivKeyAddressPairs(numAddrs)

	// Fund liquidator module account
	coins := sdk.NewCoins(
		sdk.NewCoin("token1", sdk.NewInt(100)),
		sdk.NewCoin("token2", sdk.NewInt(100)),
	)

	ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})

	modName := "liquidator"
	modBaseAcc := authtypes.NewBaseAccount(authtypes.NewModuleAddress(modName), nil, 0, 0)
	modAcc := authtypes.NewModuleAccount(modBaseAcc, modName, []string{authtypes.Minter, authtypes.Burner}...)
	suite.ModAcc = modAcc

	authGS := app.NewFundedGenStateWithSameCoinsWithModuleAccount(tApp.AppCodec(), coins, addrs, modAcc)

	params := types.NewParams(
		types.DefaultMaxAuctionDuration,
		types.DefaultBidDuration,
		types.DefaultIncrement,
		types.DefaultIncrement,
		types.DefaultIncrement,
	)

	auctionGs, err := types.NewGenesisState(types.DefaultNextAuctionID, params, []types.GenesisAuction{})
	suite.Require().NoError(err)

	moduleGs := tApp.AppCodec().MustMarshalJSON(auctionGs)
	gs := app.GenesisState{types.ModuleName: moduleGs}
	tApp.InitializeFromGenesisStates(authGS, gs)

	suite.App = tApp
	suite.Ctx = ctx
	suite.Addrs = addrs
	suite.Keeper = tApp.GetAuctionKeeper()
	suite.BankKeeper = tApp.GetBankKeeper()
	suite.AccountKeeper = tApp.GetAccountKeeper()
}

// CreateAccount adds coins to an account address
func (suite *Suite) AddCoinsToAccount(addr sdk.AccAddress, coins sdk.Coins) {
	ak := suite.App.GetAccountKeeper()
	acc := ak.NewAccountWithAddress(suite.Ctx, addr)
	ak.SetAccount(suite.Ctx, acc)

	err := simapp.FundAccount(suite.BankKeeper, suite.Ctx, acc.GetAddress(), coins)
	suite.Require().NoError(err)
}

// AddCoinsToModule adds coins to a named module account
func (suite *Suite) AddCoinsToNamedModule(moduleName string, amount sdk.Coins) {
	// Does not use suite.BankKeeper.MintCoins as module account would not have permission to mint
	err := simapp.FundModuleAccount(suite.BankKeeper, suite.Ctx, moduleName, amount)
	suite.Require().NoError(err)
}

// NewModuleAccountFromAddr creates a new module account from the provided address with the provided balance
// func (suite *Suite) NewModuleAccount(moduleName string, balance sdk.Coins) authtypes.AccountI {
// 	ak := suite.App.GetAccountKeeper()

// 	modAccAddr := authtypes.NewModuleAddress(moduleName)
// 	acc := ak.NewAccountWithAddress(suite.Ctx, modAccAddr)
// 	ak.SetAccount(suite.Ctx, acc)

// 	err := simapp.FundModuleAccount(suite.BankKeeper, suite.Ctx, moduleName, balance)
// 	suite.Require().NoError(err)

// 	return acc
// }

// CheckAccountBalanceEqual asserts that
func (suite *Suite) CheckAccountBalanceEqual(owner sdk.AccAddress, expectedCoins sdk.Coins) {
	balances := suite.BankKeeper.GetAllBalances(suite.Ctx, owner)
	suite.Equal(expectedCoins, balances)
}

// // CheckModuleAccountBalanceEqual asserts that a named module account balance matches the provided coins
// func (suite *Suite) CheckModuleAccountBalanceEqual(moduleName string, coins sdk.Coins) {
// 	balance := suite.BankKeeper.GetAllBalances(
// 		suite.Ctx,
// 		suite.AccountKeeper.GetModuleAddress(moduleName),
// 	)
// 	suite.Equal(coins, balance, fmt.Sprintf("expected module account balance to equal coins %s, but got %s", coins, balance))
// }
