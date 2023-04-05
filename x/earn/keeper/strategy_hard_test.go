package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/earn/testutil"
	"github.com/kava-labs/kava/x/earn/types"

	"github.com/stretchr/testify/suite"
)

type strategyHardTestSuite struct {
	testutil.Suite
}

func (suite *strategyHardTestSuite) SetupTest() {
	suite.Suite.SetupTest()
	suite.Keeper.SetParams(suite.Ctx, types.DefaultParams())
}

func TestStrategyLendTestSuite(t *testing.T) {
	suite.Run(t, new(strategyHardTestSuite))
}

func (suite *strategyHardTestSuite) TestGetStrategyType() {
	strategy, err := suite.Keeper.GetStrategy(types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	suite.Equal(types.STRATEGY_TYPE_HARD, strategy.GetStrategyType())
}

func (suite *strategyHardTestSuite) TestDeposit_SingleAcc() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	acc := suite.CreateAccount(sdk.NewCoins(startBalance), 0)

	err := suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	suite.HardDepositAmountEqual(sdk.NewCoins(depositAmount))
	suite.VaultTotalValuesEqual(sdk.NewCoins(depositAmount))
	suite.VaultTotalSharesEqual(types.NewVaultShares(
		types.NewVaultShare(depositAmount.Denom, sdk.NewDecFromInt(depositAmount.Amount)),
	))

	// Query vault total
	totalValue, err := suite.Keeper.GetVaultTotalValue(suite.Ctx, vaultDenom)
	suite.Require().NoError(err)

	suite.Equal(depositAmount, totalValue)
}

func (suite *strategyHardTestSuite) TestDeposit_SingleAcc_MultipleDeposits() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	acc := suite.CreateAccount(sdk.NewCoins(startBalance), 0)

	err := suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// Second deposit
	err = suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	expectedVaultBalance := depositAmount.Add(depositAmount)
	suite.HardDepositAmountEqual(sdk.NewCoins(expectedVaultBalance))
	suite.VaultTotalValuesEqual(sdk.NewCoins(expectedVaultBalance))
	suite.VaultTotalSharesEqual(types.NewVaultShares(
		types.NewVaultShare(expectedVaultBalance.Denom, sdk.NewDecFromInt(expectedVaultBalance.Amount)),
	))

	// Query vault total
	totalValue, err := suite.Keeper.GetVaultTotalValue(suite.Ctx, vaultDenom)
	suite.Require().NoError(err)

	suite.Equal(depositAmount.Add(depositAmount), totalValue)
}

func (suite *strategyHardTestSuite) TestDeposit_MultipleAcc_MultipleDeposits() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)

	expectedTotalValue := sdk.NewCoin(vaultDenom, depositAmount.Amount.MulRaw(4))

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	acc1 := suite.CreateAccount(sdk.NewCoins(startBalance), 0)
	acc2 := suite.CreateAccount(sdk.NewCoins(startBalance), 0)

	// 2 deposits each account
	for i := 0; i < 2; i++ {
		// Deposit from acc1
		err := suite.Keeper.Deposit(suite.Ctx, acc1.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
		suite.Require().NoError(err)

		// Deposit from acc2
		err = suite.Keeper.Deposit(suite.Ctx, acc2.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
		suite.Require().NoError(err)
	}

	suite.HardDepositAmountEqual(sdk.NewCoins(expectedTotalValue))
	suite.VaultTotalValuesEqual(sdk.NewCoins(expectedTotalValue))
	suite.VaultTotalSharesEqual(types.NewVaultShares(
		types.NewVaultShare(expectedTotalValue.Denom, sdk.NewDecFromInt(expectedTotalValue.Amount)),
	))

	// Query vault total
	totalValue, err := suite.Keeper.GetVaultTotalValue(suite.Ctx, vaultDenom)
	suite.Require().NoError(err)

	suite.Equal(expectedTotalValue, totalValue)
}

func (suite *strategyHardTestSuite) TestGetVaultTotalValue_Empty() {
	vaultDenom := "usdx"

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	// Query vault total
	totalValue, err := suite.Keeper.GetVaultTotalValue(suite.Ctx, vaultDenom)
	suite.Require().NoError(err)

	suite.Equal(sdk.NewCoin(vaultDenom, sdk.ZeroInt()), totalValue)
}

func (suite *strategyHardTestSuite) TestGetVaultTotalValue_NoDenomDeposit() {
	// 2 Vaults usdx, busd
	// 1st vault has deposits
	// 2nd vault has no deposits
	vaultDenom := "usdx"
	vaultDenomBusd := "busd"

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)
	suite.CreateVault(vaultDenomBusd, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)

	acc := suite.CreateAccount(sdk.NewCoins(startBalance), 0)

	// Deposit vault1
	err := suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// Query vault total, hard deposit exists for account, but amount in busd does not
	// Vault2 does not have any value, only returns amount for the correct denom
	// if a hard deposit already exists
	totalValueBusd, err := suite.Keeper.GetVaultTotalValue(suite.Ctx, vaultDenomBusd)
	suite.Require().NoError(err)

	suite.Equal(sdk.NewCoin(vaultDenomBusd, sdk.ZeroInt()), totalValueBusd)
}

