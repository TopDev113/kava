package keeper_test

// import (
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/suite"
// 	abci "github.com/tendermint/tendermint/abci/types"

// 	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
// 	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

// 	"github.com/kava-labs/kava/app"
// 	"github.com/kava-labs/kava/x/committee/types"
// )

// type TypesTestSuite struct {
// 	suite.Suite
// }

// func (suite *TypesTestSuite) TestCommittee_HasPermissionsFor() {

// 	testcases := []struct {
// 		name                 string
// 		permissions          []types.Permission
// 		pubProposal          types.PubProposal
// 		expectHasPermissions bool
// 	}{
// 		{
// 			name: "normal (single permission)",
// 			permissions: []types.Permission{types.SimpleParamChangePermission{
// 				AllowedParams: types.AllowedParams{
// 					{
// 						Subspace: "cdp",
// 						Key:      "DebtThreshold",
// 					},
// 				}}},
// 			pubProposal: paramstypes.NewParameterChangeProposal(
// 				"A Title",
// 				"A description of this proposal.",
// 				[]paramstypes.ParamChange{
// 					{
// 						Subspace: "cdp",
// 						Key:      "DebtThreshold",

// 						Value: `{"denom": "usdx", "amount": "1000000"}`,
// 					},
// 				},
// 			),
// 			expectHasPermissions: true,
// 		},
// 		{
// 			name: "normal (multiple permissions)",
// 			permissions: []types.Permission{
// 				types.SimpleParamChangePermission{
// 					AllowedParams: types.AllowedParams{
// 						{
// 							Subspace: "cdp",
// 							Key:      "DebtThreshold",
// 						},
// 					}},
// 				types.TextPermission{},
// 			},
// 			pubProposal:          govtypes.NewTextProposal("A Proposal Title", "A description of this proposal"),
// 			expectHasPermissions: true,
// 		},
// 		{
// 			name: "overruling permission",
// 			permissions: []types.Permission{
// 				types.SimpleParamChangePermission{
// 					AllowedParams: types.AllowedParams{
// 						{
// 							Subspace: "cdp",
// 							Key:      "DebtThreshold",
// 						},
// 					}},
// 				types.GodPermission{},
// 			},
// 			pubProposal: paramstypes.NewParameterChangeProposal(
// 				"A Title",
// 				"A description of this proposal.",
// 				[]paramstypes.ParamChange{
// 					{
// 						Subspace: "cdp",
// 						Key:      "CollateralParams",

// 						Value: `[]`,
// 					},
// 				},
// 			),
// 			expectHasPermissions: true,
// 		},
// 		{
// 			name:        "no permissions",
// 			permissions: nil,
// 			pubProposal: paramstypes.NewParameterChangeProposal(
// 				"A Title",
// 				"A description of this proposal.",
// 				[]paramstypes.ParamChange{
// 					{
// 						Subspace: "cdp",
// 						Key:      "CollateralParams",

// 						Value: `[]`,
// 					},
// 				},
// 			),
// 			expectHasPermissions: false,
// 		},
// 		{
// 			name: "split permissions",
// 			// These permissions looks like they allow the param change proposal, however a proposal must pass a single permission independently of others.
// 			permissions: []types.Permission{
// 				types.SimpleParamChangePermission{
// 					AllowedParams: types.AllowedParams{
// 						{
// 							Subspace: "cdp",
// 							Key:      "DebtThreshold",
// 						},
// 					}},
// 				types.SimpleParamChangePermission{
// 					AllowedParams: types.AllowedParams{
// 						{
// 							Subspace: "cdp",
// 							Key:      "DebtParams",
// 						},
// 					}},
// 			},
// 			pubProposal: paramstypes.NewParameterChangeProposal(
// 				"A Title",
// 				"A description of this proposal.",
// 				[]paramstypes.ParamChange{
// 					{
// 						Subspace: "cdp",
// 						Key:      "DebtThreshold",

// 						Value: `{"denom": "usdx", "amount": "1000000"}`,
// 					},
// 					{
// 						Subspace: "cdp",
// 						Key:      "DebtParams",

// 						Value: `[]`,
// 					},
// 				},
// 			),
// 			expectHasPermissions: false,
// 		},
// 		{
// 			name: "unregistered proposal",
// 			permissions: []types.Permission{
// 				types.SimpleParamChangePermission{
// 					AllowedParams: types.AllowedParams{
// 						{
// 							Subspace: "cdp",
// 							Key:      "DebtThreshold",
// 						},
// 					}},
// 			},
// 			pubProposal:          UnregisteredPubProposal{govtypes.TextProposal{Title: "A Title", Description: "A description."}},
// 			expectHasPermissions: false,
// 		},
// 	}

// 	for _, tc := range testcases {
// 		suite.Run(tc.name, func() {
// 			tApp := app.NewTestApp()
// 			ctx := tApp.NewContext(true, abci.Header{})
// 			tApp.InitializeFromGenesisStates()
// 			com := types.NewMemberCommittee(
// 				12,
// 				"a description of this committee",
// 				nil,
// 				tc.permissions,
// 				d("0.5"),
// 				24*time.Hour,
// 				types.FirstPastThePost,
// 			)
// 			suite.Equal(
// 				tc.expectHasPermissions,
// 				com.HasPermissionsFor(ctx, tApp.Codec(), tApp.GetParamsKeeper(), tc.pubProposal),
// 			)
// 		})
// 	}
// }

// func TestTypesTestSuite(t *testing.T) {
// 	suite.Run(t, new(TypesTestSuite))
// }
