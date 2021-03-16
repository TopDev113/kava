package v0_14

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/supply"
	supplyexported "github.com/cosmos/cosmos-sdk/x/supply/exported"

	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/bep3"
	v0_11cdp "github.com/kava-labs/kava/x/cdp/legacy/v0_11"
	v0_14committee "github.com/kava-labs/kava/x/committee"
	v0_11committee "github.com/kava-labs/kava/x/committee/legacy/v0_11"
	v0_14hard "github.com/kava-labs/kava/x/hard"
	v0_11hard "github.com/kava-labs/kava/x/hard/legacy/v0_11"
	v0_14incentive "github.com/kava-labs/kava/x/incentive"
	v0_11incentive "github.com/kava-labs/kava/x/incentive/legacy/v0_11"
	v0_11pricefeed "github.com/kava-labs/kava/x/pricefeed"
	validatorvesting "github.com/kava-labs/kava/x/validator-vesting"

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
	validatorVestingChangeAddress, err := sdk.AccAddressFromBech32("kava1a3qmze57knfj29a5knqs5ptewh76v4fg23xsvn")
	if err != nil {
		panic(err)
	}
	validatorVestingUpdatedValAddress, err := sdk.ConsAddressFromBech32("kavavalcons1ucxhn6zh7y2zun49m36psjffrhmux7ukqxdcte")
	if err != nil {
		panic(err)
	}
	bz, err := ioutil.ReadFile(filepath.Join("testdata", "kava-4-auth-state-block-500000.json"))
	require.NoError(t, err)
	var oldGenState auth.GenesisState
	cdc := app.MakeCodec()
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldGenState)
	})
	harvestCoins := getModuleAccount(oldGenState.Accounts, "harvest").GetCoins()

	newGenState := Auth(oldGenState)
	for _, acc := range newGenState.Accounts {
		if acc.GetAddress().Equals(validatorVestingChangeAddress) {
			vacc := acc.(*validatorvesting.ValidatorVestingAccount)
			require.Equal(t, int64(0), vacc.CurrentPeriodProgress.MissedBlocks)
			require.Equal(t, validatorVestingUpdatedValAddress, vacc.ValidatorAddress)
		}
	}

	err = auth.ValidateGenesis(newGenState)
	require.NoError(t, err)
	require.Equal(t, len(oldGenState.Accounts), len(newGenState.Accounts)+3)
	require.Nil(t, getModuleAccount(newGenState.Accounts, "harvest"))
	require.Equal(t, getModuleAccount(newGenState.Accounts, "hard").GetCoins(), harvestCoins)
}

func getModuleAccount(accounts authexported.GenesisAccounts, name string) supplyexported.ModuleAccountI {
	modAcc, ok := getAccount(accounts, supply.NewModuleAddress(name)).(supplyexported.ModuleAccountI)
	if !ok {
		return nil
	}
	return modAcc
}
func getAccount(accounts authexported.GenesisAccounts, address sdk.AccAddress) authexported.GenesisAccount {
	for _, acc := range accounts {
		if acc.GetAddress().Equals(address) {
			return acc
		}
	}
	return nil
}

func TestIncentive(t *testing.T) {
	bz, err := ioutil.ReadFile(filepath.Join("testdata", "kava-4-incentive-state.json"))
	require.NoError(t, err)
	var oldIncentiveGenState v0_11incentive.GenesisState
	cdc := app.MakeCodec()
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldIncentiveGenState)
	})

	bz, err = ioutil.ReadFile(filepath.Join("testdata", "kava-4-harvest-state.json"))
	require.NoError(t, err)
	var oldHarvestGenState v0_11hard.GenesisState
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldHarvestGenState)
	})
	newGenState := v0_14incentive.GenesisState{}
	require.NotPanics(t, func() {
		newGenState = Incentive(oldHarvestGenState, oldIncentiveGenState)
	})
	err = newGenState.Validate()
	require.NoError(t, err)
	fmt.Printf("Number of incentive claims in kava-4: %d\nNumber of incentive Claims in kava-5: %d\n",
		len(oldIncentiveGenState.Claims), len(newGenState.USDXMintingClaims),
	)
	fmt.Printf("Number of harvest claims in kava-4: %d\nNumber of hard claims in kava-5: %d\n", len(oldHarvestGenState.Claims), len(newGenState.HardLiquidityProviderClaims))
}

