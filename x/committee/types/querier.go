package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Query endpoints supported by the Querier
const (
	QueryCommittees     = "committees"
	QueryCommittee      = "committee"
	QueryProposals      = "proposals"
	QueryProposal       = "proposal"
	QueryNextProposalID = "next-proposal-id"
	QueryVotes          = "votes"
	QueryVote           = "vote"
	QueryTally          = "tally"
	QueryRawParams      = "raw_params"
)

type QueryCommitteeParams struct {
	CommitteeID uint64 `json:"committee_id" yaml:"committee_id"`
}

func NewQueryCommitteeParams(committeeID uint64) QueryCommitteeParams {
	return QueryCommitteeParams{
		CommitteeID: committeeID,
	}
}

type QueryProposalParams struct {
	ProposalID uint64 `json:"proposal_id" yaml:"proposal_id"`
}

func NewQueryProposalParams(proposalID uint64) QueryProposalParams {
	return QueryProposalParams{
		ProposalID: proposalID,
	}
}

type QueryVoteParams struct {
	ProposalID uint64         `json:"proposal_id" yaml:"proposal_id"`
	Voter      sdk.AccAddress `json:"voter" yaml:"voter"`
}

func NewQueryVoteParams(proposalID uint64, voter sdk.AccAddress) QueryVoteParams {
	return QueryVoteParams{
		ProposalID: proposalID,
		Voter:      voter,
	}
}

type QueryRawParamsParams struct {
	Subspace string
	Key      string
}

func NewQueryRawParamsParams(subspace, key string) QueryRawParamsParams {
	return QueryRawParamsParams{
		Subspace: subspace,
		Key:      key,
	}
}

type ProposalPollingStatus struct {
	ProposalID    uint64  `json:"proposal_id" yaml:"proposal_id"`
	YesVotes      sdk.Dec `json:"yes_votes" yaml:"yes_votes"`
	NoVotes       sdk.Dec `json:"no_votes" yaml:"no_votes"`
	CurrentVotes  sdk.Dec `json:"current_votes" yaml:"current_votes"`
	PossibleVotes sdk.Dec `json:"possible_votes" yaml:"possible_votes"`
	VoteThreshold sdk.Dec `json:"vote_threshold" yaml:"vote_threshold"`
	Quorum        sdk.Dec `json:"quorum" yaml:"quorum"`
}

func NewProposalPollingStatus(proposalID uint64, yesVotes, noVotes, currentVotes, possibleVotes,
	voteThreshold, quorum sdk.Dec) ProposalPollingStatus {
	return ProposalPollingStatus{
		ProposalID:    proposalID,
		YesVotes:      yesVotes,
		NoVotes:       noVotes,
		CurrentVotes:  currentVotes,
		PossibleVotes: possibleVotes,
		VoteThreshold: voteThreshold,
		Quorum:        quorum,
	}
}

// String implements fmt.Stringer
func (p ProposalPollingStatus) String() string {
	return fmt.Sprintf(`Proposal ID: %d
	Yes votes:         %d
	No votes:          %d
	Current votes:     %d
  	Possible votes:    %d
  	Vote threshold:    %d
	Quorum:        	   %d`,
		p.ProposalID, p.YesVotes, p.NoVotes, p.CurrentVotes,
		p.PossibleVotes, p.VoteThreshold, p.Quorum,
	)
}
