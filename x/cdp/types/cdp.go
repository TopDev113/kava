package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CDP is the state of a single collateralized debt position.
type CDP struct {
	ID              uint64         `json:"id" yaml:"id"`                             // unique id for cdp
	Owner           sdk.AccAddress `json:"owner" yaml:"owner"`                       // Account that authorizes changes to the CDP
	Type            string         `json:"type" yaml:"type"`                         // string representing the unique collateral type of the CDP
	Collateral      sdk.Coin       `json:"collateral" yaml:"collateral"`             // Amount of collateral stored in this CDP
	Principal       sdk.Coin       `json:"principal" yaml:"principal"`               // Amount of debt drawn using the CDP
	AccumulatedFees sdk.Coin       `json:"accumulated_fees" yaml:"accumulated_fees"` // Fees accumulated since the CDP was opened or debt was last repaid
	FeesUpdated     time.Time      `json:"fees_updated" yaml:"fees_updated"`         // The time when fees were last updated
	InterestFactor  sdk.Dec        `json:"interest_factor" yaml:"interest_factor"`   // the interest factor when fees were last calculated for this CDP
}

// NewCDP creates a new CDP object
func NewCDP(id uint64, owner sdk.AccAddress, collateral sdk.Coin, collateralType string, principal sdk.Coin, time time.Time, interestFactor sdk.Dec) CDP {
	fees := sdk.NewCoin(principal.Denom, sdk.ZeroInt())
	return CDP{
		ID:              id,
		Owner:           owner,
		Type:            collateralType,
		Collateral:      collateral,
		Principal:       principal,
		AccumulatedFees: fees,
		FeesUpdated:     time,
		InterestFactor:  interestFactor,
	}
}

// NewCDPWithFees creates a new CDP object, for use during migration
func NewCDPWithFees(id uint64, owner sdk.AccAddress, collateral sdk.Coin, collateralType string, principal, fees sdk.Coin, time time.Time, interestFactor sdk.Dec) CDP {
	return CDP{
		ID:              id,
		Owner:           owner,
		Type:            collateralType,
		Collateral:      collateral,
		Principal:       principal,
		AccumulatedFees: fees,
		FeesUpdated:     time,
		InterestFactor:  interestFactor,
	}
}

// String implements fmt.stringer
func (cdp CDP) String() string {
	return strings.TrimSpace(fmt.Sprintf(`CDP:
	Owner:      %s
	ID: %d
	Collateral Type: %s
	Collateral: %s
	Principal: %s
	AccumulatedFees: %s
	Fees Last Updated: %s
	Interest Factor: %s`,
		cdp.Owner,
		cdp.ID,
		cdp.Type,
		cdp.Collateral,
		cdp.Principal,
		cdp.AccumulatedFees,
		cdp.FeesUpdated,
		cdp.InterestFactor,
	))
}

// Validate performs a basic validation of the CDP fields.
func (cdp CDP) Validate() error {
	if cdp.ID == 0 {
		return errors.New("cdp id cannot be 0")
	}
	if cdp.Owner.Empty() {
		return errors.New("cdp owner cannot be empty")
	}
	if !cdp.Collateral.IsValid() {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "collateral %s", cdp.Collateral)
	}
	if !cdp.Principal.IsValid() {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "principal %s", cdp.Principal)
	}
	if !cdp.AccumulatedFees.IsValid() {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "accumulated fees %s", cdp.AccumulatedFees)
	}
	if cdp.FeesUpdated.Unix() <= 0 {
		return errors.New("cdp updated fee time cannot be zero")
	}
	if strings.TrimSpace(cdp.Type) == "" {
		return fmt.Errorf("cdp type cannot be empty")
	}
	return nil
}

// GetTotalPrincipal returns the total principle for the cdp
func (cdp CDP) GetTotalPrincipal() sdk.Coin {
	return cdp.Principal.Add(cdp.AccumulatedFees)
}

