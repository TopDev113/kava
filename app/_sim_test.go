package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/simapp/helpers"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/supply"

	"github.com/kava-labs/kava/x/auction"
	"github.com/kava-labs/kava/x/bep3"
	"github.com/kava-labs/kava/x/cdp"
	"github.com/kava-labs/kava/x/committee"
	"github.com/kava-labs/kava/x/incentive"
	"github.com/kava-labs/kava/x/kavadist"
	"github.com/kava-labs/kava/x/pricefeed"
	"github.com/kava-labs/kava/x/swap"
	validatorvesting "github.com/kava-labs/kava/x/validator-vesting"
)

type StoreKeysPrefixes struct {
	A        sdk.StoreKey
	B        sdk.StoreKey
	Prefixes [][]byte
}

// TestMain runs setup and teardown code before all tests.
func TestMain(m *testing.M) {
	// set prefixes
	config := sdk.GetConfig()
	SetBech32AddressPrefixes(config)
	config.Seal()
	// load the values from simulation specific flags
	simapp.GetSimulatorFlags()
	// run tests
	exitCode := m.Run()
	os.Exit(exitCode)
}

// fauxMerkleModeOpt returns a BaseApp option to use a dbStoreAdapter instead of
// an IAVLStore for faster simulation speed.
func fauxMerkleModeOpt(bapp *baseapp.BaseApp) {
	bapp.SetFauxMerkleMode()
}

// interBlockCacheOpt returns a BaseApp option function that sets the persistent
// inter-block write-through cache.
func interBlockCacheOpt() func(*baseapp.BaseApp) {
	return baseapp.SetInterBlockCache(store.NewCommitKVStoreCacheManager())
}

func TestFullAppSimulation(t *testing.T) {
	config, db, dir, logger, skip, err := simapp.SetupSimulation("leveldb-app-sim", "Simulation")
	if skip {
		t.Skip("skipping application simulation")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		db.Close()
		require.NoError(t, os.RemoveAll(dir))
	}()

	app := NewApp(logger, db, nil, AppOptions{InvariantCheckPeriod: simapp.FlagPeriodValue}, fauxMerkleModeOpt)
	require.Equal(t, appName, app.Name())

	// run randomized simulation
	_, simParams, simErr := simulation.SimulateFromSeed(
		t, os.Stdout, app.BaseApp, AppStateFn(app.Codec(), app.SimulationManager()),
		simapp.SimulationOperations(app, app.Codec(), config),
		app.ModuleAccountAddrs(), config,
	)

	// export state and simParams before the simulation error is checked
	err = simapp.CheckExportSimulation(app, config, simParams)
	require.NoError(t, err)
	require.NoError(t, simErr)

	if config.Commit {
		simapp.PrintStats(db)
	}
}

