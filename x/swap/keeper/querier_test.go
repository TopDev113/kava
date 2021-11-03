package keeper_test

// import (
// 	"strings"
// 	"testing"

// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/stretchr/testify/suite"
// 	abci "github.com/tendermint/tendermint/abci/types"

// 	"github.com/kava-labs/kava/app"
// 	"github.com/kava-labs/kava/x/swap/keeper"
// 	"github.com/kava-labs/kava/x/swap/testutil"
// 	"github.com/kava-labs/kava/x/swap/types"
// )

// type querierTestSuite struct {
// 	testutil.Suite
// 	querier   sdk.Querier
// 	addresses []sdk.AccAddress
// }

// func (suite *querierTestSuite) SetupTest() {
// 	suite.Suite.SetupTest()

// 	// Set up auth GenesisState
// 	_, addrs := app.GeneratePrivKeyAddressPairs(5)
// 	coins := []sdk.Coins{}
// 	for j := 0; j < 5; j++ {
// 		coins = append(coins, cs(c("ukava", 10000000000), c("bnb", 10000000000), c("usdx", 10000000000)))
// 	}
// 	suite.addresses = addrs
// 	authGS := app.NewAuthGenState(addrs, coins)

// 	suite.App.InitializeFromGenesisStates(
// 		authGS,
// 		NewSwapGenStateMulti(),
// 	)
// 	suite.querier = keeper.NewQuerier(suite.Keeper)
// }

// func (suite *querierTestSuite) TestUnkownRequest() {
// 	ctx := suite.Ctx.WithIsCheckTx(false)
// 	bz, err := suite.querier(ctx, []string{"invalid-path"}, abci.RequestQuery{})
// 	suite.Nil(bz)
// 	suite.EqualError(err, "unknown request: unknown swap query endpoint")
// }

// func (suite *querierTestSuite) TestQueryParams() {
// 	ctx := suite.Ctx.WithIsCheckTx(false)
// 	bz, err := suite.querier(ctx, []string{types.QueryGetParams}, abci.RequestQuery{})
// 	suite.Nil(err)
// 	suite.NotNil(bz)

// 	var p types.Params
// 	suite.Nil(types.ModuleCdc.UnmarshalJSON(bz, &p))

// 	swapGenesisState := NewSwapGenStateMulti()
// 	gs := types.GenesisState{}
// 	err = types.ModuleCdc.UnmarshalJSON(swapGenesisState["swap"], &gs)
// 	suite.Require().NoError(err)

// 	suite.Equal(gs.Params, p)
// }

// func (suite *querierTestSuite) TestQueryPool() {
// 	// Set up pool in store
// 	coinA := sdk.NewCoin("ukava", sdk.NewInt(10))
// 	coinB := sdk.NewCoin("usdx", sdk.NewInt(200))

// 	pool, err := types.NewDenominatedPool(sdk.NewCoins(coinA, coinB))
// 	suite.Nil(err)
// 	poolRecord := types.NewPoolRecordFromPool(pool)
// 	suite.Keeper.SetPool(suite.Ctx, poolRecord)

// 	ctx := suite.Ctx.WithIsCheckTx(false)
// 	// Set up request query
// 	query := abci.RequestQuery{
// 		Path: strings.Join([]string{"custom", types.QuerierRoute, types.QueryGetPool}, "/"),
// 		Data: types.ModuleCdc.MustMarshalJSON(types.NewQueryPoolParams(poolRecord.PoolID)),
// 	}

// 	bz, err := suite.querier(ctx, []string{types.QueryGetPool}, query)
// 	suite.Nil(err)
// 	suite.NotNil(bz)

// 	var res types.PoolStatsQueryResult
// 	suite.Nil(types.ModuleCdc.UnmarshalJSON(bz, &res))

// 	// Check that result matches expected result
// 	totalCoins := pool.ShareValue(pool.TotalShares())
// 	expectedResult := types.NewPoolStatsQueryResult(poolRecord.PoolID, totalCoins, pool.TotalShares())
// 	suite.Equal(expectedResult, res)
// }

