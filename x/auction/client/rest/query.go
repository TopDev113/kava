package rest

import (
	"fmt"
	"net/http"

	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/gorilla/mux"

	"github.com/kava-labs/kava/x/auction/types"
)

const restAuctionID = "auction-id"

// define routes that get registered by the main application
func registerQueryRoutes(cliCtx client.Context, r *mux.Router) {
	r.HandleFunc(fmt.Sprintf("/%s/parameters", types.ModuleName), queryParamsHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc(fmt.Sprintf("/%s/auctions/{%s}", types.ModuleName, restAuctionID), queryAuctionHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc(fmt.Sprintf("/%s/auctions", types.ModuleName), queryAuctionsHandlerFn(cliCtx)).Methods("GET")
}

func queryParamsHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.ModuleName, types.QueryGetParams), nil)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

func queryAuctionHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse the query height
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// Prepare params for querier
		vars := mux.Vars(r)
		if len(vars[restAuctionID]) == 0 {
			err := fmt.Errorf("%s required but not specified", restAuctionID)
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		auctionID, ok := rest.ParseUint64OrReturnBadRequest(w, vars[restAuctionID])
		if !ok {
			return
		}

		bz, err := cliCtx.LegacyAmino.MarshalJSON(types.NewQueryAuctionParams(auctionID))
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// Query
		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.ModuleName, types.QueryGetAuction), bz)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Write response
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

func queryAuctionsHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, page, limit, err := rest.ParseHTTPArgsWithLimit(r, 0)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// Parse the query height
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		var auctionType string
		var auctionOwner sdk.AccAddress
		var auctionDenom string
		var auctionPhase string

		if x := r.URL.Query().Get(RestType); len(x) != 0 {
			auctionType = strings.ToLower(strings.TrimSpace(x))
			if auctionType != types.CollateralAuctionType &&
				auctionType != types.SurplusAuctionType &&
				auctionType != types.DebtAuctionType {
				rest.WriteErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid auction type %s", x))
				return
			}
		}

		if x := r.URL.Query().Get(RestOwner); len(x) != 0 {
			if auctionType != types.CollateralAuctionType {
				rest.WriteErrorResponse(w, http.StatusBadRequest, "cannot apply owner flag to non-collateral auction type")
			}
			auctionOwnerStr := strings.ToLower(strings.TrimSpace(x))
			auctionOwner, err = sdk.AccAddressFromBech32(auctionOwnerStr)
			if err != nil {
				rest.WriteErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("cannot parse address from auction owner %s", auctionOwnerStr))
			}
		}

		if x := r.URL.Query().Get(RestDenom); len(x) != 0 {
			auctionDenom = strings.TrimSpace(x)
			err := sdk.ValidateDenom(auctionDenom)
			if err != nil {
				rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		if x := r.URL.Query().Get(RestPhase); len(x) != 0 {
			auctionPhase = strings.ToLower(strings.TrimSpace(x))
			if auctionType != types.CollateralAuctionType && len(auctionType) > 0 {
				rest.WriteErrorResponse(w, http.StatusBadRequest, "cannot apply phase flag to non-collateral auction type")
				return
			}
			if auctionPhase != types.ForwardAuctionPhase && auctionPhase != types.ReverseAuctionPhase {
				rest.WriteErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid auction phase %s", x))
				return
			}
		}

		params := types.NewQueryAllAuctionParams(page, limit, auctionType, auctionDenom, auctionPhase, auctionOwner)
		bz, err := cliCtx.LegacyAmino.MarshalJSON(params)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// Query
		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.ModuleName, types.QueryGetAuctions), bz)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Write response
		cliCtx = cliCtx.WithHeight(height)

		// Unmarshal to Auction and remarshal as AuctionWithPhase
		var auctions []types.Auction
		err = cliCtx.LegacyAmino.UnmarshalJSON(res, &auctions)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}

		auctionsWithPhase := []types.AuctionWithPhase{} // using empty slice so json returns [] instead of null when there are no auctions
		for _, a := range auctions {
			auctionsWithPhase = append(auctionsWithPhase, types.NewAuctionWithPhase(a))
		}
		rest.PostProcessResponse(w, cliCtx, cliCtx.LegacyAmino.MustMarshalJSON(auctionsWithPhase))
	}
}
