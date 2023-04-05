package simulation

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/cosmos/cosmos-sdk/x/supply"

	"github.com/kava-labs/kava/x/bep3/types"
)

var (
	MaxSupplyLimit     = 1000000000000
	MinSupplyLimit     = 100000000
	MinSwapAmountLimit = 999
	accs               []simulation.Account
	ConsistentDenoms   = [3]string{"bnb", "xrp", "btc"}
	MinBlockLock       = uint64(5)
)

// GenRandBnbDeputy randomized BnbDeputyAddress
func GenRandBnbDeputy(r *rand.Rand) simulation.Account {
	acc, _ := simulation.RandomAcc(r, accs)
	return acc
}

// GenRandFixedFee randomized FixedFee in range [1, 10000]
func GenRandFixedFee(r *rand.Rand) sdkmath.Int {
	min := int(1)
	max := types.DefaultBnbDeputyFixedFee.Int64()
	return sdkmath.NewInt(int64(r.Intn(int(max)-min) + min))
}

// GenMinSwapAmount randomized MinAmount in range [1, 1000]
func GenMinSwapAmount(r *rand.Rand) sdkmath.Int {
	return sdk.OneInt().Add(simulation.RandomAmount(r, sdkmath.NewInt(int64(MinSwapAmountLimit))))
}

// GenMaxSwapAmount randomized MaxAmount
func GenMaxSwapAmount(r *rand.Rand, minAmount sdkmath.Int, supplyMax sdkmath.Int) sdkmath.Int {
	min := minAmount.Int64()
	max := supplyMax.Quo(sdkmath.NewInt(100)).Int64()

	return sdkmath.NewInt((int64(r.Intn(int(max-min))) + min))
}

// GenSupplyLimit generates a random SupplyLimit
func GenSupplyLimit(r *rand.Rand, max int) sdkmath.Int {
	max = simulation.RandIntBetween(r, MinSupplyLimit, max)
	return sdkmath.NewInt(int64(max))
}

// GenSupplyLimit generates a random SupplyLimit
func GenAssetSupply(r *rand.Rand, denom string) types.AssetSupply {
	return types.NewAssetSupply(
		sdk.NewCoin(denom, sdk.ZeroInt()), sdk.NewCoin(denom, sdk.ZeroInt()),
		sdk.NewCoin(denom, sdk.ZeroInt()), sdk.NewCoin(denom, sdk.ZeroInt()), time.Duration(0))
}

// GenMinBlockLock randomized MinBlockLock
func GenMinBlockLock(r *rand.Rand) uint64 {
	return MinBlockLock
}

// GenMaxBlockLock randomized MaxBlockLock
func GenMaxBlockLock(r *rand.Rand, minBlockLock uint64) uint64 {
	max := int(50)
	return uint64(r.Intn(max-int(MinBlockLock)) + int(MinBlockLock+1))
}

// GenSupportedAssets gets randomized SupportedAssets
func GenSupportedAssets(r *rand.Rand) types.AssetParams {
	numAssets := (r.Intn(10) + 1)
	assets := make(types.AssetParams, numAssets+1)
	for i := 0; i < numAssets; i++ {
		denom := strings.ToLower(simulation.RandStringOfLength(r, (r.Intn(3) + 3)))
		asset := genSupportedAsset(r, denom)
		assets[i] = asset
	}
	// Add bnb, btc, or xrp as a supported asset for interactions with other modules
	assets[len(assets)-1] = genSupportedAsset(r, ConsistentDenoms[r.Intn(3)])

	return assets
}