// func (suite *querierTestSuite) TestQueryPools() {
// 	// Set up pools in store
// 	coinA := sdk.NewCoin("ukava", sdk.NewInt(10))
// 	coinB := sdk.NewCoin("usdx", sdk.NewInt(200))
// 	coinC := sdk.NewCoin("usdx", sdk.NewInt(200))

// 	poolAB, err := types.NewDenominatedPool(sdk.NewCoins(coinA, coinB))
// 	suite.Nil(err)
// 	poolRecordAB := types.NewPoolRecordFromPool(poolAB)
// 	suite.Keeper.SetPool(suite.Ctx, poolRecordAB)

// 	poolAC, err := types.NewDenominatedPool(sdk.NewCoins(coinA, coinC))
// 	suite.Nil(err)
// 	poolRecordAC := types.NewPoolRecordFromPool(poolAC)
// 	suite.Keeper.SetPool(suite.Ctx, poolRecordAC)

// 	// Build a map of pools to compare to query results
// 	pools := types.PoolRecords{poolRecordAB, poolRecordAC}
// 	poolsMap := make(map[string]types.PoolRecord)
// 	for _, pool := range pools {
// 		poolsMap[pool.PoolID] = pool
// 	}

// 	ctx := suite.Ctx.WithIsCheckTx(false)
// 	bz, err := suite.querier(ctx, []string{types.QueryGetPools}, abci.RequestQuery{})
// 	suite.Nil(err)
// 	suite.NotNil(bz)

// 	var res types.PoolStatsQueryResults
// 	suite.Nil(types.ModuleCdc.UnmarshalJSON(bz, &res))

// 	// Check that all pools are accounted for
// 	suite.Equal(len(poolsMap), len(res))
// 	// Check that each individual result matches the expected result
// 	for _, pool := range res {
// 		expectedPool, ok := poolsMap[pool.Name]
// 		suite.True(ok)
// 		suite.Equal(expectedPool.PoolID, pool.Name)
// 		suite.Equal(sdk.NewCoins(expectedPool.ReservesA, expectedPool.ReservesB), pool.Coins)
// 		suite.Equal(expectedPool.TotalShares, pool.TotalShares)
// 	}
// }

// func (suite *querierTestSuite) TestQueryDeposit() {
// 	// Set up pool in store
// 	coinA := sdk.NewCoin("ukava", sdk.NewInt(10))
// 	coinB := sdk.NewCoin("usdx", sdk.NewInt(200))
// 	pool, err := types.NewDenominatedPool(sdk.NewCoins(coinA, coinB))
// 	suite.Nil(err)
// 	poolRecord := types.NewPoolRecordFromPool(pool)
// 	suite.Keeper.SetPool(suite.Ctx, poolRecord)

// 	// Deposit into pool
// 	owner := suite.addresses[0]
// 	err = suite.Keeper.Deposit(suite.Ctx, owner, coinA, coinB, sdk.MustNewDecFromStr("0.20"))
// 	suite.Nil(err)

// 	ctx := suite.Ctx.WithIsCheckTx(false)
// 	// Set up request query
// 	query := abci.RequestQuery{
// 		Path: strings.Join([]string{"custom", types.QuerierRoute, types.QueryGetDeposits}, "/"),
// 		Data: types.ModuleCdc.MustMarshalJSON(types.NewQueryDepositsParams(1, 100, owner, poolRecord.PoolID)),
// 	}

// 	bz, err := suite.querier(ctx, []string{types.QueryGetDeposits}, query)
// 	suite.Nil(err)
// 	suite.NotNil(bz)

// 	var res types.DepositsQueryResults
// 	suite.Nil(types.ModuleCdc.UnmarshalJSON(bz, &res))

// 	// As the only depositor all pool shares should belong to the owner
// 	suite.Equal(poolRecord.TotalShares, res[0].SharesOwned)
// }

// func TestQuerierTestSuite(t *testing.T) {
// 	suite.Run(t, new(querierTestSuite))
// }
