package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/kava-labs/kava/x/committee/types"
)

// SubmitProposal adds a proposal to a committee so that it can be voted on.
func (k Keeper) SubmitProposal(ctx sdk.Context, proposer sdk.AccAddress, committeeID uint64, pubProposal types.PubProposal) (uint64, error) {
	// Limit proposals to only be submitted by committee members
	com, found := k.GetCommittee(ctx, committeeID)
	if !found {
		return 0, errorsmod.Wrapf(types.ErrUnknownCommittee, "%d", committeeID)
	}
	if !com.HasMember(proposer) {
		return 0, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "proposer not member of committee")
	}

	// Check committee has permissions to enact proposal.
	if !com.HasPermissionsFor(ctx, k.cdc, k.paramKeeper, pubProposal) {
		return 0, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "committee does not have permissions to enact proposal")
	}

	// Check proposal is valid
	if err := k.ValidatePubProposal(ctx, pubProposal); err != nil {
		return 0, err
	}

	// Get a new ID and store the proposal
	deadline := ctx.BlockTime().Add(com.GetProposalDuration())
	proposalID, err := k.StoreNewProposal(ctx, pubProposal, committeeID, deadline)
	if err != nil {
		return 0, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeProposalSubmit,
			sdk.NewAttribute(types.AttributeKeyCommitteeID, fmt.Sprintf("%d", com.GetID())),
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute(types.AttributeKeyDeadline, deadline.String()),
		),
	)
	return proposalID, nil
}

// AddVote submits a vote on a proposal.
func (k Keeper) AddVote(ctx sdk.Context, proposalID uint64, voter sdk.AccAddress, voteType types.VoteType) error {
	// Validate
	pr, found := k.GetProposal(ctx, proposalID)
	if !found {
		return errorsmod.Wrapf(types.ErrUnknownProposal, "%d", proposalID)
	}
	if pr.HasExpiredBy(ctx.BlockTime()) {
		return errorsmod.Wrapf(types.ErrProposalExpired, "%s ≥ %s", ctx.BlockTime(), pr.Deadline)
	}
	com, found := k.GetCommittee(ctx, pr.CommitteeID)
	if !found {
		return errorsmod.Wrapf(types.ErrUnknownCommittee, "%d", pr.CommitteeID)
	}

	if _, ok := com.(*types.MemberCommittee); ok {
		if !com.HasMember(voter) {
			return errorsmod.Wrap(sdkerrors.ErrUnauthorized, "voter must be a member of committee")
		}
		if voteType != types.VOTE_TYPE_YES {
			return errorsmod.Wrap(types.ErrInvalidVoteType, "member committees only accept yes votes")
		}
	}

	// Store vote, overwriting any prior vote
	k.SetVote(ctx, types.NewVote(proposalID, voter, voteType))

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeProposalVote,
			sdk.NewAttribute(types.AttributeKeyCommitteeID, fmt.Sprintf("%d", com.GetID())),
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", pr.ID)),
			sdk.NewAttribute(types.AttributeKeyVoter, voter.String()),
			sdk.NewAttribute(types.AttributeKeyVote, fmt.Sprintf("%d", voteType)),
		),
	)
	return nil
}

// ValidatePubProposal checks if a pubproposal is valid.
func (k Keeper) ValidatePubProposal(ctx sdk.Context, pubProposal types.PubProposal) (returnErr error) {
	if pubProposal == nil {
		return errorsmod.Wrap(types.ErrInvalidPubProposal, "pub proposal cannot be nil")
	}
	if err := pubProposal.ValidateBasic(); err != nil {
		return err
	}

	if !k.router.HasRoute(pubProposal.ProposalRoute()) {
		return errorsmod.Wrapf(types.ErrNoProposalHandlerExists, "%T", pubProposal)
	}

	// Run the proposal's changes through the associated handler using a cached version of state to ensure changes are not permanent.
	cacheCtx, _ := ctx.CacheContext()
	handler := k.router.GetRoute(pubProposal.ProposalRoute())

	// Handle an edge case where a param change proposal causes the proposal handler to panic.
	// A param change proposal with a registered subspace value but unregistered key value will cause a panic in the param change proposal handler.
	// This defer will catch panics and return a normal error: `recover()` gets the panic value, then the enclosing function's return value is swapped for an error.
	// reference: https://stackoverflow.com/questions/33167282/how-to-return-a-value-in-a-go-function-that-panics?noredirect=1&lq=1
	defer func() {
		if r := recover(); r != nil {
			returnErr = errorsmod.Wrapf(types.ErrInvalidPubProposal, "proposal handler panicked: %s", r)
		}
	}()

	if err := handler(cacheCtx, pubProposal); err != nil {
		return err
	}
	return nil
}

