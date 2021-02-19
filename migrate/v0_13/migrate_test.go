package v0_13

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/kava-labs/kava/app"
	v0_11cdp "github.com/kava-labs/kava/x/cdp/legacy/v0_11"
	v0_13committee "github.com/kava-labs/kava/x/committee"
	v0_11committee "github.com/kava-labs/kava/x/committee/legacy/v0_11"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	app.SetBip44CoinType(config)

	os.Exit(m.Run())
}

func TestCDP(t *testing.T) {
	bz, err := ioutil.ReadFile(filepath.Join("testdata", "kava-4-cdp-state-block-500000.json"))
	require.NoError(t, err)
	var oldGenState v0_11cdp.GenesisState
	cdc := app.MakeCodec()
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldGenState)
	})

	newGenState := CDP(oldGenState)
	err = newGenState.Validate()
	require.NoError(t, err)

	require.Equal(t, len(newGenState.Params.CollateralParams), len(newGenState.PreviousAccumulationTimes))

	cdp1 := newGenState.CDPs[0]
	require.Equal(t, sdk.OneDec(), cdp1.InterestFactor)

}

func TestAuth(t *testing.T) {
	bz, err := ioutil.ReadFile(filepath.Join("testdata", "kava-4-auth-state-block-500000.json"))
	require.NoError(t, err)
	var oldGenState auth.GenesisState
	cdc := app.MakeCodec()
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldGenState)
	})
	newGenState := Auth(oldGenState)
	err = auth.ValidateGenesis(newGenState)
	require.NoError(t, err)
	require.Equal(t, len(oldGenState.Accounts), len(newGenState.Accounts)+1)

}

func TestCommittee(t *testing.T) {
	bz, err := ioutil.ReadFile(filepath.Join("testdata", "kava-4-committee-state.json"))
	require.NoError(t, err)
	var oldGenState v0_11committee.GenesisState
	cdc := codec.New()
	sdk.RegisterCodec(cdc)
	v0_11committee.RegisterCodec(cdc)
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldGenState)
	})

	newGenState := Committee(oldGenState)
	err = newGenState.Validate()
	require.NoError(t, err)

	require.Equal(t, len(oldGenState.Committees), len(newGenState.Committees))

	for i := 0; i < len(oldGenState.Committees); i++ {
		require.Equal(t, len(oldGenState.Committees[i].Permissions), len(newGenState.Committees[i].Permissions))
	}

	oldSPCP := oldGenState.Committees[0].Permissions[0].(v0_11committee.SubParamChangePermission)
	newSPCP := newGenState.Committees[0].Permissions[0].(v0_13committee.SubParamChangePermission)
	require.Equal(t, len(oldSPCP.AllowedParams), len(newSPCP.AllowedParams))
	require.Equal(t, len(oldSPCP.AllowedAssetParams), len(newSPCP.AllowedAssetParams))
	require.Equal(t, len(oldSPCP.AllowedCollateralParams), len(newSPCP.AllowedCollateralParams))
	require.Equal(t, len(oldSPCP.AllowedMarkets), len(newSPCP.AllowedMarkets))
}
