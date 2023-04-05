package bep3_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/bep3/types"
)

const (
	TestSenderOtherChain    = "bnb1uky3me9ggqypmrsvxk7ur6hqkzq7zmv4ed4ng7"
	TestRecipientOtherChain = "bnb1urfermcg92dwq36572cx4xg84wpk3lfpksr5g7"
	TestDeputy              = "kava1xy7hrjy9r0algz9w3gzm8u6mrpq97kwta747gj"
	TestUser                = "kava1vry5lhegzlulehuutcr7nmdlmktw88awp0a39p"
)

var (
	StandardSupplyLimit = i(100000000000)
	DenomMap            = map[int]string{0: "bnb", 1: "inc"}
)

func i(in int64) sdkmath.Int                { return sdkmath.NewInt(in) }
func d(de int64) sdk.Dec                    { return sdk.NewDec(de) }
func c(denom string, amount int64) sdk.Coin { return sdk.NewInt64Coin(denom, amount) }
func cs(coins ...sdk.Coin) sdk.Coins        { return sdk.NewCoins(coins...) }
func ts(minOffset int) int64                { return tmtime.Now().Add(time.Duration(minOffset) * time.Minute).Unix() }

func NewBep3GenStateMulti(cdc codec.JSONCodec, deputy sdk.AccAddress) app.GenesisState {
	bep3Genesis := baseGenState(deputy)
	return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&bep3Genesis)}
}

func baseGenState(deputy sdk.AccAddress) types.GenesisState {
	bep3Genesis := types.GenesisState{
		Params: types.Params{
			AssetParams: types.AssetParams{
				{
					Denom:  "bnb",
					CoinID: 714,
					SupplyLimit: types.SupplyLimit{
						Limit:          sdkmath.NewInt(350000000000000),
						TimeLimited:    false,
						TimeBasedLimit: sdk.ZeroInt(),
						TimePeriod:     time.Hour,
					},
					Active:        true,
					DeputyAddress: deputy,
					FixedFee:      sdkmath.NewInt(1000),
					MinSwapAmount: sdk.OneInt(),
					MaxSwapAmount: sdkmath.NewInt(1000000000000),
					MinBlockLock:  types.DefaultMinBlockLock,
					MaxBlockLock:  types.DefaultMaxBlockLock,
				},
				{
					Denom:  "inc",
					CoinID: 9999,
					SupplyLimit: types.SupplyLimit{
						Limit:          sdkmath.NewInt(100000000000),
						TimeLimited:    false,
						TimeBasedLimit: sdk.ZeroInt(),
						TimePeriod:     time.Hour,
					},
					Active:        true,
					DeputyAddress: deputy,
					FixedFee:      sdkmath.NewInt(1000),
					MinSwapAmount: sdk.OneInt(),
					MaxSwapAmount: sdkmath.NewInt(1000000000000),
					MinBlockLock:  types.DefaultMinBlockLock,
					MaxBlockLock:  types.DefaultMaxBlockLock,
				},
			},
		},
		Supplies: types.AssetSupplies{
			types.NewAssetSupply(
				sdk.NewCoin("bnb", sdk.ZeroInt()),
				sdk.NewCoin("bnb", sdk.ZeroInt()),
				sdk.NewCoin("bnb", sdk.ZeroInt()),
				sdk.NewCoin("bnb", sdk.ZeroInt()),
				time.Duration(0),
			),
			types.NewAssetSupply(
				sdk.NewCoin("inc", sdk.ZeroInt()),
				sdk.NewCoin("inc", sdk.ZeroInt()),
				sdk.NewCoin("inc", sdk.ZeroInt()),
				sdk.NewCoin("inc", sdk.ZeroInt()),
				time.Duration(0),
			),
		},
		PreviousBlockTime: types.DefaultPreviousBlockTime,
	}
	return bep3Genesis
}

func loadSwapAndSupply(addr sdk.AccAddress, index int) (types.AtomicSwap, types.AssetSupply) {
	coin := c(DenomMap[index], 50000)
	expireOffset := types.DefaultMinBlockLock // Default expire height + offet to match timestamp
	timestamp := ts(index)                    // One minute apart
	randomNumber, _ := types.GenerateSecureRandomNumber()
	randomNumberHash := types.CalculateRandomHash(randomNumber[:], timestamp)
	swap := types.NewAtomicSwap(cs(coin), randomNumberHash,
		expireOffset, timestamp, addr, addr, TestSenderOtherChain,
		TestRecipientOtherChain, 1, types.SWAP_STATUS_OPEN, true, types.SWAP_DIRECTION_INCOMING)

	supply := types.NewAssetSupply(coin, c(coin.Denom, 0),
		c(coin.Denom, 0), c(coin.Denom, 0), time.Duration(0))

	return swap, supply
}
