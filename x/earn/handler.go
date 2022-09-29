package earn

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/kava-labs/kava/x/earn/keeper"
	"github.com/kava-labs/kava/x/earn/types"
)

// NewCommunityPoolProposalHandler
func NewCommunityPoolProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.CommunityPoolDepositProposal:
			return keeper.HandleCommunityPoolDepositProposal(ctx, k, c)
		case *types.CommunityPoolWithdrawProposal:
			return keeper.HandleCommunityPoolWithdrawProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized earn proposal content type: %T", c)
		}
	}
}
