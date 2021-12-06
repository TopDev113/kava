package v0_15

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v40auth "github.com/cosmos/cosmos-sdk/x/auth/legacy/v040"
)

const (
	CollateralAuctionType = "collateral"
	SurplusAuctionType    = "surplus"
	DebtAuctionType       = "debt"
	ForwardAuctionPhase   = "forward"
	ReverseAuctionPhase   = "reverse"
)

// Auction is an interface for handling common actions on auctions.
type Auction interface {
	GetID() uint64
	WithID(uint64) Auction

	GetInitiator() string
	GetLot() sdk.Coin
	GetBidder() sdk.AccAddress
	GetBid() sdk.Coin
	GetEndTime() time.Time
	GetHasReceivedBids() bool
	GetMaxEndTime() time.Time

	GetType() string
	GetPhase() string
}

var (
	_ Auction        = &SurplusAuction{}
	_ GenesisAuction = &SurplusAuction{}
	_ Auction        = &DebtAuction{}
	_ GenesisAuction = &DebtAuction{}
	_ Auction        = &CollateralAuction{}
	_ GenesisAuction = &CollateralAuction{}
)

// BaseAuction is a common type shared by all Auctions.
type BaseAuction struct {
	ID              uint64         `json:"id" yaml:"id"`
	Initiator       string         `json:"initiator" yaml:"initiator"`                 // Module name that starts the auction. Pays out Lot.
	Lot             sdk.Coin       `json:"lot" yaml:"lot"`                             // Coins that will paid out by Initiator to the winning bidder.
	Bidder          sdk.AccAddress `json:"bidder" yaml:"bidder"`                       // Latest bidder. Receiver of Lot.
	Bid             sdk.Coin       `json:"bid" yaml:"bid"`                             // Coins paid into the auction the bidder.
	HasReceivedBids bool           `json:"has_received_bids" yaml:"has_received_bids"` // Whether the auction has received any bids or not.
	EndTime         time.Time      `json:"end_time" yaml:"end_time"`                   // Current auction closing time. Triggers at the end of the block with time ≥ EndTime.
	MaxEndTime      time.Time      `json:"max_end_time" yaml:"max_end_time"`           // Maximum closing time. Auctions can close before this but never after.
}

// GetID is a getter for auction ID.
func (a BaseAuction) GetID() uint64 { return a.ID }

// GetInitiator is a getter for auction Initiator.
func (a BaseAuction) GetInitiator() string { return a.Initiator }

// GetLot is a getter for auction Lot.
func (a BaseAuction) GetLot() sdk.Coin { return a.Lot }

// GetBidder is a getter for auction Bidder.
func (a BaseAuction) GetBidder() sdk.AccAddress { return a.Bidder }

// GetBid is a getter for auction Bid.
func (a BaseAuction) GetBid() sdk.Coin { return a.Bid }

// GetEndTime is a getter for auction end time.
func (a BaseAuction) GetEndTime() time.Time { return a.EndTime }

// GetType returns the auction type. Used to identify auctions in event attributes.
func (a BaseAuction) GetType() string { return "base" }

// GetMaxEndTime is a getter for MaxEndTime
func (a BaseAuction) GetMaxEndTime() time.Time { return a.MaxEndTime }

// GetMaxEndTime is a getter for GetHasReceivedBids
func (a BaseAuction) GetHasReceivedBids() bool { return a.HasReceivedBids }

// Validate verifies that the auction end time is before max end time
func (a BaseAuction) Validate() error {
	// ID can be 0 for surplus, debt and collateral auctions
	if strings.TrimSpace(a.Initiator) == "" {
		return errors.New("auction initiator cannot be blank")
	}
	if !a.Lot.IsValid() {
		return fmt.Errorf("invalid lot: %s", a.Lot)
	}
	// NOTE: bidder can be empty for Surplus and Collateral auctions
	if !a.Bidder.Empty() && len(a.Bidder) != v40auth.AddrLen {
		return fmt.Errorf("the expected bidder address length is %d, actual length is %d", v40auth.AddrLen, len(a.Bidder))
	}
	if !a.Bid.IsValid() {
		return fmt.Errorf("invalid bid: %s", a.Bid)
	}
	if a.EndTime.Unix() <= 0 || a.MaxEndTime.Unix() <= 0 {
		return errors.New("end time cannot be zero")
	}
	if a.EndTime.After(a.MaxEndTime) {
		return fmt.Errorf("MaxEndTime < EndTime (%s < %s)", a.MaxEndTime, a.EndTime)
	}
	return nil
}

// DebtAuction is a reverse auction that mints what it pays out.
// It is normally used to acquire pegged asset to cover the CDP system's debts that were not covered by selling collateral.
type DebtAuction struct {
	BaseAuction `json:"base_auction" yaml:"base_auction"`

	CorrespondingDebt sdk.Coin `json:"corresponding_debt" yaml:"corresponding_debt"`
}

// WithID returns an auction with the ID set.
func (a DebtAuction) WithID(id uint64) Auction { a.ID = id; return a }

// GetType returns the auction type. Used to identify auctions in event attributes.
func (a DebtAuction) GetType() string { return DebtAuctionType }

// GetModuleAccountCoins returns the total number of coins held in the module account for this auction.
// It is used in genesis initialize the module account correctly.
func (a DebtAuction) GetModuleAccountCoins() sdk.Coins {
	// a.Lot is minted at auction close, so is never stored in the module account
	// a.Bid is paid out on bids, so is never stored in the module account
	return sdk.NewCoins(a.CorrespondingDebt)
}

