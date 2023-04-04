package keeper_test

import (
	"testing"

	"github.com/kava-labs/kava/x/earn/testutil"
	"github.com/kava-labs/kava/x/earn/types"
	"github.com/kava-labs/kava/x/earn/types/mocks"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type hookTestSuite struct {
	testutil.Suite
}

func (suite *hookTestSuite) SetupTest() {
	suite.Suite.SetupTest()
	suite.Keeper.SetParams(suite.Ctx, types.DefaultParams())
}

func TestHookTestSuite(t *testing.T) {
	suite.Run(t, new(hookTestSuite))
}

func (suite *hookTestSuite) TestHooks_DepositAndWithdraw() {
	suite.Keeper.ClearHooks()
	earnHooks := mocks.NewEarnHooks(suite.T())
	suite.Keeper.SetHooks(earnHooks)

	vault1Denom := "usdx"
	vault2Denom := "ukava"
	acc1deposit1Amount := sdk.NewInt64Coin(vault1Denom, 100)
	acc1deposit2Amount := sdk.NewInt64Coin(vault2Denom, 200)

	acc2deposit1Amount := sdk.NewInt64Coin(vault1Denom, 200)
	acc2deposit2Amount := sdk.NewInt64Coin(vault2Denom, 300)

	suite.CreateVault(vault1Denom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)
	suite.CreateVault(vault2Denom, types.StrategyTypes{types.STRATEGY_TYPE_SAVINGS}, false, nil)

	acc := suite.CreateAccount(sdk.NewCoins(
		sdk.NewInt64Coin(vault1Denom, 1000),
		sdk.NewInt64Coin(vault2Denom, 1000),
	), 0)

	acc2 := suite.CreateAccount(sdk.NewCoins(
		sdk.NewInt64Coin(vault1Denom, 1000),
		sdk.NewInt64Coin(vault2Denom, 1000),
	), 1)

	// first deposit creates vault - calls AfterVaultDepositCreated with initial shares
	// shares are 1:1
	earnHooks.On(
		"AfterVaultDepositCreated",
		suite.Ctx,
		acc1deposit1Amount.Denom,
		acc.GetAddress(),
		sdk.NewDecFromInt(acc1deposit1Amount.Amount),
	).Once()
	err := suite.Keeper.Deposit(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit1Amount,
		types.STRATEGY_TYPE_HARD,
	)
	suite.Require().NoError(err)

	// second deposit adds to vault - calls BeforeVaultDepositModified
	// shares given are the initial shares, not new the shares added to the vault
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc1deposit1Amount.Denom,
		acc.GetAddress(),
		sdk.NewDecFromInt(acc1deposit1Amount.Amount),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit1Amount,
		types.STRATEGY_TYPE_HARD,
	)
	suite.Require().NoError(err)

	// get the shares from the store from the last deposit
	shareRecord, found := suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc.GetAddress(),
	)
	suite.Require().True(found)

	// third deposit adds to vault - calls BeforeVaultDepositModified
	// shares given are the shares added in previous deposit, not the shares added to the vault now
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc1deposit1Amount.Denom,
		acc.GetAddress(),
		shareRecord.AmountOf(acc1deposit1Amount.Denom),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit1Amount,
		types.STRATEGY_TYPE_HARD,
	)
	suite.Require().NoError(err)

	// new deposit denom into vault creates the deposit and calls AfterVaultDepositCreated
	earnHooks.On(
		"AfterVaultDepositCreated",
		suite.Ctx,
		acc1deposit2Amount.Denom,
		acc.GetAddress(),
		sdk.NewDecFromInt(acc1deposit2Amount.Amount),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// second deposit into vault calls BeforeVaultDepositModified with initial shares given
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc1deposit2Amount.Denom,
		acc.GetAddress(),
		sdk.NewDecFromInt(acc1deposit2Amount.Amount),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// get the shares from the store from the last deposit
	shareRecord, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc.GetAddress(),
	)
	suite.Require().True(found)

	// third deposit into vault calls BeforeVaultDepositModified with shares from last deposit
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc1deposit2Amount.Denom,
		acc.GetAddress(),
		shareRecord.AmountOf(acc1deposit2Amount.Denom),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// ------------------------------------------------------------
	// Second account deposits

	// first deposit by user - calls AfterVaultDepositCreated with user's shares
	// not total shares
	earnHooks.On(
		"AfterVaultDepositCreated",
		suite.Ctx,
		acc2deposit1Amount.Denom,
		acc2.GetAddress(),
		sdk.NewDecFromInt(acc2deposit1Amount.Amount),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit1Amount,
		types.STRATEGY_TYPE_HARD,
	)
	suite.Require().NoError(err)

	// second deposit adds to vault - calls BeforeVaultDepositModified
	// shares given are the initial shares, not new the shares added to the vault
	// and not the total vault shares
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc2deposit1Amount.Denom,
		acc2.GetAddress(),
		sdk.NewDecFromInt(acc2deposit1Amount.Amount),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit1Amount,
		types.STRATEGY_TYPE_HARD,
	)
	suite.Require().NoError(err)

	// get the shares from the store from the last deposit
	shareRecord2, found := suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc2.GetAddress(),
	)
	suite.Require().True(found)

	// third deposit adds to vault - calls BeforeVaultDepositModified
	// shares given are the shares added in previous deposit, not the shares added to the vault now
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc2deposit1Amount.Denom,
		acc2.GetAddress(),
		shareRecord2.AmountOf(acc2deposit1Amount.Denom),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit1Amount,
		types.STRATEGY_TYPE_HARD,
	)
	suite.Require().NoError(err)

	// new deposit denom into vault creates the deposit and calls AfterVaultDepositCreated
	earnHooks.On(
		"AfterVaultDepositCreated",
		suite.Ctx,
		acc2deposit2Amount.Denom,
		acc2.GetAddress(),
		sdk.NewDecFromInt(acc2deposit2Amount.Amount),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// second deposit into vault calls BeforeVaultDepositModified with initial shares given
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc2deposit2Amount.Denom,
		acc2.GetAddress(),
		sdk.NewDecFromInt(acc2deposit2Amount.Amount),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// get the shares from the store from the last deposit
	shareRecord2, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc2.GetAddress(),
	)
	suite.Require().True(found)

	// third deposit into vault calls BeforeVaultDepositModified with shares from last deposit
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc2deposit2Amount.Denom,
		acc2.GetAddress(),
		shareRecord2.AmountOf(acc2deposit2Amount.Denom),
	).Once()
	err = suite.Keeper.Deposit(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// ------------------------------------------------------------
	// test hooks with a full withdraw of all shares deposit 1 denom
	shareRecord, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc.GetAddress(),
	)
	suite.Require().True(found)

	// all shares given to BeforeVaultDepositModified
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc1deposit1Amount.Denom,
		acc.GetAddress(),
		shareRecord.AmountOf(acc1deposit1Amount.Denom),
	).Once()
	_, err = suite.Keeper.Withdraw(
		suite.Ctx,
		acc.GetAddress(),
		// 3 deposits, multiply original deposit amount by 3
		sdk.NewCoin(acc1deposit1Amount.Denom, acc1deposit1Amount.Amount.MulRaw(3)),
		types.STRATEGY_TYPE_HARD,
	)
	suite.Require().NoError(err)

	// test hooks on partial withdraw
	shareRecord, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc.GetAddress(),
	)
	suite.Require().True(found)

	// all shares given to before deposit modified even with partial withdraw
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc1deposit2Amount.Denom,
		acc.GetAddress(),
		shareRecord.AmountOf(acc1deposit2Amount.Denom),
	).Once()
	_, err = suite.Keeper.Withdraw(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// test hooks on second partial withdraw
	shareRecord, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc.GetAddress(),
	)
	suite.Require().True(found)

	// all shares given to before deposit modified even with partial withdraw
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc1deposit2Amount.Denom,
		acc.GetAddress(),
		shareRecord.AmountOf(acc1deposit2Amount.Denom),
	).Once()
	_, err = suite.Keeper.Withdraw(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// test hooks withdraw all remaining shares
	shareRecord, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc.GetAddress(),
	)
	suite.Require().True(found)

	// all shares given to before deposit modified even with partial withdraw
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc1deposit2Amount.Denom,
		acc.GetAddress(),
		shareRecord.AmountOf(acc1deposit2Amount.Denom),
	).Once()
	_, err = suite.Keeper.Withdraw(
		suite.Ctx,
		acc.GetAddress(),
		acc1deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// ------------------------------------------------------------
	// withdraw from acc2
	shareRecord, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc2.GetAddress(),
	)
	suite.Require().True(found)

	// all shares given to BeforeVaultDepositModified
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc2deposit1Amount.Denom,
		acc2.GetAddress(),
		shareRecord.AmountOf(acc2deposit1Amount.Denom),
	).Once()
	_, err = suite.Keeper.Withdraw(
		suite.Ctx,
		acc2.GetAddress(),
		// 3 deposits, multiply original deposit amount by 3
		sdk.NewCoin(acc2deposit1Amount.Denom, acc2deposit1Amount.Amount.MulRaw(3)),
		types.STRATEGY_TYPE_HARD,
	)
	suite.Require().NoError(err)

	// test hooks on partial withdraw
	shareRecord2, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc2.GetAddress(),
	)
	suite.Require().True(found)

	// all shares given to before deposit modified even with partial withdraw
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc2deposit2Amount.Denom,
		acc2.GetAddress(),
		shareRecord2.AmountOf(acc2deposit2Amount.Denom),
	).Once()
	_, err = suite.Keeper.Withdraw(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// test hooks on second partial withdraw
	shareRecord2, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc2.GetAddress(),
	)
	suite.Require().True(found)

	// all shares given to before deposit modified even with partial withdraw
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc2deposit2Amount.Denom,
		acc2.GetAddress(),
		shareRecord2.AmountOf(acc2deposit2Amount.Denom),
	).Once()
	_, err = suite.Keeper.Withdraw(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)

	// test hooks withdraw all remaining shares
	shareRecord2, found = suite.Keeper.GetVaultAccountShares(
		suite.Ctx,
		acc2.GetAddress(),
	)
	suite.Require().True(found)

	// all shares given to before deposit modified even with partial withdraw
	earnHooks.On(
		"BeforeVaultDepositModified",
		suite.Ctx,
		acc2deposit2Amount.Denom,
		acc2.GetAddress(),
		shareRecord2.AmountOf(acc2deposit2Amount.Denom),
	).Once()
	_, err = suite.Keeper.Withdraw(
		suite.Ctx,
		acc2.GetAddress(),
		acc2deposit2Amount,
		types.STRATEGY_TYPE_SAVINGS,
	)
	suite.Require().NoError(err)
}

