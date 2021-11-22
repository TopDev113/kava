package rest

import (
	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
)

// REST variable names
// nolint
const (
	RestOwner = "owner"
	RestDenom = "denom"
	RestName  = "name"
)

// RegisterRoutes registers hard-related REST handlers to a router
func RegisterRoutes(cliCtx client.Context, r *mux.Router) {
	registerQueryRoutes(cliCtx, r)
	registerTxRoutes(cliCtx, r)
}

// PostCreateDepositReq defines the properties of a deposit create request's body
type PostCreateDepositReq struct {
	BaseReq rest.BaseReq   `json:"base_req" yaml:"base_req"`
	From    sdk.AccAddress `json:"from" yaml:"from"`
	Amount  sdk.Coins      `json:"amount" yaml:"amount"`
}

// PostCreateWithdrawReq defines the properties of a deposit withdraw request's body
type PostCreateWithdrawReq struct {
	BaseReq rest.BaseReq   `json:"base_req" yaml:"base_req"`
	From    sdk.AccAddress `json:"from" yaml:"from"`
	Amount  sdk.Coins      `json:"amount" yaml:"amount"`
}

// PostBorrowReq defines the properties of a borrow request's body
type PostBorrowReq struct {
	BaseReq rest.BaseReq   `json:"base_req" yaml:"base_req"`
	From    sdk.AccAddress `json:"from" yaml:"from"`
	Amount  sdk.Coins      `json:"amount" yaml:"amount"`
}

// PostRepayReq defines the properties of a repay request's body
type PostRepayReq struct {
	BaseReq rest.BaseReq   `json:"base_req" yaml:"base_req"`
	From    sdk.AccAddress `json:"from" yaml:"from"`
	Owner   sdk.AccAddress `json:"owner" yaml:"owner"`
	Amount  sdk.Coins      `json:"amount" yaml:"amount"`
}

// PostLiquidateReq defines the properties of a liquidate request's body
type PostLiquidateReq struct {
	BaseReq  rest.BaseReq   `json:"base_req" yaml:"base_req"`
	From     sdk.AccAddress `json:"from" yaml:"from"`
	Borrower sdk.AccAddress `json:"borrower" yaml:"borrower"`
}
