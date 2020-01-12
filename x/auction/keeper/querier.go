package keeper

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/x/auction/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// NewQuerier is the module level router for state queries
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case types.QueryGetAuction:
			return queryAuctions(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest("unknown auction query endpoint")
		}
	}
}

func queryAuctions(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) (res []byte, err sdk.Error) {
	var auctionsList types.QueryResAuctions

	keeper.IterateAuctions(ctx, func(a types.Auction) bool {
		auctionsList = append(auctionsList, fmt.Sprintf("%+v", a)) // TODO formatting
		return false
	})

	bz, err2 := codec.MarshalJSONIndent(keeper.cdc, auctionsList)
	if err2 != nil {
		panic("could not marshal result to JSON")
	}

	return bz, nil
}
