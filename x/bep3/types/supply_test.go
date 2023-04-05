package types

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestAssetSupplyValidate(t *testing.T) {
	coin := sdk.NewCoin("kava", sdk.OneInt())
	invalidCoin := sdk.Coin{Denom: "Invalid Denom", Amount: sdkmath.NewInt(-1)}
	testCases := []struct {
		msg     string
		asset   AssetSupply
		expPass bool
	}{
		{
			msg:     "valid asset",
			asset:   NewAssetSupply(coin, coin, coin, coin, time.Duration(0)),
			expPass: true,
		},
		{
			"invalid incoming supply",
			AssetSupply{IncomingSupply: invalidCoin},
			false,
		},
		{
			"invalid outgoing supply",
			AssetSupply{
				IncomingSupply: coin,
				OutgoingSupply: invalidCoin,
			},
			false,
		},
		{
			"invalid current supply",
			AssetSupply{
				IncomingSupply: coin,
				OutgoingSupply: coin,
				CurrentSupply:  invalidCoin,
			},
			false,
		},
		{
			"invalid time limitedcurrent supply",
			AssetSupply{
				IncomingSupply:           coin,
				OutgoingSupply:           coin,
				CurrentSupply:            coin,
				TimeLimitedCurrentSupply: invalidCoin,
			},
			false,
		},
		{
			"non matching denoms",
			AssetSupply{
				IncomingSupply:           coin,
				OutgoingSupply:           coin,
				CurrentSupply:            coin,
				TimeLimitedCurrentSupply: sdk.NewCoin("lol", sdk.ZeroInt()),
				TimeElapsed:              time.Hour,
			},
			false,
		},
	}

	for _, tc := range testCases {
		err := tc.asset.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

func TestAssetSupplyEquality(t *testing.T) {
	coin := sdk.NewCoin("test", sdk.OneInt())
	coin2 := sdk.NewCoin("other", sdk.OneInt())
	testCases := []struct {
		name    string
		asset1  AssetSupply
		asset2  AssetSupply
		expPass bool
	}{
		{
			name:    "equal",
			asset1:  NewAssetSupply(coin, coin, coin, coin, time.Duration(0)),
			asset2:  NewAssetSupply(coin, coin, coin, coin, time.Duration(0)),
			expPass: true,
		},
		{
			name:    "not equal duration",
			asset1:  NewAssetSupply(coin, coin, coin, coin, time.Duration(0)),
			asset2:  NewAssetSupply(coin, coin, coin, coin, time.Duration(1)),
			expPass: false,
		},
		{
			name:    "not equal coin amount",
			asset1:  NewAssetSupply(coin, coin, coin, coin, time.Duration(0)),
			asset2:  NewAssetSupply(sdk.NewCoin("test", sdk.ZeroInt()), coin, coin, coin, time.Duration(1)),
			expPass: false,
		},
		{
			name:    "not equal coin denom",
			asset1:  NewAssetSupply(coin, coin, coin, coin, time.Duration(0)),
			asset2:  NewAssetSupply(coin2, coin2, coin2, coin2, time.Duration(1)),
			expPass: false,
		},
	}

	for _, tc := range testCases {
		if tc.expPass {
			require.True(t, tc.asset1.Equal(tc.asset2), tc.name)
		} else {
			require.False(t, tc.asset1.Equal(tc.asset2), tc.name)
		}
	}
}
