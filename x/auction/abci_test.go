package auction_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/supply"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/auction"
	"github.com/kava-labs/kava/x/liquidator"
)

func TestKeeper_EndBlocker(t *testing.T) {
	// Setup
	_, addrs := app.GeneratePrivKeyAddressPairs(2)
	buyer := addrs[0]
	returnAddrs := addrs[1:]
	returnWeights := []sdk.Int{sdk.NewInt(1)}
	sellerModName := liquidator.ModuleName
	//sellerAddr := supply.NewModuleAddress(sellerModName)

	tApp := app.NewTestApp()
	sellerAcc := supply.NewEmptyModuleAccount(sellerModName)
	require.NoError(t, sellerAcc.SetCoins(cs(c("token1", 100), c("token2", 100))))
	tApp.InitializeFromGenesisStates(
		NewAuthGenStateFromAccs(authexported.GenesisAccounts{
			auth.NewBaseAccount(buyer, cs(c("token1", 100), c("token2", 100)), nil, 0, 0),
			sellerAcc,
		}),
	)

	ctx := tApp.NewContext(true, abci.Header{})
	keeper := tApp.GetAuctionKeeper()

	auctionID, err := keeper.StartForwardReverseAuction(ctx, sellerModName, c("token1", 20), c("token2", 50), returnAddrs, returnWeights)
	require.NoError(t, err)
	require.NoError(t, keeper.PlaceBid(ctx, auctionID, buyer, c("token2", 30), c("token1", 20)))

	// Run the endblocker, simulating a block time 1ns before auction expiry
	preExpiryTime := ctx.BlockTime().Add(auction.DefaultBidDuration - 1)
	auction.EndBlocker(ctx.WithBlockTime(preExpiryTime), keeper)

	// Check auction has not been closed yet
	_, found := keeper.GetAuction(ctx, auctionID)
	require.True(t, found)

	// Run the endblocker, simulating a block time equal to auction expiry
	expiryTime := ctx.BlockTime().Add(auction.DefaultBidDuration)
	auction.EndBlocker(ctx.WithBlockTime(expiryTime), keeper)

	// Check auction has been closed
	_, found = keeper.GetAuction(ctx, auctionID)
	require.False(t, found)
}

func c(denom string, amount int64) sdk.Coin { return sdk.NewInt64Coin(denom, amount) }
func cs(coins ...sdk.Coin) sdk.Coins        { return sdk.NewCoins(coins...) }

func NewAuthGenStateFromAccs(accounts authexported.GenesisAccounts) app.GenesisState {
	authGenesis := auth.NewGenesisState(auth.DefaultParams(), accounts)
	return app.GenesisState{auth.ModuleName: auth.ModuleCdc.MustMarshalJSON(authGenesis)}
}