func TestHard(t *testing.T) {
	cdc := app.MakeCodec()
	bz, err := ioutil.ReadFile(filepath.Join("testdata", "kava-4-harvest-state.json"))
	require.NoError(t, err)
	var oldHarvestGenState v0_11hard.GenesisState
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldHarvestGenState)
	})
	newGenState := v0_14hard.GenesisState{}
	require.NotPanics(t, func() {
		newGenState = Hard(oldHarvestGenState)
	})
	err = newGenState.Validate()
	require.NoError(t, err)
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
	newSPCP := newGenState.Committees[0].Permissions[0].(v0_14committee.SubParamChangePermission)
	require.Equal(t, len(oldSPCP.AllowedParams)-14, len(newSPCP.AllowedParams)) // accounts for removed/redundant keys
	require.Equal(t, len(oldSPCP.AllowedAssetParams), len(newSPCP.AllowedAssetParams))
	require.Equal(t, len(oldSPCP.AllowedCollateralParams)+3, len(newSPCP.AllowedCollateralParams)) // accounts for new cdp collateral types
	require.Equal(t, len(oldSPCP.AllowedMarkets), len(newSPCP.AllowedMarkets))
}

func TestPricefeed(t *testing.T) {
	cdc := app.MakeCodec()
	bz, err := ioutil.ReadFile(filepath.Join("testdata", "kava-4-pricefeed-state.json"))
	require.NoError(t, err)
	var oldPricefeedGenState v0_11pricefeed.GenesisState
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldPricefeedGenState)
	})
	newGenState := Pricefeed(oldPricefeedGenState)
	err = newGenState.Validate()
	require.NoError(t, err)
	require.Equal(t, len(oldPricefeedGenState.Params.Markets)+1, len(newGenState.Params.Markets))
}
func TestBep3(t *testing.T) {
	bz, err := ioutil.ReadFile(filepath.Join("testdata", "kava-4-bep3-state.json"))
	require.NoError(t, err)
	var oldGenState bep3.GenesisState
	cdc := app.MakeCodec()
	require.NotPanics(t, func() {
		cdc.MustUnmarshalJSON(bz, &oldGenState)
	})
	newGenState := Bep3(oldGenState)
	err = newGenState.Validate()
	require.NoError(t, err)

	var oldBNBSupply bep3.AssetSupply
	var newBNBSupply bep3.AssetSupply

	for _, supply := range oldGenState.Supplies {
		if supply.GetDenom() == "bnb" {
			oldBNBSupply = supply
		}
	}

	for _, supply := range newGenState.Supplies {
		if supply.GetDenom() == "bnb" {
			newBNBSupply = supply
		}
	}

	require.Equal(t, oldBNBSupply.CurrentSupply.Sub(sdk.NewCoin("bnb", sdk.NewInt(1000000000000))), newBNBSupply.CurrentSupply)
	require.Equal(t, uint64(24686), newGenState.Params.AssetParams[0].MinBlockLock)
	require.Equal(t, uint64(86400), newGenState.Params.AssetParams[0].MaxBlockLock)
}

func TestMigrateFull(t *testing.T) {
	oldGenDoc, err := tmtypes.GenesisDocFromFile(filepath.Join("testdata", "kava-4-export.json"))
	require.NoError(t, err)

	// 2) migrate
	newGenDoc := Migrate(*oldGenDoc)
	tApp := app.NewTestApp()
	cdc := app.MakeCodec()
	var newAppState genutil.AppMap
	require.NoError(t,
		cdc.UnmarshalJSON(newGenDoc.AppState, &newAppState),
	)
	err = app.ModuleBasics.ValidateGenesis(newAppState)
	if err != nil {
		require.NoError(t, err)
	}
	require.NotPanics(t, func() {
		// this runs both InitGenesis for all modules (which panic on errors) and runs all invariants
		tApp.InitializeFromGenesisStatesWithTime(newGenDoc.GenesisTime, app.GenesisState(newAppState))
	})
}