func (k Keeper) ProcessProposals(ctx sdk.Context) {
	k.IterateProposals(ctx, func(proposal types.Proposal) bool {
		committee, found := k.GetCommittee(ctx, proposal.CommitteeID)
		if !found {
			k.CloseProposal(ctx, proposal, types.Failed)
			return false
		}

		if !proposal.HasExpiredBy(ctx.BlockTime()) {
			if committee.GetTallyOption() == types.TALLY_OPTION_FIRST_PAST_THE_POST {
				passed := k.GetProposalResult(ctx, proposal.ID, committee)
				if passed {
					outcome := k.attemptEnactProposal(ctx, proposal)
					k.CloseProposal(ctx, proposal, outcome)
				}
			}
		} else {
			passed := k.GetProposalResult(ctx, proposal.ID, committee)
			outcome := types.Failed
			if passed {
				outcome = k.attemptEnactProposal(ctx, proposal)
			}
			k.CloseProposal(ctx, proposal, outcome)
		}
		return false
	})
}

func (k Keeper) GetProposalResult(ctx sdk.Context, proposalID uint64, committee types.Committee) bool {
	switch com := committee.(type) {
	case *types.MemberCommittee:
		return k.GetMemberCommitteeProposalResult(ctx, proposalID, com)
	case *types.TokenCommittee:
		return k.GetTokenCommitteeProposalResult(ctx, proposalID, com)
	default: // Should never hit default case
		return false
	}
}

// GetMemberCommitteeProposalResult gets the result of a member committee proposal
func (k Keeper) GetMemberCommitteeProposalResult(ctx sdk.Context, proposalID uint64, committee types.Committee) bool {
	currVotes := k.TallyMemberCommitteeVotes(ctx, proposalID)
	possibleVotes := sdk.NewDec(int64(len(committee.GetMembers())))
	return currVotes.GTE(committee.GetVoteThreshold().Mul(possibleVotes)) // vote threshold requirements
}

// TallyMemberCommitteeVotes returns the polling status of a member committee vote
func (k Keeper) TallyMemberCommitteeVotes(ctx sdk.Context, proposalID uint64) (totalVotes sdk.Dec) {
	votes := k.GetVotesByProposal(ctx, proposalID)
	return sdk.NewDec(int64(len(votes)))
}

// GetTokenCommitteeProposalResult gets the result of a token committee proposal
func (k Keeper) GetTokenCommitteeProposalResult(ctx sdk.Context, proposalID uint64, committee *types.TokenCommittee) bool {
	yesVotes, noVotes, totalVotes, possibleVotes := k.TallyTokenCommitteeVotes(ctx, proposalID, committee.TallyDenom)
	if totalVotes.GTE(committee.Quorum.Mul(possibleVotes)) { // quorum requirement
		nonAbstainVotes := yesVotes.Add(noVotes)
		if yesVotes.GTE(nonAbstainVotes.Mul(committee.VoteThreshold)) { // vote threshold requirements
			return true
		}
	}
	return false
}

// TallyMemberCommitteeVotes returns the polling status of a token committee vote. Returns yes votes,
// total current votes, total possible votes (equal to token supply), vote threshold (yes vote ratio
// required for proposal to pass), and quorum (votes tallied at this percentage).
func (k Keeper) TallyTokenCommitteeVotes(ctx sdk.Context, proposalID uint64,
	tallyDenom string,
) (yesVotes, noVotes, totalVotes, possibleVotes sdk.Dec) {
	votes := k.GetVotesByProposal(ctx, proposalID)

	yesVotes = sdk.ZeroDec()
	noVotes = sdk.ZeroDec()
	totalVotes = sdk.ZeroDec()
	for _, vote := range votes {
		// 1 token = 1 vote
		acc := k.accountKeeper.GetAccount(ctx, vote.Voter)
		accNumCoins := k.bankKeeper.GetBalance(ctx, acc.GetAddress(), tallyDenom).Amount

		// Add votes to counters
		totalVotes = totalVotes.Add(sdk.NewDecFromInt(accNumCoins))
		if vote.VoteType == types.VOTE_TYPE_YES {
			yesVotes = yesVotes.Add(sdk.NewDecFromInt(accNumCoins))
		} else if vote.VoteType == types.VOTE_TYPE_NO {
			noVotes = noVotes.Add(sdk.NewDecFromInt(accNumCoins))
		}
	}

	possibleVotesInt := k.bankKeeper.GetSupply(ctx, tallyDenom).Amount
	return yesVotes, noVotes, totalVotes, sdk.NewDecFromInt(possibleVotesInt)
}

