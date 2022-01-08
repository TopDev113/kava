package app

import (
	"os"
	"testing"

	"github.com/tendermint/tendermint/libs/log"
	db "github.com/tendermint/tm-db"
)

func TestNewApp(t *testing.T) {

	NewApp(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		db.NewMemDB(),
		DefaultNodeHome,
		nil,
		MakeEncodingConfig(),
		Options{},
	)
}

// func TestExport(t *testing.T) {
// 	db := db.NewMemDB()
// 	app := NewApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, AppOptions{})
// 	setGenesis(app)

// 	// Making a new app object with the db, so that initchain hasn't been called
// 	newApp := NewApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, AppOptions{})
// 	_, _, err := newApp.ExportAppStateAndValidators(false, []string{})
// 	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
// }

// // ensure that black listed addresses are properly set in bank keeper
// func TestBlackListedAddrs(t *testing.T) {
// 	db := db.NewMemDB()
// 	app := NewApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, AppOptions{})

// 	for acc := range mAccPerms {
// 		require.Equal(t, !allowedReceivingModAcc[acc], app.bankKeeper.BlacklistedAddr(app.supplyKeeper.GetModuleAddress(acc)))
// 	}
// }

// func setGenesis(app *App) error {
// 	genesisState := NewDefaultGenesisState()

// 	stateBytes, err := codec.MarshalJSONIndent(app.cdc, genesisState)
// 	if err != nil {
// 		return err
// 	}

// 	// Initialize the chain
// 	app.InitChain(
// 		abci.RequestInitChain{
// 			Validators:    []abci.ValidatorUpdate{},
// 			AppStateBytes: stateBytes,
// 		},
// 	)
// 	app.Commit()

// 	return nil
// }
