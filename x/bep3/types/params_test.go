package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/bep3/types"
)

type ParamsTestSuite struct {
	suite.Suite
	addr   sdk.AccAddress
	supply []types.SupplyLimit
}

func (suite *ParamsTestSuite) SetupTest() {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	_, addrs := app.GeneratePrivKeyAddressPairs(1)
	suite.addr = addrs[0]
	supply1 := types.SupplyLimit{
		Limit:          sdkmath.NewInt(10000000000000),
		TimeLimited:    false,
		TimeBasedLimit: sdk.ZeroInt(),
		TimePeriod:     time.Hour,
	}
	supply2 := types.SupplyLimit{
		Limit:          sdkmath.NewInt(10000000000000),
		TimeLimited:    true,
		TimeBasedLimit: sdkmath.NewInt(100000000000),
		TimePeriod:     time.Hour * 24,
	}
	suite.supply = append(suite.supply, supply1, supply2)
}

func (suite *ParamsTestSuite) TestParamValidation() {
	type args struct {
		assetParams types.AssetParams
	}

	testCases := []struct {
		name        string
		args        args
		expectPass  bool
		expectedErr string
	}{
		{
			name: "default",
			args: args{
				assetParams: types.AssetParams{},
			},
			expectPass:  true,
			expectedErr: "",
		},
		{
			name: "valid single asset",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714, suite.supply[0], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  true,
			expectedErr: "",
		},
		{
			name: "valid single asset time limited",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714, suite.supply[1], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  true,
			expectedErr: "",
		},
		{
			name: "valid multi asset",
			args: args{
				assetParams: types.AssetParams{
					types.NewAssetParam(
						"bnb", 714, suite.supply[0], true,
						suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
						types.DefaultMinBlockLock, types.DefaultMaxBlockLock),
					types.NewAssetParam(
						"btcb", 0, suite.supply[1], true,
						suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(10000000), sdkmath.NewInt(100000000000),
						types.DefaultMinBlockLock, types.DefaultMaxBlockLock),
				},
			},
			expectPass:  true,
			expectedErr: "",
		},
		{
			name: "invalid denom - empty",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"", 714, suite.supply[0], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  false,
			expectedErr: "denom invalid",
		},
		{
			name: "min block lock equal max block lock",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714, suite.supply[0], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					243, 243)},
			},
			expectPass:  true,
			expectedErr: "",
		},
		{
			name: "min block lock greater max block lock",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714, suite.supply[0], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					244, 243)},
			},
			expectPass:  false,
			expectedErr: "minimum block lock > maximum block lock",
		},
		{
			name: "min swap not positive",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714, suite.supply[0], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(0), sdkmath.NewInt(10000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  false,
			expectedErr: "must have a positive minimum swap",
		},
		{
			name: "max swap not positive",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714, suite.supply[0], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(10000), sdkmath.NewInt(0),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  false,
			expectedErr: "must have a positive maximum swap",
		},
		{
			name: "min swap greater max swap",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714, suite.supply[0], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000000), sdkmath.NewInt(10000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  false,
			expectedErr: "minimum swap amount > maximum swap amount",
		},
		{
			name: "negative coin id",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", -714, suite.supply[0], true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  false,
			expectedErr: "coin id must be a non negative",
		},
		{
			name: "negative asset limit",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714,
					types.SupplyLimit{sdkmath.NewInt(-10000000000000), false, time.Hour, sdk.ZeroInt()}, true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  false,
			expectedErr: "invalid (negative) supply limit",
		},
		{
			name: "negative asset time limit",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714,
					types.SupplyLimit{sdkmath.NewInt(10000000000000), false, time.Hour, sdkmath.NewInt(-10000000000000)}, true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  false,
			expectedErr: "invalid (negative) supply time limit",
		},
		{
			name: "asset time limit greater than overall limit",
			args: args{
				assetParams: types.AssetParams{types.NewAssetParam(
					"bnb", 714,
					types.SupplyLimit{sdkmath.NewInt(10000000000000), true, time.Hour, sdkmath.NewInt(100000000000000)},
					true,
					suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
					types.DefaultMinBlockLock, types.DefaultMaxBlockLock)},
			},
			expectPass:  false,
			expectedErr: "supply time limit > supply limit",
		},
		{
			name: "duplicate denom",
			args: args{
				assetParams: types.AssetParams{
					types.NewAssetParam(
						"bnb", 714, suite.supply[0], true,
						suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(100000000), sdkmath.NewInt(100000000000),
						types.DefaultMinBlockLock, types.DefaultMaxBlockLock),
					types.NewAssetParam(
						"bnb", 0, suite.supply[0], true,
						suite.addr, sdkmath.NewInt(1000), sdkmath.NewInt(10000000), sdkmath.NewInt(100000000000),
						types.DefaultMinBlockLock, types.DefaultMaxBlockLock),
				},
			},
			expectPass:  false,
			expectedErr: "duplicate denom",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			params := types.NewParams(tc.args.assetParams)
			err := params.Validate()
			if tc.expectPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
				suite.Require().Contains(err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}
