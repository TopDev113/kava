package testutil

import (
	"fmt"
	"reflect"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/earn/keeper"
	"github.com/kava-labs/kava/x/earn/types"
	"github.com/kava-labs/kava/x/hard"

	hardkeeper "github.com/kava-labs/kava/x/hard/keeper"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	savingskeeper "github.com/kava-labs/kava/x/savings/keeper"
	savingstypes "github.com/kava-labs/kava/x/savings/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
)

var TestBkavaDenoms = []string{
	"bkava-kavavaloper15gqc744d05xacn4n6w2furuads9fu4pqn6zxlu",
	"bkava-kavavaloper15qdefkmwswysgg4qxgqpqr35k3m49pkx8yhpte",
	"bkava-kavavaloper1ypjp0m04pyp73hwgtc0dgkx0e9rrydeckewa42",
}

// Suite implements a test suite for the earn module integration tests
type Suite struct {
	suite.Suite
	Keeper        keeper.Keeper
	App           app.TestApp
	Ctx           sdk.Context
	BankKeeper    bankkeeper.Keeper
	AccountKeeper authkeeper.AccountKeeper

	// Strategy Keepers
	HardKeeper    hardkeeper.Keeper
	SavingsKeeper savingskeeper.Keeper
}

// SetupTest instantiates a new app, keepers, and sets suite state
func (suite *Suite) SetupTest() {
	// Pricefeed required for withdrawing from hard
	pricefeedGS := pricefeedtypes.GenesisState{
		Params: pricefeedtypes.Params{
			Markets: []pricefeedtypes.Market{
				{MarketID: "usdx:usd", BaseAsset: "usdx", QuoteAsset: "usd", Oracles: []sdk.AccAddress{}, Active: true},
				{MarketID: "kava:usd", BaseAsset: "kava", QuoteAsset: "usd", Oracles: []sdk.AccAddress{}, Active: true},
				{MarketID: "bnb:usd", BaseAsset: "bnb", QuoteAsset: "usd", Oracles: []sdk.AccAddress{}, Active: true},
			},
		},
		PostedPrices: []pricefeedtypes.PostedPrice{
			{
				MarketID:      "usdx:usd",
				OracleAddress: sdk.AccAddress{},
				Price:         sdk.MustNewDecFromStr("1.00"),
				Expiry:        time.Now().Add(100 * time.Hour),
			},
			{
				MarketID:      "kava:usd",
				OracleAddress: sdk.AccAddress{},
				Price:         sdk.MustNewDecFromStr("2.00"),
				Expiry:        time.Now().Add(100 * time.Hour),
			},
			{
				MarketID:      "bnb:usd",
				OracleAddress: sdk.AccAddress{},
				Price:         sdk.MustNewDecFromStr("10.00"),
				Expiry:        time.Now().Add(100 * time.Hour),
			},
		},
	}

	hardGS := hardtypes.NewGenesisState(hardtypes.NewParams(
		hardtypes.MoneyMarkets{
			hardtypes.NewMoneyMarket(
				"usdx",
				hardtypes.NewBorrowLimit(
					true,
					sdk.MustNewDecFromStr("20000000"),
					sdk.MustNewDecFromStr("1"),
				),
				"usdx:usd",
				sdkmath.NewInt(1000000),
				hardtypes.NewInterestRateModel(
					sdk.MustNewDecFromStr("0.05"),
					sdk.MustNewDecFromStr("2"),
					sdk.MustNewDecFromStr("0.8"),
					sdk.MustNewDecFromStr("10"),
				),
				sdk.MustNewDecFromStr("0.05"),
				sdk.ZeroDec(),
			),
			hardtypes.NewMoneyMarket(
				"busd",
				hardtypes.NewBorrowLimit(
					true,
					sdk.MustNewDecFromStr("20000000"),
					sdk.MustNewDecFromStr("1"),
				),
				"busd:usd",
				sdkmath.NewInt(1000000),
				hardtypes.NewInterestRateModel(
					sdk.MustNewDecFromStr("0.05"),
					sdk.MustNewDecFromStr("2"),
					sdk.MustNewDecFromStr("0.8"),
					sdk.MustNewDecFromStr("10"),
				),
				sdk.MustNewDecFromStr("0.05"),
				sdk.ZeroDec(),
			),
			hardtypes.NewMoneyMarket(
				"kava",
				hardtypes.NewBorrowLimit(
					true,
					sdk.MustNewDecFromStr("20000000"),
					sdk.MustNewDecFromStr("1"),
				),
				"kava:usd",
				sdkmath.NewInt(1000000),
				hardtypes.NewInterestRateModel(
					sdk.MustNewDecFromStr("0.05"),
					sdk.MustNewDecFromStr("2"),
					sdk.MustNewDecFromStr("0.8"),
					sdk.MustNewDecFromStr("10"),
				),
				sdk.MustNewDecFromStr("0.05"),
				sdk.ZeroDec(),
			),
		},
		sdk.NewDec(10),
	),
		hardtypes.DefaultAccumulationTimes,
		hardtypes.DefaultDeposits,
		hardtypes.DefaultBorrows,
		hardtypes.DefaultTotalSupplied,
		hardtypes.DefaultTotalBorrowed,
		hardtypes.DefaultTotalReserves,
	)

	savingsGS := savingstypes.NewGenesisState(
		savingstypes.NewParams(
			[]string{
				"ukava",
				"busd",
				"usdx",
				TestBkavaDenoms[0],
				TestBkavaDenoms[1],
				TestBkavaDenoms[2],
			},
		),
		nil,
	)

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = "ukava"

	stakingGs := stakingtypes.GenesisState{
		Params: stakingParams,
	}

	tApp := app.NewTestApp()

	tApp.InitializeFromGenesisStates(
		app.GenesisState{
			pricefeedtypes.ModuleName: tApp.AppCodec().MustMarshalJSON(&pricefeedGS),
			hardtypes.ModuleName:      tApp.AppCodec().MustMarshalJSON(&hardGS),
			savingstypes.ModuleName:   tApp.AppCodec().MustMarshalJSON(&savingsGS),
			stakingtypes.ModuleName:   tApp.AppCodec().MustMarshalJSON(&stakingGs),
		},
	)

	ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})

	suite.Ctx = ctx
	suite.App = tApp
	suite.Keeper = tApp.GetEarnKeeper()
	suite.BankKeeper = tApp.GetBankKeeper()
	suite.AccountKeeper = tApp.GetAccountKeeper()

	suite.HardKeeper = tApp.GetHardKeeper()
	suite.SavingsKeeper = tApp.GetSavingsKeeper()

	hard.BeginBlocker(suite.Ctx, suite.HardKeeper)
}