func TestAppImportExport(t *testing.T) {
	config, db, dir, logger, skip, err := simapp.SetupSimulation("leveldb-app-sim", "Simulation")
	if skip {
		t.Skip("skipping application import/export simulation")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		db.Close()
		require.NoError(t, os.RemoveAll(dir))
	}()

	app := NewApp(logger, db, nil, AppOptions{InvariantCheckPeriod: simapp.FlagPeriodValue}, fauxMerkleModeOpt)
	require.Equal(t, appName, app.Name())

	// Run randomized simulation
	_, simParams, simErr := simulation.SimulateFromSeed(
		t, os.Stdout, app.BaseApp, AppStateFn(app.Codec(), app.SimulationManager()),
		simapp.SimulationOperations(app, app.Codec(), config),
		app.ModuleAccountAddrs(), config,
	)

	// export state and simParams before the simulation error is checked
	err = simapp.CheckExportSimulation(app, config, simParams)
	require.NoError(t, err)
	require.NoError(t, simErr)

	if config.Commit {
		simapp.PrintStats(db)
	}

	fmt.Printf("exporting genesis...\n")

	appState, _, err := app.ExportAppStateAndValidators(false, []string{})
	require.NoError(t, err)

	fmt.Printf("importing genesis...\n")

	_, newDB, newDir, _, _, err := simapp.SetupSimulation("leveldb-app-sim-2", "Simulation-2")
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		newDB.Close()
		require.NoError(t, os.RemoveAll(newDir))
	}()

	newApp := NewApp(log.NewNopLogger(), newDB, nil, AppOptions{InvariantCheckPeriod: simapp.FlagPeriodValue}, fauxMerkleModeOpt)
	require.Equal(t, appName, newApp.Name())

	var genesisState GenesisState
	err = app.Codec().UnmarshalJSON(appState, &genesisState)
	require.NoError(t, err)

	ctxA := app.NewContext(true, abci.Header{Height: app.LastBlockHeight()})
	ctxB := newApp.NewContext(true, abci.Header{Height: app.LastBlockHeight()})
	newApp.mm.InitGenesis(ctxB, genesisState)

	fmt.Printf("comparing stores...\n")

	storeKeysPrefixes := []StoreKeysPrefixes{
		{app.keys[baseapp.MainStoreKey], newApp.keys[baseapp.MainStoreKey], [][]byte{}},
		{app.keys[auth.StoreKey], newApp.keys[auth.StoreKey], [][]byte{}},
		{
			app.keys[staking.StoreKey], newApp.keys[staking.StoreKey],
			[][]byte{
				staking.UnbondingQueueKey, staking.RedelegationQueueKey, staking.ValidatorQueueKey,
			},
		}, // ordering may change but it doesn't matter
		{app.keys[slashing.StoreKey], newApp.keys[slashing.StoreKey], [][]byte{}},
		{app.keys[mint.StoreKey], newApp.keys[mint.StoreKey], [][]byte{}},
		{app.keys[distr.StoreKey], newApp.keys[distr.StoreKey], [][]byte{}},
		{app.keys[supply.StoreKey], newApp.keys[supply.StoreKey], [][]byte{}},
		{app.keys[params.StoreKey], newApp.keys[params.StoreKey], [][]byte{}},
		{app.keys[gov.StoreKey], newApp.keys[gov.StoreKey], [][]byte{}},
		{app.keys[auction.StoreKey], newApp.keys[auction.StoreKey], [][]byte{}},
		{app.keys[bep3.StoreKey], newApp.keys[bep3.StoreKey], [][]byte{}},
		{app.keys[cdp.StoreKey], newApp.keys[cdp.StoreKey], [][]byte{}},
		{app.keys[incentive.StoreKey], newApp.keys[incentive.StoreKey], [][]byte{}},
		{app.keys[kavadist.StoreKey], newApp.keys[kavadist.StoreKey], [][]byte{}},
		{app.keys[pricefeed.StoreKey], newApp.keys[pricefeed.StoreKey], [][]byte{}},
		{app.keys[validatorvesting.StoreKey], newApp.keys[validatorvesting.StoreKey], [][]byte{}},
		{app.keys[committee.StoreKey], newApp.keys[committee.StoreKey], [][]byte{}},
		{app.keys[swap.StoreKey], newApp.keys[swap.StoreKey], [][]byte{}},
	}

	for _, skp := range storeKeysPrefixes {
		storeA := ctxA.KVStore(skp.A)
		storeB := ctxB.KVStore(skp.B)

		failedKVAs, failedKVBs := sdk.DiffKVStores(storeA, storeB, skp.Prefixes)
		require.Equal(t, len(failedKVAs), len(failedKVBs), "unequal sets of key-values to compare")
		if len(failedKVAs) != 0 {
			fmt.Printf("found %d non-equal key/value pairs between %s and %s\n", len(failedKVAs), skp.A, skp.B)
		}
		require.Equal(t, len(failedKVAs), 0, simapp.GetSimulationLog(skp.A.Name(), app.SimulationManager().StoreDecoders, app.Codec(), failedKVAs, failedKVBs))
	}
}

func TestAppSimulationAfterImport(t *testing.T) {
	config, db, dir, logger, skip, err := simapp.SetupSimulation("leveldb-app-sim", "Simulation")
	if skip {
		t.Skip("skipping application simulation after import")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		db.Close()
		require.NoError(t, os.RemoveAll(dir))
	}()

	app := NewApp(logger, db, nil, AppOptions{InvariantCheckPeriod: simapp.FlagPeriodValue}, fauxMerkleModeOpt)
	require.Equal(t, appName, app.Name())

	// Run randomized simulation
	stopEarly, simParams, simErr := simulation.SimulateFromSeed(
		t, os.Stdout, app.BaseApp, AppStateFn(app.Codec(), app.SimulationManager()),
		simapp.SimulationOperations(app, app.Codec(), config),
		app.ModuleAccountAddrs(), config,
	)

	// export state and simParams before the simulation error is checked
	err = simapp.CheckExportSimulation(app, config, simParams)
	require.NoError(t, err)
	require.NoError(t, simErr)

	if config.Commit {
		simapp.PrintStats(db)
	}

	if stopEarly {
		fmt.Println("can't export or import a zero-validator genesis, exiting test...")
		return
	}

	fmt.Printf("exporting genesis...\n")

	appState, _, err := app.ExportAppStateAndValidators(true, []string{})
	require.NoError(t, err)

	fmt.Printf("importing genesis...\n")

	_, newDB, newDir, _, _, err := simapp.SetupSimulation("leveldb-app-sim-2", "Simulation-2")
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		newDB.Close()
		require.NoError(t, os.RemoveAll(newDir))
	}()

	newApp := NewApp(log.NewNopLogger(), newDB, nil, AppOptions{InvariantCheckPeriod: simapp.FlagPeriodValue}, fauxMerkleModeOpt)
	require.Equal(t, appName, newApp.Name())

	newApp.InitChain(abci.RequestInitChain{
		AppStateBytes: appState,
	})

	_, _, err = simulation.SimulateFromSeed(
		t, os.Stdout, newApp.BaseApp, AppStateFn(app.Codec(), app.SimulationManager()),
		simapp.SimulationOperations(newApp, newApp.Codec(), config),
		newApp.ModuleAccountAddrs(), config,
	)
	require.NoError(t, err)
}

