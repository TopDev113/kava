package types_test

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/suite"

	"github.com/kava-labs/kava/x/incentive/types"
)

type ParamTestSuite struct {
	suite.Suite
}

func (suite *ParamTestSuite) SetupTest() {}

func (suite *ParamTestSuite) TestParamValidation() {
	type args struct {
		usdxMintingRewardPeriods   types.RewardPeriods
		hardSupplyRewardPeriods    types.RewardPeriods
		hardBorrowRewardPeriods    types.RewardPeriods
		hardDelegatorRewardPeriods types.RewardPeriods
		multipliers                types.Multipliers
		end                        time.Time
	}

	type errArgs struct {
		expectPass bool
		contains   string
	}
	type test struct {
		name    string
		args    args
		errArgs errArgs
	}

	testCases := []test{
		{
			"default",
			args{
				usdxMintingRewardPeriods:   types.DefaultRewardPeriods,
				hardSupplyRewardPeriods:    types.DefaultRewardPeriods,
				hardBorrowRewardPeriods:    types.DefaultRewardPeriods,
				hardDelegatorRewardPeriods: types.DefaultRewardPeriods,
				multipliers:                types.DefaultMultipliers,
				end:                        types.DefaultClaimEnd,
			},
			errArgs{
				expectPass: true,
				contains:   "",
			},
		},
		{
			"valid",
			args{
				usdxMintingRewardPeriods: types.RewardPeriods{types.NewRewardPeriod(
					true, "bnb-a", time.Date(2020, 10, 15, 14, 0, 0, 0, time.UTC), time.Date(2024, 10, 15, 14, 0, 0, 0, time.UTC),
					sdk.NewCoin(types.USDXMintingRewardDenom, sdk.NewInt(122354)))},
				multipliers: types.Multipliers{
					types.NewMultiplier(
						types.Small, 1, sdk.MustNewDecFromStr("0.25"),
					),
					types.NewMultiplier(
						types.Large, 1, sdk.MustNewDecFromStr("1.0"),
					),
				},
				hardSupplyRewardPeriods:    types.DefaultRewardPeriods,
				hardBorrowRewardPeriods:    types.DefaultRewardPeriods,
				hardDelegatorRewardPeriods: types.DefaultRewardPeriods,
				end:                        time.Date(2025, 10, 15, 14, 0, 0, 0, time.UTC),
			},
			errArgs{
				expectPass: true,
				contains:   "",
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			params := types.NewParams(tc.args.usdxMintingRewardPeriods, tc.args.hardSupplyRewardPeriods,
				tc.args.hardBorrowRewardPeriods, tc.args.hardDelegatorRewardPeriods, tc.args.multipliers, tc.args.end,
			)
			err := params.Validate()
			if tc.errArgs.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().True(strings.Contains(err.Error(), tc.errArgs.contains))
			}
		})
	}
}

func TestParamTestSuite(t *testing.T) {
	suite.Run(t, new(ParamTestSuite))
}
