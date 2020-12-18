package types_test

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/suite"

	"github.com/kava-labs/kava/x/harvest/types"
)

type GenesisTestSuite struct {
	suite.Suite
}

func (suite *GenesisTestSuite) TestGenesisValidation() {
	type args struct {
		params types.Params
		pbt    time.Time
		pdts   types.GenesisDistributionTimes
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
				params: types.DefaultParams(),
				pbt:    types.DefaultPreviousBlockTime,
				pdts:   types.DefaultDistributionTimes,
			},
			expectPass:  true,
			expectedErr: "",
		},
		{
			name: "valid",
			args: args{
				params: types.NewParams(
					true,
					types.DistributionSchedules{
						types.NewDistributionSchedule(true, "bnb", time.Date(2020, 10, 8, 14, 0, 0, 0, time.UTC), time.Date(2020, 11, 22, 14, 0, 0, 0, time.UTC), sdk.NewCoin("hard", sdk.NewInt(5000)), time.Date(2021, 11, 22, 14, 0, 0, 0, time.UTC), types.Multipliers{types.NewMultiplier(types.Small, 0, sdk.OneDec()), types.NewMultiplier(types.Medium, 6, sdk.MustNewDecFromStr("1.5")), types.NewMultiplier(types.Medium, 24, sdk.MustNewDecFromStr("3"))}),
					},
					types.DelegatorDistributionSchedules{types.NewDelegatorDistributionSchedule(
						types.NewDistributionSchedule(true, "bnb", time.Date(2020, 10, 8, 14, 0, 0, 0, time.UTC), time.Date(2025, 10, 8, 14, 0, 0, 0, time.UTC), sdk.NewCoin("hard", sdk.NewInt(500)), time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC), types.Multipliers{types.NewMultiplier(types.Small, 0, sdk.MustNewDecFromStr("0.33")), types.NewMultiplier(types.Medium, 6, sdk.MustNewDecFromStr("0.5")), types.NewMultiplier(types.Medium, 24, sdk.OneDec())}),
						time.Hour*24,
					),
					},
					types.DefaultMoneyMarkets,
					types.DefaultCheckLtvIndexCount,
				),
				pbt: time.Date(2020, 10, 8, 12, 0, 0, 0, time.UTC),
				pdts: types.GenesisDistributionTimes{
					{PreviousDistributionTime: time.Date(2020, 10, 8, 12, 0, 0, 0, time.UTC), Denom: "bnb"},
				},
			},
			expectPass:  true,
			expectedErr: "",
		},
		{
			name: "invalid previous blocktime",
			args: args{
				params: types.NewParams(
					true,
					types.DistributionSchedules{
						types.NewDistributionSchedule(true, "bnb", time.Date(2020, 10, 8, 14, 0, 0, 0, time.UTC), time.Date(2020, 11, 22, 14, 0, 0, 0, time.UTC), sdk.NewCoin("hard", sdk.NewInt(5000)), time.Date(2021, 11, 22, 14, 0, 0, 0, time.UTC), types.Multipliers{types.NewMultiplier(types.Small, 0, sdk.OneDec()), types.NewMultiplier(types.Medium, 6, sdk.MustNewDecFromStr("1.5")), types.NewMultiplier(types.Medium, 24, sdk.MustNewDecFromStr("3"))}),
					},
					types.DelegatorDistributionSchedules{types.NewDelegatorDistributionSchedule(
						types.NewDistributionSchedule(true, "bnb", time.Date(2020, 10, 8, 14, 0, 0, 0, time.UTC), time.Date(2025, 10, 8, 14, 0, 0, 0, time.UTC), sdk.NewCoin("hard", sdk.NewInt(500)), time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC), types.Multipliers{types.NewMultiplier(types.Small, 0, sdk.MustNewDecFromStr("0.33")), types.NewMultiplier(types.Medium, 6, sdk.MustNewDecFromStr("0.5")), types.NewMultiplier(types.Medium, 24, sdk.OneDec())}),
						time.Hour*24,
					),
					},
					types.DefaultMoneyMarkets,
					types.DefaultCheckLtvIndexCount,
				),
				pbt: time.Time{},
				pdts: types.GenesisDistributionTimes{
					{PreviousDistributionTime: time.Date(2020, 10, 8, 12, 0, 0, 0, time.UTC), Denom: "bnb"},
				},
			},
			expectPass:  false,
			expectedErr: "previous block time not set",
		},
		{
			name: "invalid previous distribution time",
			args: args{
				params: types.NewParams(
					true,
					types.DistributionSchedules{
						types.NewDistributionSchedule(true, "bnb", time.Date(2020, 10, 8, 14, 0, 0, 0, time.UTC), time.Date(2020, 11, 22, 14, 0, 0, 0, time.UTC), sdk.NewCoin("hard", sdk.NewInt(5000)), time.Date(2021, 11, 22, 14, 0, 0, 0, time.UTC), types.Multipliers{types.NewMultiplier(types.Small, 0, sdk.OneDec()), types.NewMultiplier(types.Medium, 6, sdk.MustNewDecFromStr("1.5")), types.NewMultiplier(types.Medium, 24, sdk.MustNewDecFromStr("3"))}),
					},
					types.DelegatorDistributionSchedules{types.NewDelegatorDistributionSchedule(
						types.NewDistributionSchedule(true, "bnb", time.Date(2020, 10, 8, 14, 0, 0, 0, time.UTC), time.Date(2025, 10, 8, 14, 0, 0, 0, time.UTC), sdk.NewCoin("hard", sdk.NewInt(500)), time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC), types.Multipliers{types.NewMultiplier(types.Small, 0, sdk.MustNewDecFromStr("0.33")), types.NewMultiplier(types.Medium, 6, sdk.MustNewDecFromStr("0.5")), types.NewMultiplier(types.Medium, 24, sdk.OneDec())}),
						time.Hour*24,
					),
					},
					types.DefaultMoneyMarkets,
					types.DefaultCheckLtvIndexCount,
				),
				pbt: time.Date(2020, 10, 8, 12, 0, 0, 0, time.UTC),
				pdts: types.GenesisDistributionTimes{
					{PreviousDistributionTime: time.Time{}, Denom: "bnb"},
				},
			},
			expectPass:  false,
			expectedErr: "previous distribution time not set",
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			gs := types.NewGenesisState(tc.args.params, tc.args.pbt, tc.args.pdts)
			err := gs.Validate()
			if tc.expectPass {
				suite.NoError(err)
			} else {
				suite.Error(err)
				suite.Require().True(strings.Contains(err.Error(), tc.expectedErr))
			}
		})
	}
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}
