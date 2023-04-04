package bep3_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/bep3/keeper"
	"github.com/kava-labs/kava/x/bep3/types"
)

type GenesisTestSuite struct {
	suite.Suite

	app    app.TestApp
	ctx    sdk.Context
	keeper keeper.Keeper
	addrs  []sdk.AccAddress
}

func (suite *GenesisTestSuite) SetupTest() {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)

	tApp := app.NewTestApp()
	suite.ctx = tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})
	suite.keeper = tApp.GetBep3Keeper()
	suite.app = tApp

	_, addrs := app.GeneratePrivKeyAddressPairs(3)
	suite.addrs = addrs
}

func (suite *GenesisTestSuite) TestModulePermissionsCheck() {
	cdc := suite.app.AppCodec()

	testCases := []struct {
		name          string
		permissions   []string
		expectedPanic string
	}{
		{"no permissions", []string{}, "bep3 module account does not have burn permissions"},
		{"mint permissions", []string{authtypes.Minter}, "bep3 module account does not have burn permissions"},
		{"burn permissions", []string{authtypes.Burner}, "bep3 module account does not have mint permissions"},
		{"burn and mint permissions", []string{authtypes.Burner, authtypes.Minter}, ""},
		{"mint and burn permissions", []string{authtypes.Minter, authtypes.Burner}, ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			authGenesis := authtypes.NewGenesisState(
				authtypes.DefaultParams(),
				authtypes.GenesisAccounts{authtypes.NewEmptyModuleAccount(types.ModuleName, tc.permissions...)},
			)
			bep3Genesis := types.DefaultGenesisState()
			genState := app.GenesisState{
				authtypes.ModuleName: cdc.MustMarshalJSON(authGenesis),
				types.ModuleName:     cdc.MustMarshalJSON(&bep3Genesis),
			}

			initApp := func() { suite.app.InitializeFromGenesisStates(genState) }

			if tc.expectedPanic == "" {
				suite.NotPanics(initApp)
			} else {
				suite.PanicsWithValue(tc.expectedPanic, initApp)
			}
		})
	}
}

