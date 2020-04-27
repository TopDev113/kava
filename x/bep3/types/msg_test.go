package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/bep3/types"
)

var (
	coinsSingle  = sdk.NewCoins(sdk.NewInt64Coin("bnb", int64(50000)))
	coinsZero    = sdk.Coins{sdk.Coin{}}
	binanceAddrs = []sdk.AccAddress{
		sdk.AccAddress(crypto.AddressHash([]byte("BinanceTest1"))),
		sdk.AccAddress(crypto.AddressHash([]byte("BinanceTest2"))),
	}
	kavaAddrs = []sdk.AccAddress{
		sdk.AccAddress(crypto.AddressHash([]byte("KavaTest1"))),
		sdk.AccAddress(crypto.AddressHash([]byte("KavaTest2"))),
	}
	randomNumberBytes = []byte{15}
	timestampInt64    = int64(100)
	randomNumberHash  = types.CalculateRandomHash(randomNumberBytes, timestampInt64)
)

func TestMsgCreateAtomicSwap(t *testing.T) {
	tests := []struct {
		description         string
		from                sdk.AccAddress
		to                  sdk.AccAddress
		recipientOtherChain string
		senderOtherChain    string
		randomNumberHash    tmbytes.HexBytes
		timestamp           int64
		amount              sdk.Coins
		expectedIncome      string
		heightSpan          int64
		crossChain          bool
		expectPass          bool
	}{
		{"normal", binanceAddrs[0], kavaAddrs[0], "", "", randomNumberHash, timestampInt64, coinsSingle, "50000bnb", 500, false, true},
		{"cross-chain", binanceAddrs[0], kavaAddrs[0], kavaAddrs[0].String(), binanceAddrs[0].String(), randomNumberHash, timestampInt64, coinsSingle, "50000bnb", 80000, true, true},
		{"with other chain fields", binanceAddrs[0], kavaAddrs[0], kavaAddrs[0].String(), binanceAddrs[0].String(), randomNumberHash, timestampInt64, coinsSingle, "50000bnb", 500, false, false},
		{"cross-cross no other chain fields", binanceAddrs[0], kavaAddrs[0], "", "", randomNumberHash, timestampInt64, coinsSingle, "50000bnb", 500, true, false},
		{"zero coins", binanceAddrs[0], kavaAddrs[0], "", "", randomNumberHash, timestampInt64, coinsZero, "50000bnb", 500, true, false},
	}

	for i, tc := range tests {
		msg := types.NewMsgCreateAtomicSwap(
			tc.from,
			tc.to,
			tc.recipientOtherChain,
			tc.senderOtherChain,
			tc.randomNumberHash,
			tc.timestamp,
			tc.amount,
			tc.expectedIncome,
			tc.heightSpan,
			tc.crossChain,
		)
		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgClaimAtomicSwap(t *testing.T) {
	swapID := types.CalculateSwapID(randomNumberHash, binanceAddrs[0], "")

	tests := []struct {
		description  string
		from         sdk.AccAddress
		swapID       tmbytes.HexBytes
		randomNumber tmbytes.HexBytes
		expectPass   bool
	}{
		{"normal", binanceAddrs[0], swapID, randomNumberHash, true},
	}

	for i, tc := range tests {
		msg := types.NewMsgClaimAtomicSwap(
			tc.from,
			tc.swapID,
			tc.randomNumber,
		)
		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgRefundAtomicSwap(t *testing.T) {
	swapID := types.CalculateSwapID(randomNumberHash, binanceAddrs[0], "")

	tests := []struct {
		description string
		from        sdk.AccAddress
		swapID      tmbytes.HexBytes
		expectPass  bool
	}{
		{"normal", binanceAddrs[0], swapID, true},
	}

	for i, tc := range tests {
		msg := types.NewMsgRefundAtomicSwap(
			tc.from,
			tc.swapID,
		)
		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}
