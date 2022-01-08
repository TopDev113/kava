package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

//nolint
func i(in int64) sdk.Int { return sdk.NewInt(in) }

//nolint
func c(denom string, amount int64) sdk.Coin { return sdk.NewInt64Coin(denom, amount) }

//nolint
func cs(coins ...sdk.Coin) sdk.Coins { return sdk.NewCoins(coins...) }

// func NewAuthGenStateFromAccs(accounts ...authexported.GenesisAccount) app.GenesisState {
// 	authGenesis := auth.NewGenesisState(auth.DefaultParams(), accounts)
// 	return app.GenesisState{auth.ModuleName: auth.ModuleCdc.MustMarshalJSON(authGenesis)}
// }

// func NewSwapGenStateMulti() app.GenesisState {
// 	swapGenesis := types.GenesisState{
// 		Params: types.Params{
// 			AllowedPools: types.AllowedPools{
// 				types.NewAllowedPool("ukava", "usdx"),
// 			},
// 			SwapFee: sdk.MustNewDecFromStr("0.03"),
// 		},
// 	}

// 	return app.GenesisState{types.ModuleName: types.ModuleCdc.MustMarshalJSON(swapGenesis)}
// }
