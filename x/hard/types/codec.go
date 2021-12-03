package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDeposit{}, "hard/MsgDeposit", nil)
	cdc.RegisterConcrete(&MsgWithdraw{}, "hard/MsgWithdraw", nil)
	cdc.RegisterConcrete(&MsgBorrow{}, "hard/MsgBorrow", nil)
	cdc.RegisterConcrete(&MsgLiquidate{}, "hard/MsgLiquidate", nil)
	cdc.RegisterConcrete(&MsgRepay{}, "hard/MsgRepay", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDeposit{},
		&MsgWithdraw{},
		&MsgBorrow{},
		&MsgLiquidate{},
		&MsgRepay{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
}
