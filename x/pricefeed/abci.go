package pricefeed

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker updates the current pricefeed
func EndBlocker(ctx sdk.Context, k Keeper) {
	// Update the current price of each asset.
	for _, market := range k.GetMarkets(ctx) {
		if market.Active {
			err := k.SetCurrentPrices(ctx, market.MarketID)
			if err != nil {
				// In the event of failure, emit an event.
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						EventTypeNoValidPrices,
						sdk.NewAttribute(AttributeMarketID, fmt.Sprintf("%s", market.MarketID)),
					),
				)
				continue
			}
		}
	}
	return
}
