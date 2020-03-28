package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

var ModuleCdc = codec.New()

func init() {
	RegisterCodec(ModuleCdc)
}

// RegisterCodec registers concrete types on amino
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterInterface((*Swap)(nil), nil)

	cdc.RegisterConcrete(MsgCreateAtomicSwap{}, "bep3/MsgCreateAtomicSwap", nil)
	cdc.RegisterConcrete(MsgRefundAtomicSwap{}, "bep3/MsgRefundAtomicSwap", nil)
	cdc.RegisterConcrete(MsgClaimAtomicSwap{}, "bep3/MsgClaimAtomicSwap", nil)
}
