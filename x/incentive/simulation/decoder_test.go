package simulation

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/tendermint/libs/kv"

	"github.com/kava-labs/kava/x/incentive/types"
)

func makeTestCodec() (cdc *codec.Codec) {
	cdc = codec.New()
	sdk.RegisterCodec(cdc)
	types.RegisterCodec(cdc)
	return
}

func TestDecodeDistributionStore(t *testing.T) {
	cdc := makeTestCodec()
	addr, _ := sdk.AccAddressFromBech32("kava15qdefkmwswysgg4qxgqpqr35k3m49pkx2jdfnw")
	claim := types.NewUSDXMintingClaim(addr, sdk.NewCoin("ukava", sdk.NewInt(1000000)), types.RewardIndexes{types.NewRewardIndex("bnb-a", sdk.ZeroDec())})
	prevBlockTime := time.Now().Add(time.Hour * -1).UTC()
	factor := sdk.ZeroDec()

	kvPairs := kv.Pairs{
		kv.Pair{Key: types.ClaimKeyPrefix, Value: cdc.MustMarshalBinaryBare(claim)},
		kv.Pair{Key: []byte(types.BlockTimeKey), Value: cdc.MustMarshalBinaryBare(prevBlockTime)},
		kv.Pair{Key: []byte(types.RewardFactorKey), Value: cdc.MustMarshalBinaryBare(factor)},
		kv.Pair{Key: []byte{0x99}, Value: []byte{0x99}},
	}

	tests := []struct {
		name        string
		expectedLog string
	}{
		{"Claim", fmt.Sprintf("%v\n%v", claim, claim)},
		{"PreviousBlockTime", fmt.Sprintf("%v\n%v", prevBlockTime, prevBlockTime)},
		{"RewardFactor", fmt.Sprintf("%v\n%v", factor, factor)},
		{"other", ""},
	}
	for i, tt := range tests {
		i, tt := i, tt
		t.Run(tt.name, func(t *testing.T) {
			switch i {
			case len(tests) - 1:
				require.Panics(t, func() { DecodeStore(cdc, kvPairs[i], kvPairs[i]) }, tt.name)
			default:
				require.Equal(t, tt.expectedLog, DecodeStore(cdc, kvPairs[i], kvPairs[i]), tt.name)
			}
		})
	}
}