// ----------------------------------------------------------------------------
// Withdraw

func (suite *strategyHardTestSuite) TestWithdraw() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	acc := suite.CreateAccount(sdk.NewCoins(startBalance), 0)
	err := suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	suite.HardDepositAmountEqual(sdk.NewCoins(depositAmount))

	// Query vault total
	totalValue, err := suite.Keeper.GetVaultTotalValue(suite.Ctx, vaultDenom)
	suite.Require().NoError(err)
	suite.Equal(depositAmount, totalValue)

	// Withdraw
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	suite.HardDepositAmountEqual(sdk.NewCoins())
	suite.VaultTotalValuesEqual(sdk.NewCoins())
	suite.VaultTotalSharesEqual(types.NewVaultShares())

	totalValue, err = suite.Keeper.GetVaultTotalValue(suite.Ctx, vaultDenom)
	suite.Require().NoError(err)
	suite.Equal(sdk.NewInt64Coin(vaultDenom, 0), totalValue)

	// Withdraw again
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrVaultRecordNotFound, "vault should be deleted when no more supply")
}

func (suite *strategyHardTestSuite) TestWithdraw_OnlyWithdrawOwnSupply() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	// Deposits from 2 accounts
	acc1 := suite.CreateAccount(sdk.NewCoins(startBalance), 0).GetAddress()
	acc2 := suite.CreateAccount(sdk.NewCoins(startBalance), 1).GetAddress()
	err := suite.Keeper.Deposit(suite.Ctx, acc1, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	err = suite.Keeper.Deposit(suite.Ctx, acc2, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// Withdraw
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc1, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// Withdraw again
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc1, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().Error(err)
	suite.Require().ErrorIs(
		err,
		types.ErrVaultShareRecordNotFound,
		"should only be able to withdraw the account's own supply",
	)
}

func (suite *strategyHardTestSuite) TestWithdraw_WithAccumulatedHard() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	// Deposits accounts
	acc := suite.CreateAccount(sdk.NewCoins(startBalance), 0).GetAddress()
	acc2 := suite.CreateAccount(sdk.NewCoins(startBalance), 1).GetAddress()

	err := suite.Keeper.Deposit(suite.Ctx, acc, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// Deposit from acc2 so the vault doesn't get deleted when withdrawing
	err = suite.Keeper.Deposit(suite.Ctx, acc2, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// Direct hard deposit from module account to increase vault value
	err = suite.App.FundModuleAccount(suite.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 20)))
	suite.Require().NoError(err)

	macc := suite.AccountKeeper.GetModuleAccount(suite.Ctx, types.ModuleName)
	err = suite.HardKeeper.Deposit(suite.Ctx, macc.GetAddress(), sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 20)))
	suite.Require().NoError(err)

	// Query account value
	accValue, err := suite.Keeper.GetVaultAccountValue(suite.Ctx, vaultDenom, acc)
	suite.Require().NoError(err)
	suite.Equal(depositAmount.AddAmount(sdkmath.NewInt(10)), accValue)

	// Withdraw 100, 10 remaining
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// Withdraw 100 again -- too much
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().Error(err)
	suite.Require().ErrorIs(
		err,
		types.ErrInsufficientValue,
		"cannot withdraw more than account value",
	)

	// Half of remaining 10, 5 remaining
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc, sdk.NewCoin(vaultDenom, sdkmath.NewInt(5)), types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// Withdraw all
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc, sdk.NewCoin(vaultDenom, sdkmath.NewInt(5)), types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	accValue, err = suite.Keeper.GetVaultAccountValue(suite.Ctx, vaultDenom, acc)
	suite.Require().Errorf(
		err,
		"account should be deleted when all shares withdrawn but has %s value still",
		accValue,
	)
	suite.Require().Equal("account vault share record for usdx not found", err.Error())
}

