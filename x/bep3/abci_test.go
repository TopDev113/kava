package bep3_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/bep3"
	"github.com/kava-labs/kava/x/bep3/keeper"
	"github.com/kava-labs/kava/x/bep3/types"
)

type ABCITestSuite struct {
	suite.Suite
	keeper        keeper.Keeper
	app           app.TestApp
	ctx           sdk.Context
	addrs         []sdk.AccAddress
	swapIDs       []tmbytes.HexBytes
	randomNumbers []tmbytes.HexBytes
}

func (suite *ABCITestSuite) SetupTest() {
	tApp := app.NewTestApp()
	ctx := tApp.NewContext(true, tmproto.Header{Height: 1, Time: tmtime.Now()})

	// Set up auth GenesisState
	_, addrs := app.GeneratePrivKeyAddressPairs(12)
	coins := sdk.NewCoins(c("bnb", 10000000000), c("ukava", 10000000000))
	authGS := app.NewFundedGenStateWithSameCoins(tApp.AppCodec(), coins, addrs)
	// Initialize test app
	tApp.InitializeFromGenesisStates(authGS, NewBep3GenStateMulti(tApp.AppCodec(), addrs[11]))

	suite.ctx = ctx
	suite.app = tApp
	suite.addrs = addrs
	suite.ResetKeeper()
}

func (suite *ABCITestSuite) ResetKeeper() {
	suite.keeper = suite.app.GetBep3Keeper()

	var swapIDs []tmbytes.HexBytes
	var randomNumbers []tmbytes.HexBytes
	for i := 0; i < 10; i++ {
		// Set up atomic swap variables
		expireHeight := types.DefaultMinBlockLock
		amount := cs(c("bnb", int64(10000)))
		timestamp := ts(i)
		randomNumber, _ := types.GenerateSecureRandomNumber()
		randomNumberHash := types.CalculateRandomHash(randomNumber[:], timestamp)

		// Create atomic swap and check err to confirm creation
		err := suite.keeper.CreateAtomicSwap(suite.ctx, randomNumberHash, timestamp, expireHeight,
			suite.addrs[11], suite.addrs[i], TestSenderOtherChain, TestRecipientOtherChain,
			amount, true)
		suite.Nil(err)

		// Store swap's calculated ID and secret random number
		swapID := types.CalculateSwapID(randomNumberHash, suite.addrs[11], TestSenderOtherChain)
		swapIDs = append(swapIDs, swapID)
		randomNumbers = append(randomNumbers, randomNumber[:])
	}
	suite.swapIDs = swapIDs
	suite.randomNumbers = randomNumbers
}

func (suite *ABCITestSuite) TestBeginBlocker_UpdateExpiredAtomicSwaps() {
	testCases := []struct {
		name            string
		firstCtx        sdk.Context
		secondCtx       sdk.Context
		expectedStatus  types.SwapStatus
		expectInStorage bool
	}{
		{
			name:            "normal",
			firstCtx:        suite.ctx,
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 10),
			expectedStatus:  types.SWAP_STATUS_OPEN,
			expectInStorage: true,
		},
		{
			name:            "after expiration",
			firstCtx:        suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 400),
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 410),
			expectedStatus:  types.SWAP_STATUS_EXPIRED,
			expectInStorage: true,
		},
		{
			name:            "after completion",
			firstCtx:        suite.ctx,
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 10),
			expectedStatus:  types.SWAP_STATUS_COMPLETED,
			expectInStorage: true,
		},
		{
			name:            "after deletion",
			firstCtx:        suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 400),
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 400 + int64(types.DefaultLongtermStorageDuration)),
			expectedStatus:  types.SWAP_STATUS_UNSPECIFIED,
			expectInStorage: false,
		},
	}

	for _, tc := range testCases {
		// Reset keeper and run the initial begin blocker
		suite.ResetKeeper()
		suite.Run(tc.name, func() {
			bep3.BeginBlocker(tc.firstCtx, suite.keeper)

			switch tc.expectedStatus {
			case types.SWAP_STATUS_COMPLETED:
				for i, swapID := range suite.swapIDs {
					err := suite.keeper.ClaimAtomicSwap(tc.firstCtx, suite.addrs[5], swapID, suite.randomNumbers[i])
					suite.Nil(err)
				}
			case types.SWAP_STATUS_UNSPECIFIED:
				for _, swapID := range suite.swapIDs {
					err := suite.keeper.RefundAtomicSwap(tc.firstCtx, suite.addrs[5], swapID)
					suite.Nil(err)
				}
			}

			// Run the second begin blocker
			bep3.BeginBlocker(tc.secondCtx, suite.keeper)

			// Check each swap's availibility and status
			for _, swapID := range suite.swapIDs {
				storedSwap, found := suite.keeper.GetAtomicSwap(tc.secondCtx, swapID)
				if tc.expectInStorage {
					suite.True(found)
				} else {
					suite.False(found)
				}
				suite.Equal(tc.expectedStatus, storedSwap.Status)
			}
		})
	}
}

