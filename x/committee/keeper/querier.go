package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/kava-labs/kava/x/committee/types"
)

// NewQuerier creates a new gov Querier instance
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, error) {
		switch path[0] {

		case types.QueryCommittees:
			return queryCommittees(ctx, path[1:], req, keeper)
		case types.QueryCommittee:
			return queryCommittee(ctx, path[1:], req, keeper)
		case types.QueryProposals:
			return queryProposals(ctx, path[1:], req, keeper)
		case types.QueryProposal:
			return queryProposal(ctx, path[1:], req, keeper)
		case types.QueryVotes:
			return queryVotes(ctx, path[1:], req, keeper)
		case types.QueryVote:
			return queryVote(ctx, path[1:], req, keeper)
		case types.QueryTally:
			return queryTally(ctx, path[1:], req, keeper)
		case types.QueryNextProposalID:
			return queryNextProposalID(ctx, req, keeper)
		case types.QueryRawParams:
			return queryRawParams(ctx, path[1:], req, keeper)

		default:
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown %s query endpoint", types.ModuleName)
		}
	}
}

// ------------------------------------------
//				Committees
// ------------------------------------------

func queryCommittees(ctx sdk.Context, path []string, _ abci.RequestQuery, keeper Keeper) ([]byte, error) {

	committees := keeper.GetCommittees(ctx)

	bz, err := codec.MarshalJSONIndent(keeper.cdc, committees)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryCommittee(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var params types.QueryCommitteeParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	committee, found := keeper.GetCommittee(ctx, params.CommitteeID)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrUnknownCommittee, "%d", params.CommitteeID)
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, committee)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

// ------------------------------------------
//				Proposals
// ------------------------------------------

func queryProposals(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var params types.QueryCommitteeParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	proposals := keeper.GetProposalsByCommittee(ctx, params.CommitteeID)

	bz, err := codec.MarshalJSONIndent(keeper.cdc, proposals)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryProposal(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var params types.QueryProposalParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	proposal, found := keeper.GetProposal(ctx, params.ProposalID)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrUnknownProposal, "%d", params.ProposalID)
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, proposal)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryNextProposalID(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	nextProposalID, _ := keeper.GetNextProposalID(ctx)

	bz, err := types.ModuleCdc.MarshalJSON(nextProposalID)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	return bz, nil
}

// ------------------------------------------
//				Votes
// ------------------------------------------

func queryVotes(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var params types.QueryProposalParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)

	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	votes := keeper.GetVotesByProposal(ctx, params.ProposalID)

	bz, err := codec.MarshalJSONIndent(keeper.cdc, votes)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryVote(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var params types.QueryVoteParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	vote, found := keeper.GetVote(ctx, params.ProposalID, params.Voter)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrUnknownVote, "proposal id: %d, voter: %s", params.ProposalID, params.Voter)
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, vote)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

// ------------------------------------------
//				Tally
// ------------------------------------------

func queryTally(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var params types.QueryProposalParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	proposal, found := keeper.GetProposal(ctx, params.ProposalID)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrUnknownProposal, "%d", params.ProposalID)
	}

	committee, found := keeper.GetCommittee(ctx, proposal.CommitteeID)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrUnknownCommittee, "%d", proposal.CommitteeID)
	}

	var pollingStatus types.ProposalPollingStatus
	switch com := committee.(type) {
	case types.MemberCommittee:
		currVotes := keeper.TallyMemberCommitteeVotes(ctx, params.ProposalID)
		possibleVotes := sdk.NewDec(int64(len(com.Members)))
		memberPollingStatus := types.NewProposalPollingStatus(params.ProposalID, currVotes,
			currVotes, possibleVotes, com.VoteThreshold, sdk.Dec{Int: nil})
		pollingStatus = memberPollingStatus
	case types.TokenCommittee:
		yesVotes, _, currVotes, possibleVotes := keeper.TallyTokenCommitteeVotes(ctx, params.ProposalID, com.TallyDenom)
		tokenPollingStatus := types.NewProposalPollingStatus(params.ProposalID, yesVotes,
			currVotes, possibleVotes, com.VoteThreshold, com.Quorum)
		pollingStatus = tokenPollingStatus
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, pollingStatus)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

// ------------------------------------------
//				Raw Params
// ------------------------------------------

func queryRawParams(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var params types.QueryRawParamsParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	subspace, found := keeper.ParamKeeper.GetSubspace(params.Subspace)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrUnknownSubspace, "subspace: %s", params.Subspace)
	}
	rawParams := subspace.GetRaw(ctx, []byte(params.Key))

	// encode the raw params as json, which converts them to a base64 string
	bz, err := codec.MarshalJSONIndent(keeper.cdc, rawParams)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}