func (suite *hookTestSuite) TestHooks_NoPanicsOnNilHooks() {
	suite.Keeper.ClearHooks()

	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)
	withdrawAmount := sdk.NewInt64Coin(vaultDenom, 100)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	acc := suite.CreateAccount(sdk.NewCoins(startBalance), 0)

	// AfterVaultDepositModified should not panic if no hooks are registered
	err := suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// BeforeVaultDepositModified should not panic if no hooks are registered
	err = suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	// BeforeVaultDepositModified should not panic if no hooks are registered
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc.GetAddress(), withdrawAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)
}

func (suite *hookTestSuite) TestHooks_HookOrdering() {
	suite.Keeper.ClearHooks()
	earnHooks := &mocks.EarnHooks{}
	suite.Keeper.SetHooks(earnHooks)

	vaultDenom := "usdx"
	startBalance := sdk.NewInt64Coin(vaultDenom, 1000)
	depositAmount := sdk.NewInt64Coin(vaultDenom, 100)

	suite.CreateVault(vaultDenom, types.StrategyTypes{types.STRATEGY_TYPE_HARD}, false, nil)

	acc := suite.CreateAccount(sdk.NewCoins(startBalance), 0)

	earnHooks.On("AfterVaultDepositCreated", suite.Ctx, depositAmount.Denom, acc.GetAddress(), sdk.NewDecFromInt(depositAmount.Amount)).
		Run(func(args mock.Arguments) {
			shares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc.GetAddress())
			suite.Require().True(found, "expected after hook to be called after shares are updated")
			suite.Require().Equal(sdk.NewDecFromInt(depositAmount.Amount), shares.AmountOf(depositAmount.Denom))
		})
	err := suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	earnHooks.On("BeforeVaultDepositModified", suite.Ctx, depositAmount.Denom, acc.GetAddress(), sdk.NewDecFromInt(depositAmount.Amount)).
		Run(func(args mock.Arguments) {
			shares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc.GetAddress())
			suite.Require().True(found, "expected after hook to be called after shares are updated")
			suite.Require().Equal(sdk.NewDecFromInt(depositAmount.Amount), shares.AmountOf(depositAmount.Denom))
		})
	err = suite.Keeper.Deposit(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)

	existingShares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc.GetAddress())
	suite.Require().True(found)
	earnHooks.On("BeforeVaultDepositModified", suite.Ctx, depositAmount.Denom, acc.GetAddress(), existingShares.AmountOf(depositAmount.Denom)).
		Run(func(args mock.Arguments) {
			shares, found := suite.Keeper.GetVaultAccountShares(suite.Ctx, acc.GetAddress())
			suite.Require().True(found, "expected after hook to be called after shares are updated")
			suite.Require().Equal(sdk.NewDecFromInt(depositAmount.Amount.MulRaw(2)), shares.AmountOf(depositAmount.Denom))
		})
	_, err = suite.Keeper.Withdraw(suite.Ctx, acc.GetAddress(), depositAmount, types.STRATEGY_TYPE_HARD)
	suite.Require().NoError(err)
}