func (suite *ABCITestSuite) TestBeginBlocker_DeleteClosedAtomicSwapsFromLongtermStorage() {
	type Action int
	const (
		NULL   Action = 0x00
		Refund Action = 0x01
		Claim  Action = 0x02
	)

	testCases := []struct {
		name            string
		firstCtx        sdk.Context
		action          Action
		secondCtx       sdk.Context
		expectInStorage bool
	}{
		{
			name:            "no action with long storage duration",
			firstCtx:        suite.ctx,
			action:          NULL,
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + int64(types.DefaultLongtermStorageDuration)),
			expectInStorage: true,
		},
		{
			name:            "claim with short storage duration",
			firstCtx:        suite.ctx,
			action:          Claim,
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 5000),
			expectInStorage: true,
		},
		{
			name:            "claim with long storage duration",
			firstCtx:        suite.ctx,
			action:          Claim,
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + int64(types.DefaultLongtermStorageDuration)),
			expectInStorage: false,
		},
		{
			name:            "refund with short storage duration",
			firstCtx:        suite.ctx,
			action:          Refund,
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 5000),
			expectInStorage: true,
		},
		{
			name:            "refund with long storage duration",
			firstCtx:        suite.ctx,
			action:          Refund,
			secondCtx:       suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + int64(types.DefaultLongtermStorageDuration)),
			expectInStorage: false,
		},
	}

	for _, tc := range testCases {
		// Reset keeper and run the initial begin blocker
		suite.ResetKeeper()
		suite.Run(tc.name, func() {
			bep3.BeginBlocker(tc.firstCtx, suite.keeper)

			switch tc.action {
			case Claim:
				for i, swapID := range suite.swapIDs {
					err := suite.keeper.ClaimAtomicSwap(tc.firstCtx, suite.addrs[5], swapID, suite.randomNumbers[i])
					suite.Nil(err)
				}
			case Refund:
				for _, swapID := range suite.swapIDs {
					swap, _ := suite.keeper.GetAtomicSwap(tc.firstCtx, swapID)
					refundCtx := suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + int64(swap.ExpireHeight))
					bep3.BeginBlocker(refundCtx, suite.keeper)
					err := suite.keeper.RefundAtomicSwap(refundCtx, suite.addrs[5], swapID)
					suite.Nil(err)
					// Add expire height to second ctx block height
					tc.secondCtx = tc.secondCtx.WithBlockHeight(tc.secondCtx.BlockHeight() + int64(swap.ExpireHeight))
				}
			}

			// Run the second begin blocker
			bep3.BeginBlocker(tc.secondCtx, suite.keeper)

			// Check each swap's availability and status
			for _, swapID := range suite.swapIDs {
				_, found := suite.keeper.GetAtomicSwap(tc.secondCtx, swapID)
				if tc.expectInStorage {
					suite.True(found)
				} else {
					suite.False(found)
				}
			}
		})
	}
}

func TestABCITestSuite(t *testing.T) {
	suite.Run(t, new(ABCITestSuite))
}
