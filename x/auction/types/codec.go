package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
)

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the
// governance module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgPlaceBid{}, "auction/MsgPlaceBid", nil)

	cdc.RegisterInterface((*GenesisAuction)(nil), nil)
	cdc.RegisterInterface((*Auction)(nil), nil)
	cdc.RegisterConcrete(&SurplusAuction{}, "auction/SurplusAuction", nil)
	cdc.RegisterConcrete(&DebtAuction{}, "auction/DebtAuction", nil)
	cdc.RegisterConcrete(&CollateralAuction{}, "auction/CollateralAuction", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgPlaceBid{},
	)

	registry.RegisterInterface(
		"kava.auction.v1beta1.Auction",
		(*Auction)(nil),
		&SurplusAuction{},
		&DebtAuction{},
		&CollateralAuction{},
	)

	registry.RegisterInterface(
		"kava.auction.v1beta1.GenesisAuction",
		(*GenesisAuction)(nil),
		&SurplusAuction{},
		&DebtAuction{},
		&CollateralAuction{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc is an amino codec instance with this module's types registered.
	//
	// Deprecated: The codec used for serialization should be provided to this module and
	// defined at the application level.
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)

	// Register all Amino interfaces and concrete types on the authz Amino codec so that this can later be
	// used to properly serialize MsgGrant and MsgExec instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
}
