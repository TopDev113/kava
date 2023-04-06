package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the
// earn module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDeposit{}, "earn/MsgDeposit", nil)
	cdc.RegisterConcrete(&MsgWithdraw{}, "earn/MsgWithdraw", nil)
	cdc.RegisterConcrete(&CommunityPoolDepositProposal{}, "kava/CommunityPoolDepositProposal", nil)
	cdc.RegisterConcrete(&CommunityPoolWithdrawProposal{}, "kava/CommunityPoolWithdrawProposal", nil)
}

// RegisterInterfaces registers proto messages under their interfaces for unmarshalling,
// in addition to registerting the msg service for handling tx msgs
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDeposit{},
		&MsgWithdraw{},
	)
	registry.RegisterImplementations((*govv1beta1.Content)(nil),
		&CommunityPoolDepositProposal{},
		&CommunityPoolWithdrawProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino = codec.NewLegacyAmino()
	// ModuleCdc represents the legacy amino codec for the module
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)

	// Register all Amino interfaces and concrete types on the authz Amino codec so that this can later be
	// used to properly serialize MsgGrant and MsgExec instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
}
