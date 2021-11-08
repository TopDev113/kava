package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/kavadist/types"
)

type paramTest struct {
	params     types.Params
	expectPass bool
}

type ParamTestSuite struct {
	suite.Suite

	tests []paramTest
}

func (suite *ParamTestSuite) SetupTest() {
	p1 := types.Params{
		Active: true,
		Periods: []types.Period{
			{
				Start:     time.Date(2020, time.March, 1, 1, 0, 0, 0, time.UTC),
				End:       time.Date(2021, time.March, 1, 1, 0, 0, 0, time.UTC),
				Inflation: sdk.MustNewDecFromStr("1.000000003022265980"),
			},
			{
				Start:     time.Date(2021, time.March, 1, 1, 0, 0, 0, time.UTC),
				End:       time.Date(2022, time.March, 1, 1, 0, 0, 0, time.UTC),
				Inflation: sdk.MustNewDecFromStr("1.000000003022265980"),
			},
		},
	}
	p2 := types.Params{
		Active: true,
		Periods: []types.Period{
			{
				Start:     time.Date(2022, time.March, 1, 1, 0, 0, 0, time.UTC),
				End:       time.Date(2021, time.March, 1, 1, 0, 0, 0, time.UTC),
				Inflation: sdk.MustNewDecFromStr("1.000000003022265980"),
			},
			{
				Start:     time.Date(2023, time.March, 1, 1, 0, 0, 0, time.UTC),
				End:       time.Date(2024, time.March, 1, 1, 0, 0, 0, time.UTC),
				Inflation: sdk.MustNewDecFromStr("1.000000003022265980"),
			},
		},
	}
	p3 := types.Params{
		Active: true,
		Periods: []types.Period{
			{
				Start:     time.Date(2020, time.March, 1, 1, 0, 0, 0, time.UTC),
				End:       time.Date(2021, time.March, 1, 1, 0, 0, 0, time.UTC),
				Inflation: sdk.MustNewDecFromStr("1.000000003022265980"),
			},
			{
				Start:     time.Date(2020, time.March, 1, 1, 0, 0, 0, time.UTC),
				End:       time.Date(2022, time.March, 1, 1, 0, 0, 0, time.UTC),
				Inflation: sdk.MustNewDecFromStr("1.000000003022265980"),
			},
		},
	}

	suite.tests = []paramTest{
		{
			params:     p1,
			expectPass: true,
		},
		{
			params:     p2,
			expectPass: false,
		},
		{
			params:     p3,
			expectPass: false,
		},
	}
}

func (suite *ParamTestSuite) TestParamValidation() {
	for _, t := range suite.tests {
		err := t.params.Validate()
		if t.expectPass {
			suite.Require().NoError(err)
		} else {
			suite.Require().Error(err)
		}
	}
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamTestSuite))
}