// GetPhase returns the direction of a debt auction, which never changes.
func (a DebtAuction) GetPhase() string { return ReverseAuctionPhase }

// Validate validates the DebtAuction fields values.
func (a DebtAuction) Validate() error {
	if !a.CorrespondingDebt.IsValid() {
		return fmt.Errorf("invalid corresponding debt: %s", a.CorrespondingDebt)
	}
	return a.BaseAuction.Validate()
}

// SurplusAuction is a forward auction that burns what it receives from bids.
// It is normally used to sell off excess pegged asset acquired by the CDP system.
type SurplusAuction struct {
	BaseAuction `json:"base_auction" yaml:"base_auction"`
}

// WithID returns an auction with the ID set.
func (a SurplusAuction) WithID(id uint64) Auction { a.ID = id; return a }

// GetType returns the auction type. Used to identify auctions in event attributes.
func (a SurplusAuction) GetType() string { return SurplusAuctionType }

// GetModuleAccountCoins returns the total number of coins held in the module account for this auction.
// It is used in genesis initialize the module account correctly.
func (a SurplusAuction) GetModuleAccountCoins() sdk.Coins {
	// a.Bid is paid out on bids, so is never stored in the module account
	return sdk.NewCoins(a.Lot)
}

// GetPhase returns the direction of a surplus auction, which never changes.
func (a SurplusAuction) GetPhase() string { return ForwardAuctionPhase }

// CollateralAuction is a two phase auction.
// Initially, in forward auction phase, bids can be placed up to a max bid.
// Then it switches to a reverse auction phase, where the initial amount up for auction is bid down.
// Unsold Lot is sent to LotReturns, being divided among the addresses by weight.
// Collateral auctions are normally used to sell off collateral seized from CDPs.
type CollateralAuction struct {
	BaseAuction `json:"base_auction" yaml:"base_auction"`

	CorrespondingDebt sdk.Coin          `json:"corresponding_debt" yaml:"corresponding_debt"`
	MaxBid            sdk.Coin          `json:"max_bid" yaml:"max_bid"`
	LotReturns        WeightedAddresses `json:"lot_returns" yaml:"lot_returns"`
}

// WithID returns an auction with the ID set.
func (a CollateralAuction) WithID(id uint64) Auction { a.ID = id; return a }

// GetType returns the auction type. Used to identify auctions in event attributes.
func (a CollateralAuction) GetType() string { return CollateralAuctionType }

// GetModuleAccountCoins returns the total number of coins held in the module account for this auction.
// It is used in genesis initialize the module account correctly.
func (a CollateralAuction) GetModuleAccountCoins() sdk.Coins {
	// a.Bid is paid out on bids, so is never stored in the module account
	return sdk.NewCoins(a.Lot).Add(sdk.NewCoins(a.CorrespondingDebt)...)
}

// IsReversePhase returns whether the auction has switched over to reverse phase or not.
// CollateralAuctions initially start in forward phase.
func (a CollateralAuction) IsReversePhase() bool {
	return a.Bid.IsEqual(a.MaxBid)
}

// GetPhase returns the direction of a collateral auction.
func (a CollateralAuction) GetPhase() string {
	if a.IsReversePhase() {
		return ReverseAuctionPhase
	}
	return ForwardAuctionPhase
}

// GetLotReturns returns a collateral auction's lot owners
func (a CollateralAuction) GetLotReturns() WeightedAddresses {
	return a.LotReturns
}

// Validate validates the CollateralAuction fields values.
func (a CollateralAuction) Validate() error {
	if !a.CorrespondingDebt.IsValid() {
		return fmt.Errorf("invalid corresponding debt: %s", a.CorrespondingDebt)
	}
	if !a.MaxBid.IsValid() {
		return fmt.Errorf("invalid max bid: %s", a.MaxBid)
	}
	if err := a.LotReturns.Validate(); err != nil {
		return fmt.Errorf("invalid lot returns: %w", err)
	}
	return a.BaseAuction.Validate()
}

// WeightedAddresses is a type for storing some addresses and associated weights.
type WeightedAddresses struct {
	Addresses []sdk.AccAddress `json:"addresses" yaml:"addresses"`
	Weights   []sdk.Int        `json:"weights" yaml:"weights"`
}

// Validate checks for that the weights are not negative, not all zero, and the lengths match.
func (wa WeightedAddresses) Validate() error {
	if len(wa.Weights) < 1 {
		return fmt.Errorf("must be at least 1 weighted address")
	}

	if len(wa.Addresses) != len(wa.Weights) {
		return fmt.Errorf("number of addresses doesn't match number of weights, %d ≠ %d", len(wa.Addresses), len(wa.Weights))
	}

	totalWeight := sdk.ZeroInt()
	for i := range wa.Addresses {
		if wa.Addresses[i].Empty() {
			return fmt.Errorf("address %d cannot be empty", i)
		}
		if len(wa.Addresses[i]) != v40auth.AddrLen {
			return fmt.Errorf("address %d has an invalid length: expected %d, got %d", i, v40auth.AddrLen, len(wa.Addresses[i]))
		}
		if wa.Weights[i].IsNegative() {
			return fmt.Errorf("weight %d contains a negative amount: %s", i, wa.Weights[i])
		}
		totalWeight = totalWeight.Add(wa.Weights[i])
	}

	if !totalWeight.IsPositive() {
		return fmt.Errorf("total weight must be positive")
	}

	return nil
}