func (suite *strategyHardTestSuite) TestAccountShares() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)
	err := suite.App.FundModuleAccount(suite.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 1000)))
	suite.Require().NoError(err)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	// Deposit from account1
	acc1 := suite.CreateAccount(sdk.NewCoins(startBalance), 0).GetAddress()
	acc2 := suite.CreateAccount(sdk.NewCoins(startBalance), 1).GetAddress()

	// 1. acc1 deposit 100
	err = suite.Keeper.Deposit(suite.Ctx, acc1, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	acc1Shares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc1)
	suite.Require().True(found)
	suite.Equal(sdk.NewDec(100), acc1Shares.AmountOf(vaultDenom), "initial deposit 1:1 shares")

	// 2. Direct hard deposit from module account to increase vault value
	// Total value: 100 -> 110
	macc := suite.AccountKeeper.GetModuleAccount(suite.Ctx, types.ModuleName)
	err = suite.HardKeeper.Deposit(suite.Ctx, macc.GetAddress(), sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 10)))
	suite.Require().NoError(err)

	// 2. acc2 deposit 100
	// share price is 10% more expensive now
	// hard 110 -> 210
	err = suite.Keeper.Deposit(suite.Ctx, acc2, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// 100 * 100 / 210 = 47.619047619 shares
	// 2.1 price * 47.619047619 = 99.9999999999
	acc2Value, err := suite.Keeper.GetVaultAccountValue(suite.Ctx, vaultDenom, acc2)
	suite.Require().NoError(err)
	suite.Equal(
		sdkmath.NewInt(99),
		acc2Value.Amount,
		"value 1 less than deposit amount with different share price, decimals truncated",
	)

	acc2Shares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc2)
	suite.Require().True(found)
	// 100 * 100 / 110 = 190.909090909090909091
	// QuoInt64() truncates
	expectedAcc2Shares := sdk.NewDec(100).MulInt64(100).QuoInt64(110)
	suite.Equal(expectedAcc2Shares, acc2Shares.AmountOf(vaultDenom))

	vaultTotalShares, found := suite.Keeper.GetVaultTotalShares(suite.Ctx, vaultDenom)
	suite.Require().True(found)
	suite.Equal(sdk.NewDec(100).Add(expectedAcc2Shares), vaultTotalShares.Amount)

	// Hard deposit again from module account to triple original value
	// 210 -> 300
	err = suite.HardKeeper.Deposit(suite.Ctx, macc.GetAddress(), sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 90)))
	suite.Require().NoError(err)

	// Deposit again from acc1
	err = suite.Keeper.Deposit(suite.Ctx, acc1, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	acc1Shares, found = suite.Keeper.GetVaultAccountShares(suite.Ctx, acc1)
	suite.Require().True(found)
	// totalShares = 100 + 90            = 190
	// totalValue  = 100 + 10 + 100 + 90 = 300
	// sharesIssued = assetAmount * (shareCount / totalTokens)
	// sharedIssued = 100 * 190 / 300 = 63.3 = 63
	// total shares = 100 + 63 = 163
	suite.Equal(
		sdk.NewDec(100).Add(sdk.NewDec(100).Mul(vaultTotalShares.Amount).Quo(sdk.NewDec(300))),
		acc1Shares.AmountOf(vaultDenom),
		"shares should consist of 100 of 1x share price and 63 of 3x share price",
	)
}

func (suite *strategyHardTestSuite) TestWithdraw_AccumulatedAmount() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)
	err := suite.App.FundModuleAccount(suite.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 1000)))
	suite.Require().NoError(err)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	// Deposit from account1
	acc1 := suite.CreateAccount(sdk.NewCoins(startBalance), 0).GetAddress()
	acc2 := suite.CreateAccount(sdk.NewCoins(startBalance), 1).GetAddress()

	// 1. acc1 deposit 100
	err = suite.Keeper.Deposit(suite.Ctx, acc1, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// acc2 deposit 100, just to make sure other deposits do not affect acc1
	err = suite.Keeper.Deposit(suite.Ctx, acc2, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	acc1Shares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc1)
	suite.Require().True(found)
	suite.Equal(sdk.NewDec(100), acc1Shares.AmountOf(vaultDenom), "initial deposit 1:1 shares")

	// 2. Direct hard deposit from module account to increase vault value
	// Total value: 200 -> 220, 110 each account
	macc := suite.AccountKeeper.GetModuleAccount(suite.Ctx, types.ModuleName)
	err = suite.HardKeeper.Deposit(suite.Ctx, macc.GetAddress(), sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 20)))
	suite.Require().NoError(err)

	// 3. Withdraw all from acc1 - including accumulated amount
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc1, depositAmount.AddAmount(sdkmath.NewInt(10)), types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	_, found = suite.Keeper.GetVaultAccountShares(suite.Ctx, acc1)
	suite.Require().False(found, "should have withdrawn entire shares")
}

