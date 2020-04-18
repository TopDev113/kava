package types

import (
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Market an asset in the pricefeed
type Market struct {
	MarketID   string           `json:"market_id" yaml:"market_id"`
	BaseAsset  string           `json:"base_asset" yaml:"base_asset"`
	QuoteAsset string           `json:"quote_asset" yaml:"quote_asset"`
	Oracles    []sdk.AccAddress `json:"oracles" yaml:"oracles"`
	Active     bool             `json:"active" yaml:"active"`
}

// String implement fmt.Stringer
func (a Market) String() string {
	return fmt.Sprintf(`Asset:
	Market ID: %s
	Base Asset: %s
	Quote Asset: %s
	Oracles: %s
	Active: %t`,
		a.MarketID, a.BaseAsset, a.QuoteAsset, a.Oracles, a.Active)
}

// Markets array type for oracle
type Markets []Market

// String implements fmt.Stringer
func (ms Markets) String() string {
	out := "Markets:\n"
	for _, m := range ms {
		out += fmt.Sprintf("%s\n", m.String())
	}
	return strings.TrimSpace(out)
}

// CurrentPrice struct that contains the metadata of a current price for a particular market in the pricefeed module.
type CurrentPrice struct {
	MarketID string  `json:"market_id" yaml:"market_id"`
	Price    sdk.Dec `json:"price" yaml:"price"`
}

// NewCurrentPrice returns an instance of CurrentPrice
func NewCurrentPrice(marketID string, price sdk.Dec) CurrentPrice {
	return CurrentPrice{MarketID: marketID, Price: price}
}

// CurrentPrices type for an array of CurrentPrice
type CurrentPrices []CurrentPrice

// PostedPrice price for market posted by a specific oracle
type PostedPrice struct {
	MarketID      string         `json:"market_id" yaml:"market_id"`
	OracleAddress sdk.AccAddress `json:"oracle_address" yaml:"oracle_address"`
	Price         sdk.Dec        `json:"price" yaml:"price"`
	Expiry        time.Time      `json:"expiry" yaml:"expiry"`
}

// NewPostedPrice returns a new PostedPrice
func NewPostedPrice(marketID string, oracle sdk.AccAddress, price sdk.Dec, expiry time.Time) PostedPrice {
	return PostedPrice{
		MarketID:      marketID,
		OracleAddress: oracle,
		Price:         price,
		Expiry:        expiry,
	}
}

// PostedPrices type for an array of PostedPrice
type PostedPrices []PostedPrice

// implement fmt.Stringer
func (cp CurrentPrice) String() string {
	return strings.TrimSpace(fmt.Sprintf(`Market ID: %s
Price: %s`, cp.MarketID, cp.Price))
}

// implement fmt.Stringer
func (pp PostedPrice) String() string {
	return strings.TrimSpace(fmt.Sprintf(`Market ID: %s
Oracle Address: %s
Price: %s
Expiry: %s`, pp.MarketID, pp.OracleAddress, pp.Price, pp.Expiry))
}

// String implements fmt.Stringer
func (ps PostedPrices) String() string {
	out := "Posted Prices:\n"
	for _, p := range ps {
		out += fmt.Sprintf("%s\n", p.String())
	}
	return strings.TrimSpace(out)
}

// SortDecs provides the interface needed to sort sdk.Dec slices
type SortDecs []sdk.Dec

func (a SortDecs) Len() int           { return len(a) }
func (a SortDecs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortDecs) Less(i, j int) bool { return a[i].LT(a[j]) }
