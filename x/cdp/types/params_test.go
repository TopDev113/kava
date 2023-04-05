package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/cdp/types"
)

type ParamsTestSuite struct {
	suite.Suite
}

func (suite *ParamsTestSuite) SetupTest() {
}

func (suite *ParamsTestSuite) TestParamValidation() {
	type args struct {
		globalDebtLimit  sdk.Coin
		collateralParams types.CollateralParams
		debtParam        types.DebtParam
		surplusThreshold sdkmath.Int
		surplusLot       sdkmath.Int
		debtThreshold    sdkmath.Int
		debtLot          sdkmath.Int
		breaker          bool
	}
	type errArgs struct {
		expectPass bool
		contains   string
	}

	testCases := []struct {
		name    string
		args    args
		errArgs errArgs
	}{
		{
			name: "default",
			args: args{
				globalDebtLimit:  types.DefaultGlobalDebt,
				collateralParams: types.DefaultCollateralParams,
				debtParam:        types.DefaultDebtParam,
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: true,
				contains:   "",
			},
		},
		{
			name: "valid single-collateral",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 4000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: true,
				contains:   "",
			},
		},
		{
			name: "invalid single-collateral mismatched debt denoms",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 4000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "susd",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "does not match global debt denom",
			},
		},
		{
			name: "invalid single-collateral over debt limit",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 1000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "exceeds global debt limit",
			},
		},
		{
			name: "valid multi-collateral",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 4000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
					{
						Denom:                            "xrp",
						Type:                             "xrp-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "xrp:usd",
						LiquidationMarketID:              "xrp:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(6),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: true,
				contains:   "",
			},
		},
		{
			name: "invalid multi-collateral over debt limit",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
					{
						Denom:                            "xrp",
						Type:                             "xrp-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "xrp:usd",
						LiquidationMarketID:              "xrp:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(6),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "sum of collateral debt limits",
			},
		},
		{
			name: "invalid multi-collateral multiple debt denoms",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 4000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
					{
						Denom:                            "xrp",
						Type:                             "xrp-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("susd", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "xrp:usd",
						LiquidationMarketID:              "xrp:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(6),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "does not match global debt limit denom",
			},
		},
		{
			name: "invalid collateral params empty denom",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "collateral denom invalid",
			},
		},
		{
			name: "invalid collateral params empty market id",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "",
						LiquidationMarketID:              "",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "market id cannot be blank",
			},
		},
		{
			name: "invalid collateral params duplicate denom + type",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 1000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 1000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "duplicate cdp collateral type",
			},
		},
		{
			name: "valid collateral params duplicate denom + different type",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 1000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
					{
						Denom:                            "bnb",
						Type:                             "bnb-b",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 1000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: true,
				contains:   "",
			},
		},
		{
			name: "invalid collateral params nil debt limit",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.Coin{},
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "debt limit for all collaterals should be positive",
			},
		},
		{
			name: "invalid collateral params liquidation ratio out of range",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 1000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("1.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "liquidation penalty should be between 0 and 1",
			},
		},
		{
			name: "invalid collateral params auction size zero",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 1000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdk.ZeroInt(),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "auction size should be positive",
			},
		},
		{
			name: "invalid collateral params stability fee out of range",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 1000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.1"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "stability fee must be ≥ 1.0",
			},
		},
		{
			name: "invalid collateral params zero liquidation ratio",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("0.0"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 1_000_000_000_000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.1"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50_000_000_000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam:        types.DefaultDebtParam,
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "liquidation ratio must be > 0",
			},
		},
		{
			name: "invalid debt param empty denom",
			args: args{
				globalDebtLimit: sdk.NewInt64Coin("usdx", 2000000000000),
				collateralParams: types.CollateralParams{
					{
						Denom:                            "bnb",
						Type:                             "bnb-a",
						LiquidationRatio:                 sdk.MustNewDecFromStr("1.5"),
						DebtLimit:                        sdk.NewInt64Coin("usdx", 2000000000000),
						StabilityFee:                     sdk.MustNewDecFromStr("1.000000001547125958"),
						LiquidationPenalty:               sdk.MustNewDecFromStr("0.05"),
						AuctionSize:                      sdkmath.NewInt(50000000000),
						SpotMarketID:                     "bnb:usd",
						LiquidationMarketID:              "bnb:usd",
						KeeperRewardPercentage:           sdk.MustNewDecFromStr("0.01"),
						ConversionFactor:                 sdkmath.NewInt(8),
						CheckCollateralizationIndexCount: sdkmath.NewInt(10),
					},
				},
				debtParam: types.DebtParam{
					Denom:            "",
					ReferenceAsset:   "usd",
					ConversionFactor: sdkmath.NewInt(6),
					DebtFloor:        sdkmath.NewInt(10000000),
				},
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "debt denom invalid",
			},
		},
		{
			name: "nil debt limit",
			args: args{
				globalDebtLimit:  sdk.Coin{},
				collateralParams: types.DefaultCollateralParams,
				debtParam:        types.DefaultDebtParam,
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "global debt limit <nil>: invalid coins",
			},
		},
		{
			name: "zero surplus auction threshold",
			args: args{
				globalDebtLimit:  types.DefaultGlobalDebt,
				collateralParams: types.DefaultCollateralParams,
				debtParam:        types.DefaultDebtParam,
				surplusThreshold: sdk.ZeroInt(),
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "surplus auction threshold should be positive",
			},
		},
		{
			name: "zero debt auction threshold",
			args: args{
				globalDebtLimit:  types.DefaultGlobalDebt,
				collateralParams: types.DefaultCollateralParams,
				debtParam:        types.DefaultDebtParam,
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    sdk.ZeroInt(),
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "debt auction threshold should be positive",
			},
		},
		{
			name: "zero surplus auction lot",
			args: args{
				globalDebtLimit:  types.DefaultGlobalDebt,
				collateralParams: types.DefaultCollateralParams,
				debtParam:        types.DefaultDebtParam,
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       sdk.ZeroInt(),
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          types.DefaultDebtLot,
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "surplus auction lot should be positive",
			},
		},
		{
			name: "zero debt auction lot",
			args: args{
				globalDebtLimit:  types.DefaultGlobalDebt,
				collateralParams: types.DefaultCollateralParams,
				debtParam:        types.DefaultDebtParam,
				surplusThreshold: types.DefaultSurplusThreshold,
				surplusLot:       types.DefaultSurplusLot,
				debtThreshold:    types.DefaultDebtThreshold,
				debtLot:          sdk.ZeroInt(),
				breaker:          types.DefaultCircuitBreaker,
			},
			errArgs: errArgs{
				expectPass: false,
				contains:   "debt auction lot should be positive",
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			params := types.NewParams(tc.args.globalDebtLimit, tc.args.collateralParams, tc.args.debtParam, tc.args.surplusThreshold, tc.args.surplusLot, tc.args.debtThreshold, tc.args.debtLot, tc.args.breaker)
			err := params.Validate()
			if tc.errArgs.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errArgs.contains)
			}
		})
	}
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}