func genSupportedAsset(r *rand.Rand, denom string) types.AssetParam {
	coinID, _ := simulation.RandPositiveInt(r, sdkmath.NewInt(100000))
	limit := GenSupplyLimit(r, MaxSupplyLimit)

	minSwapAmount := GenMinSwapAmount(r)
	minBlockLock := GenMinBlockLock(r)
	timeLimited := r.Float32() < 0.5
	timeBasedLimit := sdk.ZeroInt()
	if timeLimited {
		// set time-based limit to between 10 and 25% of the total limit
		min := int(limit.Quo(sdkmath.NewInt(10)).Int64())
		max := int(limit.Quo(sdkmath.NewInt(4)).Int64())
		timeBasedLimit = sdkmath.NewInt(int64(simulation.RandIntBetween(r, min, max)))
	}
	return types.AssetParam{
		Denom:  denom,
		CoinID: int(coinID.Int64()),
		SupplyLimit: types.SupplyLimit{
			Limit:          limit,
			TimeLimited:    timeLimited,
			TimePeriod:     time.Hour * 24,
			TimeBasedLimit: timeBasedLimit,
		},
		Active:        true,
		DeputyAddress: GenRandBnbDeputy(r).Address,
		FixedFee:      GenRandFixedFee(r),
		MinSwapAmount: minSwapAmount,
		MaxSwapAmount: GenMaxSwapAmount(r, minSwapAmount, limit),
		MinBlockLock:  minBlockLock,
		MaxBlockLock:  GenMaxBlockLock(r, minBlockLock),
	}
}

// RandomizedGenState generates a random GenesisState
func RandomizedGenState(simState *module.SimulationState) {
	accs = simState.Accounts

	bep3Genesis := loadRandomBep3GenState(simState)
	fmt.Printf("Selected randomly generated %s parameters:\n%s\n", types.ModuleName, codec.MustMarshalJSONIndent(simState.Cdc, bep3Genesis))
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(bep3Genesis)

	authGenesis, totalCoins := loadAuthGenState(simState, bep3Genesis)
	simState.GenState[auth.ModuleName] = simState.Cdc.MustMarshalJSON(authGenesis)

	// Update supply to match amount of coins in auth
	var supplyGenesis supply.GenesisState
	simState.Cdc.MustUnmarshalJSON(simState.GenState[supply.ModuleName], &supplyGenesis)

	for _, deputyCoin := range totalCoins {
		supplyGenesis.Supply = supplyGenesis.Supply.Add(deputyCoin...)
	}
	simState.GenState[supply.ModuleName] = simState.Cdc.MustMarshalJSON(supplyGenesis)
}

func loadRandomBep3GenState(simState *module.SimulationState) types.GenesisState {
	supportedAssets := GenSupportedAssets(simState.Rand)
	supplies := types.AssetSupplies{}
	for _, asset := range supportedAssets {
		supply := GenAssetSupply(simState.Rand, asset.Denom)
		supplies = append(supplies, supply)
	}

	bep3Genesis := types.GenesisState{
		Params: types.Params{
			AssetParams: supportedAssets,
		},
		Supplies:          supplies,
		PreviousBlockTime: types.DefaultPreviousBlockTime,
	}

	return bep3Genesis
}

func loadAuthGenState(simState *module.SimulationState, bep3Genesis types.GenesisState) (auth.GenesisState, []sdk.Coins) {
	var authGenesis auth.GenesisState
	simState.Cdc.MustUnmarshalJSON(simState.GenState[auth.ModuleName], &authGenesis)
	// Load total limit of each supported asset to deputy's account
	var totalCoins []sdk.Coins
	for _, asset := range bep3Genesis.Params.AssetParams {
		deputy, found := getAccount(authGenesis.Accounts, asset.DeputyAddress)
		if !found {
			panic("deputy address not found in available accounts")
		}
		assetCoin := sdk.NewCoins(sdk.NewCoin(asset.Denom, asset.SupplyLimit.Limit))
		if err := deputy.SetCoins(deputy.GetCoins().Add(assetCoin...)); err != nil {
			panic(err)
		}
		totalCoins = append(totalCoins, assetCoin)
		authGenesis.Accounts = replaceOrAppendAccount(authGenesis.Accounts, deputy)
	}

	return authGenesis, totalCoins
}

// Return an account from a list of accounts that matches an address.
func getAccount(accounts []authexported.GenesisAccount, addr sdk.AccAddress) (authexported.GenesisAccount, bool) {
	for _, a := range accounts {
		if a.GetAddress().Equals(addr) {
			return a, true
		}
	}
	return nil, false
}

// In a list of accounts, replace the first account found with the same address. If not found, append the account.
func replaceOrAppendAccount(accounts []authexported.GenesisAccount, acc authexported.GenesisAccount) []authexported.GenesisAccount {
	newAccounts := accounts
	for i, a := range accounts {
		if a.GetAddress().Equals(acc.GetAddress()) {
			newAccounts[i] = acc
			return newAccounts
		}
	}
	return append(newAccounts, acc)
}
