package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	tmtime "github.com/tendermint/tendermint/types/time"
)

// Parameter keys
var (
	KeyGlobalDebtLimit       = []byte("GlobalDebtLimit")
	KeyCollateralParams      = []byte("CollateralParams")
	KeyDebtParams            = []byte("DebtParams")
	KeyCircuitBreaker        = []byte("CircuitBreaker")
	DefaultGlobalDebt        = sdk.Coins{}
	DefaultCircuitBreaker    = false
	DefaultCollateralParams  = CollateralParams{}
	DefaultDebtParams        = DebtParams{}
	DefaultCdpStartingID     = uint64(1)
	DefaultDebtDenom         = "debt"
	DefaultPreviousBlockTime = tmtime.Canonical(time.Unix(0, 0))
	minCollateralPrefix      = 0
	maxCollateralPrefix      = 255
)

// Params governance parameters for cdp module
type Params struct {
	CollateralParams CollateralParams `json:"collateral_params" yaml:"collateral_params"`
	DebtParams       DebtParams       `json:"debt_params" yaml:"debt_params"`
	GlobalDebtLimit  sdk.Coins        `json:"global_debt_limit" yaml:"global_debt_limit"`
	CircuitBreaker   bool             `json:"circuit_breaker" yaml:"circuit_breaker"`
}

// String implements fmt.Stringer
func (p Params) String() string {
	return fmt.Sprintf(`Params:
	Global Debt Limit: %s
	Collateral Params: %s
	Debt Params: %s
	Circuit Breaker: %t`,
		p.GlobalDebtLimit, p.CollateralParams, p.DebtParams, p.CircuitBreaker,
	)
}

// NewParams returns a new params object
func NewParams(debtLimit sdk.Coins, collateralParams CollateralParams, debtParams DebtParams, breaker bool) Params {
	return Params{
		GlobalDebtLimit:  debtLimit,
		CollateralParams: collateralParams,
		DebtParams:       debtParams,
		CircuitBreaker:   breaker,
	}
}

// DefaultParams returns default params for cdp module
func DefaultParams() Params {
	return NewParams(DefaultGlobalDebt, DefaultCollateralParams, DefaultDebtParams, DefaultCircuitBreaker)
}

// CollateralParam governance parameters for each collateral type within the cdp module
type CollateralParam struct {
	Denom            string    `json:"denom" yaml:"denom"`                         // Coin name of collateral type
	LiquidationRatio sdk.Dec   `json:"liquidation_ratio" yaml:"liquidation_ratio"` // The ratio (Collateral (priced in stable coin) / Debt) under which a CDP will be liquidated
	DebtLimit        sdk.Coins `json:"debt_limit" yaml:"debt_limit"`               // Maximum amount of debt allowed to be drawn from this collateral type
	StabilityFee     sdk.Dec   `json:"stability_fee" yaml:"stability_fee"`         // per second stability fee for loans opened using this collateral
	Prefix           byte      `json:"prefix" yaml:"prefix"`
	MarketID         string    `json:"market_id" yaml:"market_id"`                 // marketID for fetching price of the asset from the pricefeed
	ConversionFactor sdk.Int   `json:"conversion_factor" yaml:"conversion_factor"` // factor for converting internal units to one base unit of collateral
}

// String implements fmt.Stringer
func (cp CollateralParam) String() string {
	return fmt.Sprintf(`Collateral:
	Denom: %s
	Liquidation Ratio: %s
	Stability Fee: %s
	Debt Limit: %s
	Prefix: %b
	Market ID: %s
	Conversion Factor: %s`,
		cp.Denom, cp.LiquidationRatio, cp.StabilityFee, cp.DebtLimit, cp.Prefix, cp.MarketID, cp.ConversionFactor)
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
	Denom            string    `json:"denom" yaml:"denom"`
	ReferenceAsset   string    `json:"reference_asset" yaml:"reference_asset"`
	DebtLimit        sdk.Coins `json:"debt_limit" yaml:"debt_limit"`
	ConversionFactor sdk.Int   `json:"conversion_factor" yaml:"conversion_factor"`
	DebtFloor        sdk.Int   `json:"debt_floor" yaml:"debt_floor"` // minimum active loan size, used to prevent dust
}

