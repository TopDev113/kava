package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/params"
)

// Parameter keys
var (
	KeyGlobalDebtLimit      = []byte("GlobalDebtLimit")
	KeyCollateralParams     = []byte("CollateralParams")
	KeyDebtParam            = []byte("DebtParam")
	KeyCircuitBreaker       = []byte("CircuitBreaker")
	KeyDebtThreshold        = []byte("DebtThreshold")
	KeyDebtLot              = []byte("DebtLot")
	KeySurplusThreshold     = []byte("SurplusThreshold")
	KeySurplusLot           = []byte("SurplusLot")
	DefaultGlobalDebt       = sdk.NewCoin(DefaultStableDenom, sdk.ZeroInt())
	DefaultCircuitBreaker   = false
	DefaultCollateralParams = CollateralParams{}
	DefaultDebtParam        = DebtParam{
		Denom:            "usdx",
		ReferenceAsset:   "usd",
		ConversionFactor: sdk.NewInt(6),
		DebtFloor:        sdk.NewInt(10000000),
	}
	DefaultCdpStartingID    = uint64(1)
	DefaultDebtDenom        = "debt"
	DefaultGovDenom         = "ukava"
	DefaultStableDenom      = "usdx"
	DefaultSurplusThreshold = sdk.NewInt(500000000000)
	DefaultDebtThreshold    = sdk.NewInt(100000000000)
	DefaultSurplusLot       = sdk.NewInt(10000000000)
	DefaultDebtLot          = sdk.NewInt(10000000000)
	minCollateralPrefix     = 0
	maxCollateralPrefix     = 255
	stabilityFeeMax         = sdk.MustNewDecFromStr("1.000000051034942716") // 500% APR
)

// Params governance parameters for cdp module
type Params struct {
	CollateralParams        CollateralParams `json:"collateral_params" yaml:"collateral_params"`
	DebtParam               DebtParam        `json:"debt_param" yaml:"debt_param"`
	GlobalDebtLimit         sdk.Coin         `json:"global_debt_limit" yaml:"global_debt_limit"`
	SurplusAuctionThreshold sdk.Int          `json:"surplus_auction_threshold" yaml:"surplus_auction_threshold"`
	SurplusAuctionLot       sdk.Int          `json:"surplus_auction_lot" yaml:"surplus_auction_lot"`
	DebtAuctionThreshold    sdk.Int          `json:"debt_auction_threshold" yaml:"debt_auction_threshold"`
	DebtAuctionLot          sdk.Int          `json:"debt_auction_lot" yaml:"debt_auction_lot"`
	CircuitBreaker          bool             `json:"circuit_breaker" yaml:"circuit_breaker"`
}

// String implements fmt.Stringer
func (p Params) String() string {
	return fmt.Sprintf(`Params:
	Global Debt Limit: %s
	Collateral Params: %s
	Debt Params: %s
	Surplus Auction Threshold: %s
	Surplus Auction Lot: %s
	Debt Auction Threshold: %s
	Debt Auction Lot: %s
	Circuit Breaker: %t`,
		p.GlobalDebtLimit, p.CollateralParams, p.DebtParam, p.SurplusAuctionThreshold, p.SurplusAuctionLot,
		p.DebtAuctionThreshold, p.DebtAuctionLot, p.CircuitBreaker,
	)
}

// NewParams returns a new params object
func NewParams(
	debtLimit sdk.Coin, collateralParams CollateralParams, debtParam DebtParam, surplusThreshold,
	surplusLot, debtThreshold, debtLot sdk.Int, breaker bool,
) Params {
	return Params{
		GlobalDebtLimit:         debtLimit,
		CollateralParams:        collateralParams,
		DebtParam:               debtParam,
		SurplusAuctionThreshold: surplusThreshold,
		SurplusAuctionLot:       surplusLot,
		DebtAuctionThreshold:    debtThreshold,
		DebtAuctionLot:          debtLot,
		CircuitBreaker:          breaker,
	}
}

// DefaultParams returns default params for cdp module
func DefaultParams() Params {
	return NewParams(
		DefaultGlobalDebt, DefaultCollateralParams, DefaultDebtParam, DefaultSurplusThreshold,
		DefaultSurplusLot, DefaultDebtThreshold, DefaultDebtLot,
		DefaultCircuitBreaker,
	)
}

