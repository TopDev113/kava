package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/cdp"
	"github.com/kava-labs/kava/x/pricefeed"
	tmtime "github.com/tendermint/tendermint/types/time"
)

// Avoid cluttering test cases with long function names
func i(in int64) sdk.Int                    { return sdk.NewInt(in) }
func d(str string) sdk.Dec                  { return sdk.MustNewDecFromStr(str) }
func c(denom string, amount int64) sdk.Coin { return sdk.NewInt64Coin(denom, amount) }
func cs(coins ...sdk.Coin) sdk.Coins        { return sdk.NewCoins(coins...) }

func NewPricefeedGenState(asset string, price sdk.Dec) app.GenesisState {
	pfGenesis := pricefeed.GenesisState{
		Params: pricefeed.Params{
			Markets: []pricefeed.Market{
				pricefeed.Market{MarketID: asset + ":usd", BaseAsset: asset, QuoteAsset: "usd", Oracles: pricefeed.Oracles{}, Active: true},
			},
		},
		PostedPrices: []pricefeed.PostedPrice{
			pricefeed.PostedPrice{
				MarketID:      asset + ":usd",
				OracleAddress: sdk.AccAddress{},
				Price:         price,
				Expiry:        time.Now().Add(1 * time.Hour),
			},
		},
	}
	return app.GenesisState{pricefeed.ModuleName: pricefeed.ModuleCdc.MustMarshalJSON(pfGenesis)}
}

func NewCDPGenState(asset string, liquidationRatio sdk.Dec) app.GenesisState {
	cdpGenesis := cdp.GenesisState{
		Params: cdp.Params{
			GlobalDebtLimit: sdk.NewCoins(sdk.NewInt64Coin("usdx", 1000000000000)),
			CollateralParams: cdp.CollateralParams{
				{
					Denom:            asset,
					LiquidationRatio: liquidationRatio,
					DebtLimit:        sdk.NewCoins(sdk.NewInt64Coin("usdx", 1000000000000)),
					StabilityFee:     sdk.MustNewDecFromStr("1.000000001547125958"), // %5 apr
					Prefix:           0x20,
					ConversionFactor: i(6),
					MarketID:         asset + ":usd",
				},
			},
			DebtParams: cdp.DebtParams{
				{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					DebtLimit:        sdk.NewCoins(sdk.NewInt64Coin("usdx", 1000000000000)),
					ConversionFactor: i(6),
					DebtFloor:        i(10000000),
				},
			},
		},
		StartingCdpID:     cdp.DefaultCdpStartingID,
		DebtDenom:         cdp.DefaultDebtDenom,
		CDPs:              cdp.CDPs{},
		PreviousBlockTime: cdp.DefaultPreviousBlockTime,
	}
	return app.GenesisState{cdp.ModuleName: cdp.ModuleCdc.MustMarshalJSON(cdpGenesis)}
}

func NewPricefeedGenStateMulti() app.GenesisState {
	pfGenesis := pricefeed.GenesisState{
		Params: pricefeed.Params{
			Markets: []pricefeed.Market{
				pricefeed.Market{MarketID: "btc:usd", BaseAsset: "btc", QuoteAsset: "usd", Oracles: pricefeed.Oracles{}, Active: true},
				pricefeed.Market{MarketID: "xrp:usd", BaseAsset: "xrp", QuoteAsset: "usd", Oracles: pricefeed.Oracles{}, Active: true},
			},
		},
		PostedPrices: []pricefeed.PostedPrice{
			pricefeed.PostedPrice{
				MarketID:      "btc:usd",
				OracleAddress: sdk.AccAddress{},
				Price:         sdk.MustNewDecFromStr("8000.00"),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
			pricefeed.PostedPrice{
				MarketID:      "xrp:usd",
				OracleAddress: sdk.AccAddress{},
				Price:         sdk.MustNewDecFromStr("0.25"),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
		},
	}
	return app.GenesisState{pricefeed.ModuleName: pricefeed.ModuleCdc.MustMarshalJSON(pfGenesis)}
}
func NewCDPGenStateMulti() app.GenesisState {
	cdpGenesis := cdp.GenesisState{
		Params: cdp.Params{
			GlobalDebtLimit: sdk.NewCoins(sdk.NewInt64Coin("usdx", 1000000000000), sdk.NewInt64Coin("susd", 1000000000000)),
			CollateralParams: cdp.CollateralParams{
				{
					Denom:            "xrp",
					LiquidationRatio: sdk.MustNewDecFromStr("2.0"),
					DebtLimit:        sdk.NewCoins(sdk.NewInt64Coin("usdx", 500000000000), sdk.NewInt64Coin("susd", 500000000000)),
					StabilityFee:     sdk.MustNewDecFromStr("1.000000001547125958"), // %5 apr
					Prefix:           0x20,
					MarketID:         "xrp:usd",
					ConversionFactor: i(6),
				},
				{
					Denom:            "btc",
					LiquidationRatio: sdk.MustNewDecFromStr("1.5"),
					DebtLimit:        sdk.NewCoins(sdk.NewInt64Coin("usdx", 500000000000), sdk.NewInt64Coin("susd", 500000000000)),
					StabilityFee:     sdk.MustNewDecFromStr("1.000000000782997609"), // %2.5 apr
					Prefix:           0x21,
					MarketID:         "btc:usd",
					ConversionFactor: i(8),
				},
			},
			DebtParams: cdp.DebtParams{
				{
					Denom:            "usdx",
					ReferenceAsset:   "usd",
					DebtLimit:        sdk.NewCoins(sdk.NewInt64Coin("usdx", 1000000000000)),
					ConversionFactor: i(6),
					DebtFloor:        i(10000000),
				},
				{
					Denom:            "susd",
					ReferenceAsset:   "usd",
					DebtLimit:        sdk.NewCoins(sdk.NewInt64Coin("susd", 1000000000000)),
					ConversionFactor: i(6),
					DebtFloor:        i(10000000),
				},
			},
		},
		StartingCdpID:     cdp.DefaultCdpStartingID,
		DebtDenom:         cdp.DefaultDebtDenom,
		CDPs:              cdp.CDPs{},
		PreviousBlockTime: cdp.DefaultPreviousBlockTime,
	}
	return app.GenesisState{cdp.ModuleName: cdp.ModuleCdc.MustMarshalJSON(cdpGenesis)}
}

func cdps() (cdps cdp.CDPs) {
	_, addrs := app.GeneratePrivKeyAddressPairs(3)
	c1 := cdp.NewCDP(uint64(1), addrs[0], sdk.NewCoins(sdk.NewCoin("xrp", sdk.NewInt(10000000))), sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(8000000))), tmtime.Canonical(time.Now()))
	c2 := cdp.NewCDP(uint64(2), addrs[1], sdk.NewCoins(sdk.NewCoin("xrp", sdk.NewInt(100000000))), sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(10000000))), tmtime.Canonical(time.Now()))
	c3 := cdp.NewCDP(uint64(3), addrs[1], sdk.NewCoins(sdk.NewCoin("btc", sdk.NewInt(1000000000))), sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(10000000))), tmtime.Canonical(time.Now()))
	c4 := cdp.NewCDP(uint64(4), addrs[2], sdk.NewCoins(sdk.NewCoin("xrp", sdk.NewInt(1000000000))), sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(500000000))), tmtime.Canonical(time.Now()))
	cdps = append(cdps, c1, c2, c3, c4)
	return
}