func TestAppStateDeterminism(t *testing.T) {
	if !simapp.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	config := simapp.NewConfigFromFlags()
	config.InitialBlockHeight = 1
	config.ExportParamsPath = ""
	config.OnOperation = false
	config.AllInvariants = false
	config.ChainID = helpers.SimAppChainID

	numTimesToRunPerSeed := 2
	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)

	for j := 0; j < numTimesToRunPerSeed; j++ {
		logger := log.NewNopLogger()
		db := dbm.NewMemDB()
		app := NewApp(logger, db, nil, AppOptions{InvariantCheckPeriod: simapp.FlagPeriodValue}, interBlockCacheOpt())

		fmt.Printf(
			"running non-determinism simulation; seed %d: attempt: %d/%d\n",
			config.Seed, j+1, numTimesToRunPerSeed,
		)

		_, _, err := simulation.SimulateFromSeed(
			t, os.Stdout, app.BaseApp, AppStateFn(app.Codec(), app.SimulationManager()),
			simapp.SimulationOperations(app, app.Codec(), config),
			app.ModuleAccountAddrs(), config,
		)
		require.NoError(t, err)

		appHash := app.LastCommitID().Hash
		appHashList[j] = appHash

		if j != 0 {
			require.Equal(
				t, appHashList[0], appHashList[j],
				"non-determinism in seed %d: attempt: %d/%d\n", config.Seed, j+1, numTimesToRunPerSeed,
			)
		}
	}
}

// AppStateFn returns the initial application state using a genesis or the simulation parameters.
// It panics if the user provides files for both of them.
// If a file is not given for the genesis or the sim params, it creates a randomized one.
// Note: this was copied in from the sdk/simapp/state.go, and modified to not generate genesis times too far in the future.
// Dates greater than the year 9000 interfere with new auctions who's EndTime is set to 90000.
func AppStateFn(cdc *codec.Codec, simManager *module.SimulationManager) simulation.AppStateFn {
	return func(r *rand.Rand, accs []simulation.Account, config simulation.Config,
	) (appState json.RawMessage, simAccs []simulation.Account, chainID string, genesisTimestamp time.Time) {
		if simapp.FlagGenesisTimeValue == 0 {
			genesisTimestamp = time.Unix(r.Int63n(190288396800), 0) // 1st Jan year 8000
		} else {
			genesisTimestamp = time.Unix(simapp.FlagGenesisTimeValue, 0)
		}

		chainID = config.ChainID
		switch {
		case config.ParamsFile != "" && config.GenesisFile != "":
			panic("cannot provide both a genesis file and a params file")

		case config.GenesisFile != "":
			// override the default chain-id from simapp to set it later to the config
			genesisDoc, accounts := simapp.AppStateFromGenesisFileFn(r, cdc, config.GenesisFile)

			if simapp.FlagGenesisTimeValue == 0 {
				// use genesis timestamp if no custom timestamp is provided (i.e no random timestamp)
				genesisTimestamp = genesisDoc.GenesisTime
			}

			appState = genesisDoc.AppState
			chainID = genesisDoc.ChainID
			simAccs = accounts

		case config.ParamsFile != "":
			appParams := make(simulation.AppParams)
			bz, err := ioutil.ReadFile(config.ParamsFile)
			if err != nil {
				panic(err)
			}

			cdc.MustUnmarshalJSON(bz, &appParams)
			appState, simAccs = simapp.AppStateRandomizedFn(simManager, r, cdc, accs, genesisTimestamp, appParams)

		default:
			appParams := make(simulation.AppParams)
			appState, simAccs = simapp.AppStateRandomizedFn(simManager, r, cdc, accs, genesisTimestamp, appParams)
		}

		return appState, simAccs, chainID, genesisTimestamp
	}
}