// CollateralParam governance parameters for each collateral type within the cdp module
type CollateralParam struct {
	Denom                            string   `json:"denom" yaml:"denom"` // Coin name of collateral type
	Type                             string   `json:"type" yaml:"type"`
	LiquidationRatio                 sdk.Dec  `json:"liquidation_ratio" yaml:"liquidation_ratio"`     // The ratio (Collateral (priced in stable coin) / Debt) under which a CDP will be liquidated
	DebtLimit                        sdk.Coin `json:"debt_limit" yaml:"debt_limit"`                   // Maximum amount of debt allowed to be drawn from this collateral type
	StabilityFee                     sdk.Dec  `json:"stability_fee" yaml:"stability_fee"`             // per second stability fee for loans opened using this collateral
	AuctionSize                      sdk.Int  `json:"auction_size" yaml:"auction_size"`               // Max amount of collateral to sell off in any one auction.
	LiquidationPenalty               sdk.Dec  `json:"liquidation_penalty" yaml:"liquidation_penalty"` // percentage penalty (between [0, 1]) applied to a cdp if it is liquidated
	Prefix                           byte     `json:"prefix" yaml:"prefix"`
	SpotMarketID                     string   `json:"spot_market_id" yaml:"spot_market_id"`                                           // marketID of the spot price of the asset from the pricefeed - used for opening CDPs, depositing, withdrawing
	LiquidationMarketID              string   `json:"liquidation_market_id" yaml:"liquidation_market_id"`                             // marketID of the pricefeed used for liquidation
	KeeperRewardPercentage           sdk.Dec  `json:"keeper_reward_percentage" yaml:"keeper_reward_percentage"`                       // the percentage of a CDPs collateral that gets rewarded to a keeper that liquidates the position
	CheckCollateralizationIndexCount sdk.Int  `json:"check_collateralization_index_count" yaml:"check_collateralization_index_count"` // the number of cdps that will be checked for liquidation in the begin blocker
	ConversionFactor                 sdk.Int  `json:"conversion_factor" yaml:"conversion_factor"`                                     // factor for converting internal units to one base unit of collateral
}

// NewCollateralParam returns a new CollateralParam
func NewCollateralParam(
	denom, ctype string, liqRatio sdk.Dec, debtLimit sdk.Coin, stabilityFee sdk.Dec, auctionSize sdk.Int,
	liqPenalty sdk.Dec, prefix byte, spotMarketID, liquidationMarketID string, keeperReward sdk.Dec, checkIndexCount sdk.Int, conversionFactor sdk.Int) CollateralParam {
	return CollateralParam{
		Denom:                            denom,
		Type:                             ctype,
		LiquidationRatio:                 liqRatio,
		DebtLimit:                        debtLimit,
		StabilityFee:                     stabilityFee,
		AuctionSize:                      auctionSize,
		LiquidationPenalty:               liqPenalty,
		Prefix:                           prefix,
		SpotMarketID:                     spotMarketID,
		LiquidationMarketID:              liquidationMarketID,
		KeeperRewardPercentage:           keeperReward,
		CheckCollateralizationIndexCount: checkIndexCount,
		ConversionFactor:                 conversionFactor,
	}
}

// String implements fmt.Stringer
func (cp CollateralParam) String() string {
	return fmt.Sprintf(`Collateral:
	Denom: %s
	Type: %s
	Liquidation Ratio: %s
	Stability Fee: %s
	Liquidation Penalty: %s
	Debt Limit: %s
	Auction Size: %s
	Prefix: %b
	Spot Market ID: %s
	Liquidation Market ID: %s
	Keeper Reward Percentage: %s
	Check Collateralization Count: %s
	Conversion Factor: %s`,
		cp.Denom, cp.Type, cp.LiquidationRatio, cp.StabilityFee, cp.LiquidationPenalty,
		cp.DebtLimit, cp.AuctionSize, cp.Prefix, cp.SpotMarketID, cp.LiquidationMarketID,
		cp.KeeperRewardPercentage, cp.CheckCollateralizationIndexCount, cp.ConversionFactor)
}

// CollateralParams array of CollateralParam
type CollateralParams []CollateralParam

// String implements fmt.Stringer
func (cps CollateralParams) String() string {
	out := "Collateral Params\n"
	for _, cp := range cps {
		out += fmt.Sprintf("%s\n", cp)
	}
	return out
}

// DebtParam governance params for debt assets
type DebtParam struct {
	Denom            string  `json:"denom" yaml:"denom"`
	ReferenceAsset   string  `json:"reference_asset" yaml:"reference_asset"`
	ConversionFactor sdk.Int `json:"conversion_factor" yaml:"conversion_factor"`
	DebtFloor        sdk.Int `json:"debt_floor" yaml:"debt_floor"` // minimum active loan size, used to prevent dust
}

