package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/types/rest"

	"github.com/kava-labs/kava/x/swap/types"
)

func registerTxRoutes(cliCtx client.Context, r *mux.Router) {
	r.HandleFunc(fmt.Sprintf("/%s/deposit", types.ModuleName), postDepositHandlerFn(cliCtx)).Methods("POST")
	r.HandleFunc(fmt.Sprintf("/%s/withdraw", types.ModuleName), postWithdrawHandlerFn(cliCtx)).Methods("POST")
	r.HandleFunc(fmt.Sprintf("/%s/swapExactForTokens", types.ModuleName), postSwapExactForTokensHandlerFn(cliCtx)).Methods("POST")
	r.HandleFunc(fmt.Sprintf("/%s/swapForExactTokens", types.ModuleName), postSwapForExactTokensHandlerFn(cliCtx)).Methods("POST")
}

func postDepositHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode POST request body
		var req PostCreateDepositReq
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}
		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		msg := types.NewMsgDeposit(req.From.String(), req.TokenA, req.TokenB, req.Slippage, req.Deadline)
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

func postWithdrawHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode POST request body
		var req PostCreateWithdrawReq
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}
		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		msg := types.NewMsgWithdraw(req.From.String(), req.Shares, req.MinTokenA, req.MinTokenA, req.Deadline)
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

func postSwapExactForTokensHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode POST request body
		var req PostCreateSwapExactForTokensReq
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}
		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		msg := types.NewMsgSwapExactForTokens(req.Requester.String(), req.ExactTokenA, req.TokenB, req.Slippage, req.Deadline)
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

func postSwapForExactTokensHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode POST request body
		var req PostCreateSwapForExactTokensReq
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}
		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		msg := types.NewMsgSwapForExactTokens(req.Requester.String(), req.TokenA, req.ExactTokenB, req.Slippage, req.Deadline)
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}