func (dp DebtParam) String() string {
	return fmt.Sprintf(`Debt:
	Denom: %s
	Reference Asset: %s
	Debt Limit: %s
	Conversion Factor: %s
	Debt Floot %s`, dp.Denom, dp.ReferenceAsset, dp.DebtLimit, dp.ConversionFactor, dp.DebtFloor)
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
		{Key: KeyGlobalDebtLimit, Value: &p.GlobalDebtLimit},
		{Key: KeyCollateralParams, Value: &p.CollateralParams},
		{Key: KeyDebtParams, Value: &p.DebtParams},
		{Key: KeyCircuitBreaker, Value: &p.CircuitBreaker},
	}
}

// Validate checks that the parameters have valid values.
func (p Params) Validate() error {
	debtDenoms := make(map[string]int)
	debtParamsDebtLimit := sdk.Coins{}
	for _, dp := range p.DebtParams {
		_, found := debtDenoms[dp.Denom]
		if found {
			return fmt.Errorf("duplicate debt denom: %s", dp.Denom)
		}
		debtDenoms[dp.Denom] = 1
		if dp.DebtLimit.IsAnyNegative() {
			return fmt.Errorf("debt limit for all debt tokens should be positive, is %s for %s", dp.DebtLimit, dp.Denom)
		}
		debtParamsDebtLimit = debtParamsDebtLimit.Add(dp.DebtLimit)
	}
	if !debtParamsDebtLimit.DenomsSubsetOf(p.GlobalDebtLimit) {
		return fmt.Errorf("debt denom not found in global debt limit:\n\tglobal debt limit: %s\n\tdebt limits: %s",
			p.GlobalDebtLimit, debtParamsDebtLimit)
	}
	if debtParamsDebtLimit.IsAnyGT(p.GlobalDebtLimit) {
		return fmt.Errorf("debt limit exceeds global debt limit:\n\tglobal debt limit: %s\n\tdebt limits: %s",
			p.GlobalDebtLimit, debtParamsDebtLimit)
	}

	collateralDupMap := make(map[string]int)
	prefixDupMap := make(map[int]int)
	collateralParamsDebtLimit := sdk.Coins{}
	for _, cp := range p.CollateralParams {
		prefix := int(cp.Prefix)
		if prefix < minCollateralPrefix || prefix > maxCollateralPrefix {
			return fmt.Errorf("invalid prefix for collateral denom %s: %b", cp.Denom, cp.Prefix)
		}
		_, found := prefixDupMap[prefix]
		if found {
			return fmt.Errorf("duplicate prefix for collateral denom %s: %v", cp.Denom, []byte{cp.Prefix})
		}

		prefixDupMap[prefix] = 1
		_, found = collateralDupMap[cp.Denom]

		if found {
			return fmt.Errorf("duplicate collateral denom: %s", cp.Denom)
		}
		collateralDupMap[cp.Denom] = 1

		if cp.DebtLimit.IsAnyNegative() {
			return fmt.Errorf("debt limit for all collaterals should be positive, is %s for %s", cp.DebtLimit, cp.Denom)
		}
		collateralParamsDebtLimit = collateralParamsDebtLimit.Add(cp.DebtLimit)

		for _, dc := range cp.DebtLimit {
			_, found := debtDenoms[dc.Denom]
			if !found {
				return fmt.Errorf("debt limit for collateral %s contains invalid debt denom %s", cp.Denom, dc.Denom)
			}
		}
		if cp.DebtLimit.IsAnyGT(p.GlobalDebtLimit) {
			return fmt.Errorf("collateral debt limit for %s exceeds global debt limit: \n\tglobal debt limit: %s\n\tcollateral debt limits: %s",
				cp.Denom, p.GlobalDebtLimit, cp.DebtLimit)
		}
	}
	if collateralParamsDebtLimit.IsAnyGT(p.GlobalDebtLimit) {
		return fmt.Errorf("collateral debt limit exceeds global debt limit:\n\tglobal debt limit: %s\n\tcollateral debt limits: %s",
			p.GlobalDebtLimit, collateralParamsDebtLimit)
	}

	if p.GlobalDebtLimit.IsAnyNegative() {
		return fmt.Errorf("global debt limit should be positive for all debt tokens, is %s", p.GlobalDebtLimit)
	}
	return nil
}