// NewDebtParam returns a new DebtParam
func NewDebtParam(denom, refAsset string, conversionFactor, debtFloor sdk.Int) DebtParam {
	return DebtParam{
		Denom:            denom,
		ReferenceAsset:   refAsset,
		ConversionFactor: conversionFactor,
		DebtFloor:        debtFloor,
	}
}

func (dp DebtParam) String() string {
	return fmt.Sprintf(`Debt:
	Denom: %s
	Reference Asset: %s
	Conversion Factor: %s
	Debt Floor %s
	`, dp.Denom, dp.ReferenceAsset, dp.ConversionFactor, dp.DebtFloor)
}

// DebtParams array of DebtParam
type DebtParams []DebtParam

// String implements fmt.Stringer
func (dps DebtParams) String() string {
	out := "Debt Params\n"
	for _, dp := range dps {
		out += fmt.Sprintf("%s\n", dp)
	}
	return out
}

// ParamKeyTable Key declaration for parameters
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of auth module's parameters.
// nolint
func (p *Params) ParamSetPairs() params.ParamSetPairs {
	return params.ParamSetPairs{
		params.NewParamSetPair(KeyGlobalDebtLimit, &p.GlobalDebtLimit, validateGlobalDebtLimitParam),
		params.NewParamSetPair(KeyCollateralParams, &p.CollateralParams, validateCollateralParams),
		params.NewParamSetPair(KeyDebtParam, &p.DebtParam, validateDebtParam),
		params.NewParamSetPair(KeyCircuitBreaker, &p.CircuitBreaker, validateCircuitBreakerParam),
		params.NewParamSetPair(KeySurplusThreshold, &p.SurplusAuctionThreshold, validateSurplusAuctionThresholdParam),
		params.NewParamSetPair(KeySurplusLot, &p.SurplusAuctionLot, validateSurplusAuctionLotParam),
		params.NewParamSetPair(KeyDebtThreshold, &p.DebtAuctionThreshold, validateDebtAuctionThresholdParam),
		params.NewParamSetPair(KeyDebtLot, &p.DebtAuctionLot, validateDebtAuctionLotParam),
	}
}

// Validate checks that the parameters have valid values.
func (p Params) Validate() error {
	if err := validateGlobalDebtLimitParam(p.GlobalDebtLimit); err != nil {
		return err
	}

	if err := validateCollateralParams(p.CollateralParams); err != nil {
		return err
	}

	if err := validateDebtParam(p.DebtParam); err != nil {
		return err
	}

	if err := validateCircuitBreakerParam(p.CircuitBreaker); err != nil {
		return err
	}

	if err := validateSurplusAuctionThresholdParam(p.SurplusAuctionThreshold); err != nil {
		return err
	}

	if err := validateSurplusAuctionLotParam(p.SurplusAuctionLot); err != nil {
		return err
	}

	if err := validateDebtAuctionThresholdParam(p.DebtAuctionThreshold); err != nil {
		return err
	}

	if err := validateDebtAuctionLotParam(p.DebtAuctionLot); err != nil {
		return err
	}

	if len(p.CollateralParams) == 0 { // default value OK
		return nil
	}

	if (DebtParam{}) != p.DebtParam {
		if p.DebtParam.Denom != p.GlobalDebtLimit.Denom {
			return fmt.Errorf("debt denom %s does not match global debt denom %s",
				p.DebtParam.Denom, p.GlobalDebtLimit.Denom)
		}
	}

	// validate collateral params
	collateralDupMap := make(map[string]int)
	prefixDupMap := make(map[int]int)
	collateralParamsDebtLimit := sdk.ZeroInt()

	for _, cp := range p.CollateralParams {

		prefix := int(cp.Prefix)
		prefixDupMap[prefix] = 1
		collateralDupMap[cp.Denom] = 1

		if cp.DebtLimit.Denom != p.GlobalDebtLimit.Denom {
			return fmt.Errorf("collateral debt limit denom %s does not match global debt limit denom %s",
				cp.DebtLimit.Denom, p.GlobalDebtLimit.Denom)
		}

		collateralParamsDebtLimit = collateralParamsDebtLimit.Add(cp.DebtLimit.Amount)

		if cp.DebtLimit.Amount.GT(p.GlobalDebtLimit.Amount) {
			return fmt.Errorf("collateral debt limit %s exceeds global debt limit: %s", cp.DebtLimit, p.GlobalDebtLimit)
		}
	}

	if collateralParamsDebtLimit.GT(p.GlobalDebtLimit.Amount) {
		return fmt.Errorf("sum of collateral debt limits %s exceeds global debt limit %s",
			collateralParamsDebtLimit, p.GlobalDebtLimit)
	}

	return nil
}

