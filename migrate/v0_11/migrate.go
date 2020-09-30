package v0_11

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	v0_11bep3 "github.com/kava-labs/kava/x/bep3/legacy/v0_11"
	v0_9bep3 "github.com/kava-labs/kava/x/bep3/legacy/v0_9"
	v0_11pricefeed "github.com/kava-labs/kava/x/pricefeed"
	v0_9pricefeed "github.com/kava-labs/kava/x/pricefeed/legacy/v0_9"
)

// MigrateBep3 migrates from a v0.9 (or v0.10) bep3 genesis state to a v0.11 bep3 genesis state
func MigrateBep3(oldGenState v0_9bep3.GenesisState) v0_11bep3.GenesisState {
	var assetParams v0_11bep3.AssetParams
	var assetSupplies v0_11bep3.AssetSupplies
	v0_9Params := oldGenState.Params

	for _, asset := range v0_9Params.SupportedAssets {
		v10AssetParam := v0_11bep3.AssetParam{
			Active:        asset.Active,
			Denom:         asset.Denom,
			CoinID:        asset.CoinID,
			DeputyAddress: v0_9Params.BnbDeputyAddress,
			FixedFee:      v0_9Params.BnbDeputyFixedFee,
			MinSwapAmount: sdk.OneInt(), // set min swap to one - prevents accounts that hold zero bnb from creating spam txs
			MaxSwapAmount: v0_9Params.MaxAmount,
			MinBlockLock:  v0_9Params.MinBlockLock,
			MaxBlockLock:  v0_9Params.MaxBlockLock,
			SupplyLimit: v0_11bep3.SupplyLimit{
				Limit:          asset.Limit,
				TimeLimited:    false,
				TimePeriod:     time.Duration(0),
				TimeBasedLimit: sdk.ZeroInt(),
			},
		}
		assetParams = append(assetParams, v10AssetParam)
	}
	for _, supply := range oldGenState.AssetSupplies {
		newSupply := v0_11bep3.NewAssetSupply(supply.IncomingSupply, supply.OutgoingSupply, supply.CurrentSupply, sdk.NewCoin(supply.CurrentSupply.Denom, sdk.ZeroInt()), time.Duration(0))
		assetSupplies = append(assetSupplies, newSupply)
	}
	var swaps v0_11bep3.AtomicSwaps
	for _, oldSwap := range oldGenState.AtomicSwaps {
		newSwap := v0_11bep3.AtomicSwap{
			Amount:              oldSwap.Amount,
			RandomNumberHash:    oldSwap.RandomNumberHash,
			ExpireHeight:        oldSwap.ExpireHeight,
			Timestamp:           oldSwap.Timestamp,
			Sender:              oldSwap.Sender,
			Recipient:           oldSwap.Recipient,
			SenderOtherChain:    oldSwap.SenderOtherChain,
			RecipientOtherChain: oldSwap.RecipientOtherChain,
			ClosedBlock:         oldSwap.ClosedBlock,
			Status:              v0_11bep3.SwapStatus(oldSwap.Status),
			CrossChain:          oldSwap.CrossChain,
			Direction:           v0_11bep3.SwapDirection(oldSwap.Direction),
		}
		swaps = append(swaps, newSwap)
	}
	return v0_11bep3.GenesisState{
		Params: v0_11bep3.Params{
			AssetParams: assetParams},
		AtomicSwaps:       swaps,
		Supplies:          assetSupplies,
		PreviousBlockTime: v0_11bep3.DefaultPreviousBlockTime,
	}
}

// MigratePricefeed migrates from a v0.9 (or v0.10) pricefeed genesis state to a v0.11 pricefeed genesis state
func MigratePricefeed(oldGenState v0_9pricefeed.GenesisState) v0_11pricefeed.GenesisState {
	var newMarkets v0_11pricefeed.Markets
	var newPostedPrices v0_11pricefeed.PostedPrices
	var oracles []sdk.AccAddress

	for _, market := range oldGenState.Params.Markets {
		newMarket := v0_11pricefeed.NewMarket(market.MarketID, market.BaseAsset, market.QuoteAsset, market.Oracles, market.Active)
		newMarkets = append(newMarkets, newMarket)
		oracles = market.Oracles
	}
	// ------- add btc, xrp, busd markets --------
	btcSpotMarket := v0_11pricefeed.NewMarket("btc:usd", "btc", "usd", oracles, true)
	btcLiquidationMarket := v0_11pricefeed.NewMarket("btc:usd:30", "btc", "usd", oracles, true)
	xrpSpotMarket := v0_11pricefeed.NewMarket("xrp:usd", "xrp", "usd", oracles, true)
	xrpLiquidationMarket := v0_11pricefeed.NewMarket("xrp:usd:30", "xrp", "usd", oracles, true)
	busdSpotMarket := v0_11pricefeed.NewMarket("busd:usd", "busd", "usd", oracles, true)
	busdLiquidationMarket := v0_11pricefeed.NewMarket("busd:usd:30", "busd", "usd", oracles, true)
	newMarkets = append(newMarkets, btcSpotMarket, btcLiquidationMarket, xrpSpotMarket, xrpLiquidationMarket, busdSpotMarket, busdLiquidationMarket)

	for _, price := range oldGenState.PostedPrices {
		newPrice := v0_11pricefeed.NewPostedPrice(price.MarketID, price.OracleAddress, price.Price, price.Expiry)
		newPostedPrices = append(newPostedPrices, newPrice)
	}
	newParams := v0_11pricefeed.NewParams(newMarkets)

	return v0_11pricefeed.NewGenesisState(newParams, newPostedPrices)
}
