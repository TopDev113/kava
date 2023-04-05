package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/kava-labs/kava/x/community/testutil"
	"github.com/kava-labs/kava/x/community/types"
)

// Test suite used for all keeper tests
type KeeperTestSuite struct {
	testutil.Suite
}

// The default state used by each test
func (suite *KeeperTestSuite) SetupTest() {
	suite.Suite.SetupTest()
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestCommunityPool() {
	suite.SetupTest()
	maccAddr := suite.App.GetAccountKeeper().GetModuleAddress(types.ModuleAccountName)

	funds := sdk.NewCoins(
		sdk.NewCoin("ukava", sdkmath.NewInt(10000)),
		sdk.NewCoin("usdx", sdkmath.NewInt(100)),
	)
	sender := suite.CreateFundedAccount(funds)

	suite.Run("FundCommunityPool", func() {
		err := suite.Keeper.FundCommunityPool(suite.Ctx, sender, funds)
		suite.Require().NoError(err)

		// check that community pool received balance
		suite.App.CheckBalance(suite.T(), suite.Ctx, maccAddr, funds)
		suite.Equal(funds, suite.Keeper.GetModuleAccountBalance(suite.Ctx))
		// check that sender had balance deducted
		suite.App.CheckBalance(suite.T(), suite.Ctx, sender, sdk.NewCoins())
	})

	// send it back
	suite.Run("DistributeFromCommunityPool - valid", func() {
		err := suite.Keeper.DistributeFromCommunityPool(suite.Ctx, sender, funds)
		suite.Require().NoError(err)

		// community pool has funds deducted
		suite.App.CheckBalance(suite.T(), suite.Ctx, maccAddr, sdk.NewCoins())
		suite.Equal(sdk.NewCoins(), suite.Keeper.GetModuleAccountBalance(suite.Ctx))
		// receiver receives the funds
		suite.App.CheckBalance(suite.T(), suite.Ctx, sender, funds)
	})

	// can't send more than we have!
	suite.Run("DistributeFromCommunityPool - insufficient funds", func() {
		suite.Equal(sdk.NewCoins(), suite.Keeper.GetModuleAccountBalance(suite.Ctx))
		err := suite.Keeper.DistributeFromCommunityPool(suite.Ctx, sender, funds)
		suite.Require().ErrorContains(err, "insufficient funds")
	})
}