func (k Keeper) attemptEnactProposal(ctx sdk.Context, proposal types.Proposal) types.ProposalOutcome {
	err := k.enactProposal(ctx, proposal)
	if err != nil {
		return types.Invalid
	}
	return types.Passed
}

// enactProposal makes the changes proposed in a proposal.
func (k Keeper) enactProposal(ctx sdk.Context, proposal types.Proposal) error {
	// Check committee still has permissions for the proposal
	// Since the proposal was submitted params could have changed, invalidating the permission of the committee.
	com, found := k.GetCommittee(ctx, proposal.CommitteeID)
	if !found {
		return errorsmod.Wrapf(types.ErrUnknownCommittee, "%d", proposal.CommitteeID)
	}
	if !com.HasPermissionsFor(ctx, k.cdc, k.paramKeeper, proposal.GetContent()) {
		return errorsmod.Wrap(sdkerrors.ErrUnauthorized, "committee does not have permissions to enact proposal")
	}

	if err := k.ValidatePubProposal(ctx, proposal.GetContent()); err != nil {
		return err
	}

	// enact the proposal
	handler := k.router.GetRoute(proposal.GetContent().ProposalRoute())
	if err := handler(ctx, proposal.GetContent()); err != nil {
		// the handler should not error as it was checked in ValidatePubProposal
		panic(fmt.Sprintf("unexpected handler error: %s", err))
	}
	return nil
}

// GetProposalTallyResponse returns the tally results of a proposal.
func (k Keeper) GetProposalTallyResponse(ctx sdk.Context, proposalID uint64) (*types.QueryTallyResponse, bool) {
	proposal, found := k.GetProposal(ctx, proposalID)
	if !found {
		return nil, found
	}
	committee, found := k.GetCommittee(ctx, proposal.CommitteeID)
	if !found {
		return nil, found
	}
	var proposalTally types.QueryTallyResponse
	switch com := committee.(type) {
	case *types.MemberCommittee:
		currVotes := k.TallyMemberCommitteeVotes(ctx, proposal.ID)
		possibleVotes := sdk.NewDec(int64(len(com.Members)))
		proposalTally = types.QueryTallyResponse{
			ProposalID:    proposal.ID,
			YesVotes:      currVotes,
			NoVotes:       sdk.ZeroDec(),
			CurrentVotes:  currVotes,
			PossibleVotes: possibleVotes,
			VoteThreshold: com.VoteThreshold,
			Quorum:        sdk.ZeroDec(),
		}
	case *types.TokenCommittee:
		yesVotes, noVotes, currVotes, possibleVotes := k.TallyTokenCommitteeVotes(ctx, proposal.ID, com.TallyDenom)
		proposalTally = types.QueryTallyResponse{
			ProposalID:    proposal.ID,
			YesVotes:      yesVotes,
			NoVotes:       noVotes,
			CurrentVotes:  currVotes,
			PossibleVotes: possibleVotes,
			VoteThreshold: com.VoteThreshold,
			Quorum:        com.Quorum,
		}
	}
	return &proposalTally, true
}

// CloseProposal deletes proposals and their votes, emitting an event denoting the final status of the proposal
func (k Keeper) CloseProposal(ctx sdk.Context, proposal types.Proposal, outcome types.ProposalOutcome) {
	tally, _ := k.GetProposalTallyResponse(ctx, proposal.ID)
	k.DeleteProposalAndVotes(ctx, proposal.ID)

	bz, err := k.cdc.MarshalJSON(tally)
	if err != nil {
		fmt.Println("error marshaling proposal tally to bytes:", tally.String())
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeProposalClose,
			sdk.NewAttribute(types.AttributeKeyCommitteeID, fmt.Sprintf("%d", proposal.CommitteeID)),
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.ID)),
			sdk.NewAttribute(types.AttributeKeyProposalTally, string(bz)),
			sdk.NewAttribute(types.AttributeKeyProposalOutcome, outcome.String()),
		),
	)
}