// GetEvents returns emitted events on the sdk context
func (suite *Suite) GetEvents() sdk.Events {
	return suite.Ctx.EventManager().Events()
}

// AddCoinsToModule adds coins to the earn module account
func (suite *Suite) AddCoinsToModule(amount sdk.Coins) {
	// Does not use suite.BankKeeper.MintCoins as module account would not have permission to mint
	err := banktestutil.FundModuleAccount(suite.BankKeeper, suite.Ctx, types.ModuleName, amount)
	suite.Require().NoError(err)
}

// RemoveCoinsFromModule removes coins to the earn module account
func (suite *Suite) RemoveCoinsFromModule(amount sdk.Coins) {
	// Earn module does not have BurnCoins permission so we need to transfer to gov first to burn
	err := suite.BankKeeper.SendCoinsFromModuleToModule(suite.Ctx, types.ModuleAccountName, govtypes.ModuleName, amount)
	suite.Require().NoError(err)
	err = suite.BankKeeper.BurnCoins(suite.Ctx, govtypes.ModuleName, amount)
	suite.Require().NoError(err)
}

// CreateAccount creates a new account from the provided balance, using index
// to create different new addresses.
func (suite *Suite) CreateAccount(initialBalance sdk.Coins, index int) authtypes.AccountI {
	_, addrs := app.GeneratePrivKeyAddressPairs(index + 1)
	ak := suite.App.GetAccountKeeper()

	acc := ak.NewAccountWithAddress(suite.Ctx, addrs[index])
	ak.SetAccount(suite.Ctx, acc)

	err := banktestutil.FundAccount(suite.BankKeeper, suite.Ctx, acc.GetAddress(), initialBalance)
	suite.Require().NoError(err)

	return acc
}

