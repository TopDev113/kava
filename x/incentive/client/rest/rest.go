package rest

import (
	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"

	"github.com/kava-labs/kava/x/incentive/types"
)

// REST variable names
// nolint
const (
	RestDenom = "denom"
)

// RegisterRoutes registers incentive-related REST handlers to a router
func RegisterRoutes(cliCtx client.Context, r *mux.Router) {
	registerQueryRoutes(cliCtx, r)
	registerTxRoutes(cliCtx, r)
}

// PostClaimReq defines the properties of claim transaction's request body.
type PostClaimReq struct {
	BaseReq       rest.BaseReq     `json:"base_req" yaml:"base_req"`
	Sender        sdk.AccAddress   `json:"sender" yaml:"sender"`
	DenomsToClaim types.Selections `json:"denoms_to_claim" yaml:"denoms_to_claim"`
}

// PostClaimReq defines the properties of claim transaction's request body.
type PostClaimVVestingReq struct {
	BaseReq       rest.BaseReq     `json:"base_req" yaml:"base_req"`
	Sender        sdk.AccAddress   `json:"sender" yaml:"sender"`
	Receiver      sdk.AccAddress   `json:"receiver" yaml:"receiver"`
	DenomsToClaim types.Selections `json:"denoms_to_claim" yaml:"denoms_to_claim"`
}