func (suite *GenesisTestSuite) TestGenesisState() {
	type GenState func() app.GenesisState

	cdc := suite.app.AppCodec()

	testCases := []struct {
		name        string
		genState    GenState
		expectPass  bool
		expectedErr interface{}
	}{
		{
			name: "default",
			genState: func() app.GenesisState {
				return NewBep3GenStateMulti(cdc, suite.addrs[0])
			},
			expectPass: true,
		},
		{
			name: "import atomic swaps and asset supplies",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				_, addrs := app.GeneratePrivKeyAddressPairs(2)
				var swaps types.AtomicSwaps
				var supplies types.AssetSupplies
				for i := 0; i < 2; i++ {
					swap, supply := loadSwapAndSupply(addrs[i], i)
					swaps = append(swaps, swap)
					supplies = append(supplies, supply)
				}
				gs.AtomicSwaps = swaps
				gs.Supplies = supplies
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass: true,
		},
		{
			name: "0 deputy fees",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				gs.Params.AssetParams[0].FixedFee = sdk.ZeroInt()
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass: true,
		},
		{
			name: "incoming supply doesn't match amount in incoming atomic swaps",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0]) // incoming supply is zero
				_, addrs := app.GeneratePrivKeyAddressPairs(1)
				swap, _ := loadSwapAndSupply(addrs[0], 0)
				gs.AtomicSwaps = types.AtomicSwaps{swap}
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "asset's incoming supply 0bnb does not match amount 50000 in incoming atomic swaps",
		},
		{
			name: "current supply above limit",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				bnbSupplyLimit := math.ZeroInt()
				for _, ap := range gs.Params.AssetParams {
					if ap.Denom == "bnb" {
						bnbSupplyLimit = ap.SupplyLimit.Limit
					}
				}
				gs.Supplies = types.AssetSupplies{
					types.NewAssetSupply(
						c("bnb", 0),
						c("bnb", 0),
						sdk.NewCoin("bnb", bnbSupplyLimit.Add(i(1))),
						c("bnb", 0),
						0,
					),
				}
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "asset's current supply 350000000000001bnb is over the supply limit 350000000000000",
		},
		{
			name: "incoming supply above limit",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				// Set up overlimit amount
				bnbSupplyLimit := math.ZeroInt()
				for _, ap := range gs.Params.AssetParams {
					if ap.Denom == "bnb" {
						bnbSupplyLimit = ap.SupplyLimit.Limit
					}
				}
				overLimitAmount := bnbSupplyLimit.Add(i(1))

				// Set up an atomic swap with amount equal to the currently asset supply
				_, addrs := app.GeneratePrivKeyAddressPairs(2)
				timestamp := ts(0)
				randomNumber, _ := types.GenerateSecureRandomNumber()
				randomNumberHash := types.CalculateRandomHash(randomNumber[:], timestamp)
				swap := types.NewAtomicSwap(cs(c("bnb", overLimitAmount.Int64())), randomNumberHash,
					types.DefaultMinBlockLock, timestamp, suite.addrs[0], addrs[1], TestSenderOtherChain,
					TestRecipientOtherChain, 0, types.SWAP_STATUS_OPEN, true, types.SWAP_DIRECTION_INCOMING)
				gs.AtomicSwaps = types.AtomicSwaps{swap}

				// Set up asset supply with overlimit current supply
				gs.Supplies = types.AssetSupplies{
					types.NewAssetSupply(
						c("bnb", overLimitAmount.Int64()),
						c("bnb", 0),
						c("bnb", 0),
						c("bnb", 0),
						0,
					),
				}
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "asset's incoming supply 350000000000001bnb is over the supply limit 350000000000000",
		},
		{
			name: "incoming supply + current supply above limit",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				// Set up overlimit amount
				bnbSupplyLimit := math.ZeroInt()
				for _, ap := range gs.Params.AssetParams {
					if ap.Denom == "bnb" {
						bnbSupplyLimit = ap.SupplyLimit.Limit
					}
				}
				halfLimit := bnbSupplyLimit.Int64() / 2
				overHalfLimit := halfLimit + 1

				// Set up an atomic swap with amount equal to the currently asset supply
				_, addrs := app.GeneratePrivKeyAddressPairs(2)
				timestamp := ts(0)
				randomNumber, _ := types.GenerateSecureRandomNumber()
				randomNumberHash := types.CalculateRandomHash(randomNumber[:], timestamp)
				swap := types.NewAtomicSwap(cs(c("bnb", halfLimit)), randomNumberHash,
					uint64(360), timestamp, suite.addrs[0], addrs[1], TestSenderOtherChain,
					TestRecipientOtherChain, 0, types.SWAP_STATUS_OPEN, true, types.SWAP_DIRECTION_INCOMING)
				gs.AtomicSwaps = types.AtomicSwaps{swap}

				// Set up asset supply with overlimit supply
				gs.Supplies = types.AssetSupplies{
					types.NewAssetSupply(
						c("bnb", halfLimit),
						c("bnb", 0),
						c("bnb", overHalfLimit),
						c("bnb", 0),
						0,
					),
				}
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "asset's incoming supply + current supply 350000000000001bnb is over the supply limit 350000000000000",
		},
		{
			name: "outgoing supply above limit",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				// Set up overlimit amount
				bnbSupplyLimit := math.ZeroInt()
				for _, ap := range gs.Params.AssetParams {
					if ap.Denom == "bnb" {
						bnbSupplyLimit = ap.SupplyLimit.Limit
					}
				}
				overLimitAmount := bnbSupplyLimit.Add(i(1))

				// Set up an atomic swap with amount equal to the currently asset supply
				_, addrs := app.GeneratePrivKeyAddressPairs(2)
				timestamp := ts(0)
				randomNumber, _ := types.GenerateSecureRandomNumber()
				randomNumberHash := types.CalculateRandomHash(randomNumber[:], timestamp)
				swap := types.NewAtomicSwap(cs(c("bnb", overLimitAmount.Int64())), randomNumberHash,
					types.DefaultMinBlockLock, timestamp, addrs[1], suite.addrs[0], TestSenderOtherChain,
					TestRecipientOtherChain, 0, types.SWAP_STATUS_OPEN, true, types.SWAP_DIRECTION_OUTGOING)
				gs.AtomicSwaps = types.AtomicSwaps{swap}

				// Set up asset supply with overlimit outgoing supply
				gs.Supplies = types.AssetSupplies{
					types.NewAssetSupply(
						c("bnb", 0),
						c("bnb", overLimitAmount.Int64()),
						c("bnb", 0),
						c("bnb", 0),
						0,
					),
				}
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "asset's outgoing supply 350000000000001bnb is over the supply limit 350000000000000",
		},
		{
			name: "asset supply denom is not a supported asset",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				gs.Supplies = types.AssetSupplies{
					types.NewAssetSupply(
						c("fake", 0),
						c("fake", 0),
						c("fake", 0),
						c("fake", 0),
						0,
					),
				}
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "asset's supply limit not found: fake: asset not found",
		},
		{
			name: "atomic swap asset type is unsupported",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				_, addrs := app.GeneratePrivKeyAddressPairs(2)
				timestamp := ts(0)
				randomNumber, _ := types.GenerateSecureRandomNumber()
				randomNumberHash := types.CalculateRandomHash(randomNumber[:], timestamp)
				swap := types.NewAtomicSwap(cs(c("fake", 500000)), randomNumberHash,
					uint64(360), timestamp, suite.addrs[0], addrs[1], TestSenderOtherChain,
					TestRecipientOtherChain, 0, types.SWAP_STATUS_OPEN, true, types.SWAP_DIRECTION_INCOMING)

				gs.AtomicSwaps = types.AtomicSwaps{swap}
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "swap has invalid asset: fake: asset not found",
		},
		{
			name: "atomic swap status is invalid",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				_, addrs := app.GeneratePrivKeyAddressPairs(2)
				timestamp := ts(0)
				randomNumber, _ := types.GenerateSecureRandomNumber()
				randomNumberHash := types.CalculateRandomHash(randomNumber[:], timestamp)
				swap := types.NewAtomicSwap(cs(c("bnb", 5000)), randomNumberHash,
					uint64(360), timestamp, suite.addrs[0], addrs[1], TestSenderOtherChain,
					TestRecipientOtherChain, 0, types.SWAP_STATUS_UNSPECIFIED, true, types.SWAP_DIRECTION_INCOMING)

				gs.AtomicSwaps = types.AtomicSwaps{swap}
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "failed to validate bep3 genesis state: invalid swap status",
		},
		{
			name: "minimum block lock cannot be > maximum block lock",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				gs.Params.AssetParams[0].MinBlockLock = 201
				gs.Params.AssetParams[0].MaxBlockLock = 200
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "failed to validate bep3 genesis state: asset bnb has minimum block lock > maximum block lock 201 > 200",
		},
		{
			name: "empty supported asset denom",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				gs.Params.AssetParams[0].Denom = ""
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "failed to validate bep3 genesis state: asset denom invalid: ",
		},
		{
			name: "negative supported asset limit",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				gs.Params.AssetParams[0].SupplyLimit.Limit = i(-100)
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "failed to validate bep3 genesis state: asset bnb has invalid (negative) supply limit: -100",
		},
		{
			name: "duplicate supported asset denom",
			genState: func() app.GenesisState {
				gs := baseGenState(suite.addrs[0])
				gs.Params.AssetParams[1].Denom = "bnb"
				return app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(&gs)}
			},
			expectPass:  false,
			expectedErr: "failed to validate bep3 genesis state: asset bnb cannot have duplicate denom",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			gs := tc.genState()
			if tc.expectPass {
				suite.NotPanics(func() {
					suite.app.InitializeFromGenesisStates(gs)
				}, tc.name)
			} else {
				suite.PanicsWithValue(tc.expectedErr, func() {
					suite.app.InitializeFromGenesisStates(gs)
				}, tc.name)
			}
		})
	}
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}