// NewAccountFromAddr creates a new account from the provided address with the provided balance
func (suite *Suite) NewAccountFromAddr(addr sdk.AccAddress, balance sdk.Coins) authtypes.AccountI {
	ak := suite.App.GetAccountKeeper()

	acc := ak.NewAccountWithAddress(suite.Ctx, addr)
	ak.SetAccount(suite.Ctx, acc)

	err := banktestutil.FundAccount(suite.BankKeeper, suite.Ctx, acc.GetAddress(), balance)
	suite.Require().NoError(err)

	return acc
}

// CreateVault adds a new vault to the keeper parameters
func (suite *Suite) CreateVault(
	vaultDenom string,
	vaultStrategies types.StrategyTypes,
	isPrivateVault bool,
	allowedDepositors []sdk.AccAddress,
) {
	vault := types.NewAllowedVault(vaultDenom, vaultStrategies, isPrivateVault, allowedDepositors)

	allowedVaults := suite.Keeper.GetAllowedVaults(suite.Ctx)
	allowedVaults = append(allowedVaults, vault)

	params := types.NewParams(allowedVaults)

	suite.Keeper.SetParams(
		suite.Ctx,
		params,
	)
}

// AccountBalanceEqual asserts that the coins match the account balance
func (suite *Suite) AccountBalanceEqual(addr sdk.AccAddress, coins sdk.Coins) {
	balance := suite.BankKeeper.GetAllBalances(suite.Ctx, addr)
	suite.Equal(coins, balance, fmt.Sprintf("expected account balance to equal coins %s, but got %s", coins, balance))
}

// ModuleAccountBalanceEqual asserts that the earn module account balance matches the provided coins
func (suite *Suite) ModuleAccountBalanceEqual(coins sdk.Coins) {
	balance := suite.BankKeeper.GetAllBalances(
		suite.Ctx,
		suite.AccountKeeper.GetModuleAddress(types.ModuleAccountName),
	)
	suite.Equal(coins, balance, fmt.Sprintf("expected module account balance to equal coins %s, but got %s", coins, balance))
}

// ----------------------------------------------------------------------------
// Earn

// VaultTotalValuesEqual asserts that the vault total values match the provided
// values.
func (suite *Suite) VaultTotalValuesEqual(expected sdk.Coins) {
	for _, coin := range expected {
		vaultBal, err := suite.Keeper.GetVaultTotalValue(suite.Ctx, coin.Denom)
		suite.Require().NoError(err, "failed to get vault balance")
		suite.Require().Equal(coin, vaultBal)
	}
}

// VaultTotalSharesEqual asserts that the vault total shares match the provided
// values.
func (suite *Suite) VaultTotalSharesEqual(expected types.VaultShares) {
	for _, share := range expected {
		vaultBal, found := suite.Keeper.GetVaultTotalShares(suite.Ctx, share.Denom)
		suite.Require().Truef(found, "%s vault does not exist", share.Denom)
		suite.Require().Equal(share.Amount, vaultBal.Amount)
	}
}

// VaultAccountSharesEqual asserts that the vault account shares match the provided
// values.
func (suite *Suite) VaultAccountSharesEqual(accs []sdk.AccAddress, supplies []sdk.Coins) {
	for i, acc := range accs {
		coins := supplies[i]

		accVaultBal, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc)
		suite.Require().True(found)

		for _, coin := range coins {
			suite.Require().Equal(
				coin.Amount,
				accVaultBal.AmountOf(coin.Denom),
				"expected account vault balance to equal coins %s, but got %s",
				coins, accVaultBal,
			)
		}
	}
}

// ----------------------------------------------------------------------------
// Hard

// HardDepositAmountEqual asserts that the hard deposit amount matches the provided
// values.
func (suite *Suite) HardDepositAmountEqual(expected sdk.Coins) {
	macc := suite.AccountKeeper.GetModuleAccount(suite.Ctx, types.ModuleName)

	hardDeposit, found := suite.HardKeeper.GetSyncedDeposit(suite.Ctx, macc.GetAddress())
	if expected.IsZero() {
		suite.Require().False(found)
		return
	}

	suite.Require().True(found, "hard should have a deposit")
	suite.Require().Equalf(
		expected,
		hardDeposit.Amount,
		"hard should have a deposit with the amount %v",
		expected,
	)
}