func (suite *strategyHardTestSuite) TestWithdraw_AccumulatedTruncated() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)
	err := suite.App.FundModuleAccount(suite.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 1000)))
	suite.Require().NoError(err)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	// Deposit from account1
	acc1 := suite.CreateAccount(sdk.NewCoins(startBalance), 0).GetAddress()
	acc2 := suite.CreateAccount(sdk.NewCoins(startBalance), 1).GetAddress()

	// 1. acc1 deposit 100
	err = suite.Keeper.Deposit(suite.Ctx, acc1, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// acc2 deposit 100, just to make sure other deposits do not affect acc1
	err = suite.Keeper.Deposit(suite.Ctx, acc2, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	acc1Shares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc1)
	suite.Require().True(found)
	suite.Equal(sdk.NewDec(100), acc1Shares.AmountOf(vaultDenom), "initial deposit 1:1 shares")

	// 2. Direct hard deposit from module account to increase vault value
	// Total value: 200 -> 211, 105.5 each account
	macc := suite.AccountKeeper.GetModuleAccount(suite.Ctx, types.ModuleName)
	err = suite.HardKeeper.Deposit(suite.Ctx, macc.GetAddress(), sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 11)))
	suite.Require().NoError(err)

	accBal, err := suite.Keeper.GetVaultAccountValue(suite.Ctx, vaultDenom, acc1)
	suite.Require().NoError(err)
	suite.Equal(depositAmount.AddAmount(sdkmath.NewInt(5)), accBal, "acc1 should have 105 usdx")

	// 3. Withdraw all from acc1 - including accumulated amount
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc1, depositAmount.AddAmount(sdkmath.NewInt(5)), types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	acc1Shares, found = suite.Keeper.GetVaultAccountShares(suite.Ctx, acc1)
	suite.Require().Falsef(found, "should have withdrawn entire shares but has %s", acc1Shares)

	_, err = suite.Keeper.GetVaultAccountValue(suite.Ctx, vaultDenom, acc1)
	suite.Require().Error(err)
}

func (suite *strategyHardTestSuite) TestWithdraw_ExpensiveShares() {
	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)
	err := suite.App.FundModuleAccount(suite.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 2000)))
	suite.Require().NoError(err)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	// Deposit from account1
	acc1 := suite.CreateAccount(sdk.NewCoins(startBalance), 0).GetAddress()

	// 1. acc1 deposit 100
	err = suite.Keeper.Deposit(suite.Ctx, acc1, depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	acc1Shares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc1)
	suite.Require().True(found)
	suite.Equal(sdk.NewDec(100), acc1Shares.AmountOf(vaultDenom), "initial deposit 1:1 shares")

	// 2. Direct hard deposit from module account to increase vault value
	// Total value: 100 -> 2000, shares now 10usdx each
	macc := suite.AccountKeeper.GetModuleAccount(suite.Ctx, types.ModuleName)
	err = suite.HardKeeper.Deposit(suite.Ctx, macc.GetAddress(), sdk.NewCoins(sdk.NewInt64Coin(vaultDenom, 1900)))
	suite.Require().NoError(err)

	accBal, err := suite.Keeper.GetVaultAccountValue(suite.Ctx, vaultDenom, acc1)
	suite.Require().NoError(err)
	suite.Equal(sdkmath.NewInt(2000), accBal.Amount, "acc1 should have 2000 usdx")

	// 3. Withdraw all from acc1 - including accumulated amount
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc1, sdk.NewInt64Coin(vaultDenom, 2000), types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	acc1Shares, found = suite.Keeper.GetVaultAccountShares(suite.Ctx, acc1)
	suite.Require().Falsef(found, "should have withdrawn entire shares but has %s", acc1Shares)

	_, err = suite.Keeper.GetVaultAccountValue(suite.Ctx, vaultDenom, acc1)
	suite.Require().Error(err)
}