func validateGlobalDebtLimitParam(i interface{}) error {
	globalDebtLimit, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !globalDebtLimit.IsValid() {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "global debt limit %s", globalDebtLimit.String())
	}

	return nil
}

func validateCollateralParams(i interface{}) error {
	collateralParams, ok := i.(CollateralParams)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	prefixDupMap := make(map[int]bool)
	typeDupMap := make(map[string]bool)
	for _, cp := range collateralParams {
		if err := sdk.ValidateDenom(cp.Denom); err != nil {
			return fmt.Errorf("collateral denom invalid %s", cp.Denom)
		}

		if strings.TrimSpace(cp.SpotMarketID) == "" {
			return fmt.Errorf("spot market id cannot be blank %s", cp)
		}

		if strings.TrimSpace(cp.Type) == "" {
			return fmt.Errorf("collateral type cannot be blank %s", cp)
		}

		if strings.TrimSpace(cp.LiquidationMarketID) == "" {
			return fmt.Errorf("liquidation market id cannot be blank %s", cp)
		}

		prefix := int(cp.Prefix)
		if prefix < minCollateralPrefix || prefix > maxCollateralPrefix {
			return fmt.Errorf("invalid prefix for collateral denom %s: %b", cp.Denom, cp.Prefix)
		}

		_, found := prefixDupMap[prefix]
		if found {
			return fmt.Errorf("duplicate prefix for collateral denom %s: %v", cp.Denom, []byte{cp.Prefix})
		}

		prefixDupMap[prefix] = true

		_, found = typeDupMap[cp.Type]
		if found {
			return fmt.Errorf("duplicate cdp collateral type: %s", cp.Type)
		}
		typeDupMap[cp.Type] = true

		if !cp.DebtLimit.IsValid() {
			return fmt.Errorf("debt limit for all collaterals should be positive, is %s for %s", cp.DebtLimit, cp.Denom)
		}

		if cp.LiquidationRatio.IsNil() || !cp.LiquidationRatio.IsPositive() {
			return fmt.Errorf("liquidation ratio must be > 0")
		}

		if cp.LiquidationPenalty.LT(sdk.ZeroDec()) || cp.LiquidationPenalty.GT(sdk.OneDec()) {
			return fmt.Errorf("liquidation penalty should be between 0 and 1, is %s for %s", cp.LiquidationPenalty, cp.Denom)
		}
		if !cp.AuctionSize.IsPositive() {
			return fmt.Errorf("auction size should be positive, is %s for %s", cp.AuctionSize, cp.Denom)
		}
		if cp.StabilityFee.LT(sdk.OneDec()) || cp.StabilityFee.GT(stabilityFeeMax) {
			return fmt.Errorf("stability fee must be ≥ 1.0, ≤ %s, is %s for %s", stabilityFeeMax, cp.StabilityFee, cp.Denom)
		}
		if cp.KeeperRewardPercentage.IsNegative() || cp.KeeperRewardPercentage.GT(sdk.OneDec()) {
			return fmt.Errorf("keeper reward percentage should be between 0 and 1, is %s for %s", cp.KeeperRewardPercentage, cp.Denom)
		}
		if cp.CheckCollateralizationIndexCount.IsNegative() {
			return fmt.Errorf("keeper reward percentage should be positive, is %s for %s", cp.CheckCollateralizationIndexCount, cp.Denom)
		}
	}

	return nil
}

func validateDebtParam(i interface{}) error {
	debtParam, ok := i.(DebtParam)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if err := sdk.ValidateDenom(debtParam.Denom); err != nil {
		return fmt.Errorf("debt denom invalid %s", debtParam.Denom)
	}

	return nil
}

func validateCircuitBreakerParam(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateSurplusAuctionThresholdParam(i interface{}) error {
	sat, ok := i.(sdk.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !sat.IsPositive() {
		return fmt.Errorf("surplus auction threshold should be positive: %s", sat)
	}

	return nil
}

func validateSurplusAuctionLotParam(i interface{}) error {
	sal, ok := i.(sdk.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !sal.IsPositive() {
		return fmt.Errorf("surplus auction lot should be positive: %s", sal)
	}

	return nil
}

func validateDebtAuctionThresholdParam(i interface{}) error {
	dat, ok := i.(sdk.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !dat.IsPositive() {
		return fmt.Errorf("debt auction threshold should be positive: %s", dat)
	}

	return nil
}

func validateDebtAuctionLotParam(i interface{}) error {
	dal, ok := i.(sdk.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !dal.IsPositive() {
		return fmt.Errorf("debt auction lot should be positive: %s", dal)
	}

	return nil
}