// GetNormalizedPrincipal returns the total cdp principal divided by the interest factor.
//
// Multiplying the normalized principal by the current global factor gives the current debt (ie including all interest, ie a synced cdp).
// The normalized principal is effectively how big the principal would have been if it had been borrowed at time 0 and not touched since.
//
// An error is returned if the cdp interest factor is in an invalid state.
func (cdp CDP) GetNormalizedPrincipal() (sdk.Dec, error) {
	unsyncedDebt := cdp.GetTotalPrincipal().Amount
	if cdp.InterestFactor.LT(sdk.OneDec()) {
		return sdk.Dec{}, fmt.Errorf("interest factor '%s' must be ≥ 1", cdp.InterestFactor)
	}
	return unsyncedDebt.ToDec().Quo(cdp.InterestFactor), nil
}

// CDPs a collection of CDP objects
type CDPs []CDP

// String implements stringer
func (cdps CDPs) String() string {
	out := ""
	for _, cdp := range cdps {
		out += cdp.String() + "\n"
	}
	return out
}

// Validate validates each CDP
func (cdps CDPs) Validate() error {
	for _, cdp := range cdps {
		if err := cdp.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// AugmentedCDP provides additional information about an active CDP
type AugmentedCDP struct {
	CDP                    `json:"cdp" yaml:"cdp"`
	CollateralValue        sdk.Coin `json:"collateral_value" yaml:"collateral_value"`               // collateral's market value in debt coin
	CollateralizationRatio sdk.Dec  `json:"collateralization_ratio" yaml:"collateralization_ratio"` // current collateralization ratio
}

// NewAugmentedCDP creates a new AugmentedCDP object
func NewAugmentedCDP(cdp CDP, collateralValue sdk.Coin, collateralizationRatio sdk.Dec) AugmentedCDP {
	augmentedCDP := AugmentedCDP{
		CDP: CDP{
			ID:              cdp.ID,
			Owner:           cdp.Owner,
			Type:            cdp.Type,
			Collateral:      cdp.Collateral,
			Principal:       cdp.Principal,
			AccumulatedFees: cdp.AccumulatedFees,
			FeesUpdated:     cdp.FeesUpdated,
			InterestFactor:  cdp.InterestFactor,
		},
		CollateralValue:        collateralValue,
		CollateralizationRatio: collateralizationRatio,
	}
	return augmentedCDP
}

// String implements fmt.stringer
func (augCDP AugmentedCDP) String() string {
	return strings.TrimSpace(fmt.Sprintf(`AugmentedCDP:
	Owner:      %s
	ID: %d
	Collateral Type: %s
	Collateral: %s
	Collateral Value: %s
	Principal: %s
	Fees: %s
	Fees Last Updated: %s
	Interest Factor: %s
	Collateralization ratio: %s`,
		augCDP.Owner,
		augCDP.ID,
		augCDP.Type,
		augCDP.Collateral,
		augCDP.CollateralValue,
		augCDP.Principal,
		augCDP.AccumulatedFees,
		augCDP.FeesUpdated,
		augCDP.InterestFactor,
		augCDP.CollateralizationRatio,
	))
}

// AugmentedCDPs a collection of AugmentedCDP objects
type AugmentedCDPs []AugmentedCDP

// String implements stringer
func (augcdps AugmentedCDPs) String() string {
	out := ""
	for _, augcdp := range augcdps {
		out += augcdp.String() + "\n"
	}
	return out
}

// TotalCDPPrincipal is a total principal of a given collateral type
type TotalCDPPrincipal struct {
	CollateralType string   `json:"collateral_type" yaml:"collateral_type"` // string representing the unique collateral type of the CDP
	Amount         sdk.Coin `json:"amount" yaml:"amount"`                   // Amount of principal stored in this CDP
}

// TotalCDPPrincipal returns a new TotalCDPPrincipal
func NewTotalCDPPrincipal(collateralType string, amount sdk.Coin) TotalCDPPrincipal {
	return TotalCDPPrincipal{
		CollateralType: collateralType,
		Amount:         amount,
	}
}

// TotalCDPCollateral is a total principal of a given collateral type
type TotalCDPCollateral struct {
	CollateralType string   `json:"collateral_type" yaml:"collateral_type"` // string representing the unique collateral type of the CDP
	Amount         sdk.Coin `json:"amount" yaml:"amount"`                   // Amount of collateral stored in this CDP
}

// TotalCDPCollateral returns a new TotalCDPCollateral
func NewTotalCDPCollateral(collateralType string, amount sdk.Coin) TotalCDPCollateral {
	return TotalCDPCollateral{
		CollateralType: collateralType,
		Amount:         amount,
	}
}