// ----------------------------------------------------------------------------
// Savings

// SavingsDepositAmountEqual asserts that the savings deposit amount matches the
// provided values.
func (suite *Suite) SavingsDepositAmountEqual(expected sdk.Coins) {
	macc := suite.AccountKeeper.GetModuleAccount(suite.Ctx, types.ModuleName)

	savingsDeposit, found := suite.SavingsKeeper.GetDeposit(suite.Ctx, macc.GetAddress())
	if expected.IsZero() {
		suite.Require().False(found)
		return
	}

	suite.Require().True(found, "savings should have a deposit")
	suite.Require().Equalf(
		expected,
		savingsDeposit.Amount,
		"savings should have a deposit with the amount %v",
		expected,
	)
}

// ----------------------------------------------------------------------------
// Staking

// CreateNewUnbondedValidator creates a new validator in the staking module.
// New validators are unbonded until the end blocker is run.
func (suite *Suite) CreateNewUnbondedValidator(addr sdk.ValAddress, selfDelegation sdkmath.Int) stakingtypes.Validator {
	// Create a validator
	err := suite.deliverMsgCreateValidator(suite.Ctx, addr, suite.NewBondCoin(selfDelegation))
	suite.Require().NoError(err)

	// New validators are created in an unbonded state. Note if the end blocker is run later this validator could become bonded.

	validator, found := suite.App.GetStakingKeeper().GetValidator(suite.Ctx, addr)
	suite.Require().True(found)
	return validator
}

// NewBondCoin creates a Coin with the current staking denom.
func (suite *Suite) NewBondCoin(amount sdkmath.Int) sdk.Coin {
	stakingDenom := suite.App.GetStakingKeeper().BondDenom(suite.Ctx)
	return sdk.NewCoin(stakingDenom, amount)
}

// CreateDelegation delegates tokens to a validator.
func (suite *Suite) CreateDelegation(valAddr sdk.ValAddress, delegator sdk.AccAddress, amount sdkmath.Int) sdk.Dec {
	sk := suite.App.GetStakingKeeper()

	stakingDenom := sk.BondDenom(suite.Ctx)
	msg := stakingtypes.NewMsgDelegate(
		delegator,
		valAddr,
		sdk.NewCoin(stakingDenom, amount),
	)

	msgServer := stakingkeeper.NewMsgServerImpl(sk)
	_, err := msgServer.Delegate(sdk.WrapSDKContext(suite.Ctx), msg)
	suite.Require().NoError(err)

	del, found := sk.GetDelegation(suite.Ctx, delegator, valAddr)
	suite.Require().True(found)
	return del.Shares
}

func (suite *Suite) deliverMsgCreateValidator(ctx sdk.Context, address sdk.ValAddress, selfDelegation sdk.Coin) error {
	msg, err := stakingtypes.NewMsgCreateValidator(
		address,
		ed25519.GenPrivKey().PubKey(),
		selfDelegation,
		stakingtypes.Description{},
		stakingtypes.NewCommissionRates(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
		sdkmath.NewInt(1e6),
	)
	if err != nil {
		return err
	}

	msgServer := stakingkeeper.NewMsgServerImpl(suite.App.GetStakingKeeper())
	_, err = msgServer.CreateValidator(sdk.WrapSDKContext(suite.Ctx), msg)
	return err
}

// ----------------------------------------------------------------------------

// EventsContains asserts that the expected event is in the provided events
func (suite *Suite) EventsContains(events sdk.Events, expectedEvent sdk.Event) {
	foundMatch := false
	for _, event := range events {
		if event.Type == expectedEvent.Type {
			if reflect.DeepEqual(attrsToMap(expectedEvent.Attributes), attrsToMap(event.Attributes)) {
				foundMatch = true
			}
		}
	}

	suite.True(foundMatch, fmt.Sprintf("event of type %s not found or did not match", expectedEvent.Type))
}

func attrsToMap(attrs []abci.EventAttribute) []sdk.Attribute { // new cosmos changed the event attribute type
	out := []sdk.Attribute{}

	for _, attr := range attrs {
		out = append(out, sdk.NewAttribute(string(attr.Key), string(attr.Value)))
	}

	return out
}
