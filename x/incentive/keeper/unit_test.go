package keeper_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/libs/log"
	db "github.com/tendermint/tm-db"

	"github.com/kava-labs/kava/app"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	earntypes "github.com/kava-labs/kava/x/earn/types"
	tmprototypes "github.com/tendermint/tendermint/proto/tendermint/types"

	hardtypes "github.com/kava-labs/kava/x/hard/types"
	"github.com/kava-labs/kava/x/incentive/keeper"
	"github.com/kava-labs/kava/x/incentive/types"
)

// NewTestContext sets up a basic context with an in-memory db
func NewTestContext(requiredStoreKeys ...sdk.StoreKey) sdk.Context {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)

	for _, key := range requiredStoreKeys {
		cms.MountStoreWithDB(key, sdk.StoreTypeIAVL, nil)
	}

	if err := cms.LoadLatestVersion(); err != nil {
		panic(err)
	}

	return sdk.NewContext(cms, tmprototypes.Header{}, false, log.NewNopLogger())
}

// unitTester is a wrapper around suite.Suite, with common functionality for keeper unit tests.
// It can be embedded in structs the same way as suite.Suite.
type unitTester struct {
	suite.Suite
	keeper keeper.Keeper
	ctx    sdk.Context

	cdc               codec.Codec
	incentiveStoreKey sdk.StoreKey
}

func (suite *unitTester) SetupSuite() {
	tApp := app.NewTestApp()
	suite.cdc = tApp.AppCodec()

	suite.incentiveStoreKey = sdk.NewKVStoreKey(types.StoreKey)
}

func (suite *unitTester) SetupTest() {
	suite.ctx = NewTestContext(suite.incentiveStoreKey)
	suite.keeper = suite.NewKeeper(&fakeParamSubspace{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func (suite *unitTester) TearDownTest() {
	suite.keeper = keeper.Keeper{}
	suite.ctx = sdk.Context{}
}

func (suite *unitTester) NewKeeper(
	paramSubspace types.ParamSubspace,
	bk types.BankKeeper, cdpk types.CdpKeeper, hk types.HardKeeper,
	ak types.AccountKeeper, stk types.StakingKeeper, swk types.SwapKeeper,
	svk types.SavingsKeeper, lqk types.LiquidKeeper, ek types.EarnKeeper,
) keeper.Keeper {
	return keeper.NewKeeper(
		suite.cdc, suite.incentiveStoreKey, paramSubspace,
		bk, cdpk, hk, ak, stk, swk, svk, lqk, ek,
		nil, nil, nil,
	)
}

func (suite *unitTester) storeGlobalBorrowIndexes(indexes types.MultiRewardIndexes) {
	for _, i := range indexes {
		suite.keeper.SetHardBorrowRewardIndexes(suite.ctx, i.CollateralType, i.RewardIndexes)
	}
}

func (suite *unitTester) storeGlobalSupplyIndexes(indexes types.MultiRewardIndexes) {
	for _, i := range indexes {
		suite.keeper.SetHardSupplyRewardIndexes(suite.ctx, i.CollateralType, i.RewardIndexes)
	}
}

func (suite *unitTester) storeGlobalDelegatorIndexes(multiRewardIndexes types.MultiRewardIndexes) {
	// Hardcoded to use bond denom
	multiRewardIndex, _ := multiRewardIndexes.GetRewardIndex(types.BondDenom)
	suite.keeper.SetDelegatorRewardIndexes(suite.ctx, types.BondDenom, multiRewardIndex.RewardIndexes)
}

func (suite *unitTester) storeGlobalSwapIndexes(indexes types.MultiRewardIndexes) {
	for _, i := range indexes {
		suite.keeper.SetSwapRewardIndexes(suite.ctx, i.CollateralType, i.RewardIndexes)
	}
}

func (suite *unitTester) storeGlobalSavingsIndexes(indexes types.MultiRewardIndexes) {
	for _, i := range indexes {
		suite.keeper.SetSavingsRewardIndexes(suite.ctx, i.CollateralType, i.RewardIndexes)
	}
}

func (suite *unitTester) storeGlobalEarnIndexes(indexes types.MultiRewardIndexes) {
	for _, i := range indexes {
		suite.keeper.SetEarnRewardIndexes(suite.ctx, i.CollateralType, i.RewardIndexes)
	}
}

func (suite *unitTester) storeHardClaim(claim types.HardLiquidityProviderClaim) {
	suite.keeper.SetHardLiquidityProviderClaim(suite.ctx, claim)
}

func (suite *unitTester) storeDelegatorClaim(claim types.DelegatorClaim) {
	suite.keeper.SetDelegatorClaim(suite.ctx, claim)
}

func (suite *unitTester) storeSwapClaim(claim types.SwapClaim) {
	suite.keeper.SetSwapClaim(suite.ctx, claim)
}

func (suite *unitTester) storeSavingsClaim(claim types.SavingsClaim) {
	suite.keeper.SetSavingsClaim(suite.ctx, claim)
}

func (suite *unitTester) storeEarnClaim(claim types.EarnClaim) {
	suite.keeper.SetEarnClaim(suite.ctx, claim)
}

type TestKeeperBuilder struct {
	cdc           codec.Codec
	key           sdk.StoreKey
	paramSubspace types.ParamSubspace
	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	cdpKeeper     types.CdpKeeper
	hardKeeper    types.HardKeeper
	stakingKeeper types.StakingKeeper
	swapKeeper    types.SwapKeeper
	savingsKeeper types.SavingsKeeper
	liquidKeeper  types.LiquidKeeper
	earnKeeper    types.EarnKeeper

	// Keepers used for APY queries
	mintKeeper      types.MintKeeper
	distrKeeper     types.DistrKeeper
	pricefeedKeeper types.PricefeedKeeper
}

func (suite *unitTester) NewTestKeeper(
	paramSubspace types.ParamSubspace,
) *TestKeeperBuilder {
	if !paramSubspace.HasKeyTable() {
		paramSubspace = paramSubspace.WithKeyTable(types.ParamKeyTable())
	}

	return &TestKeeperBuilder{
		cdc:             suite.cdc,
		key:             suite.incentiveStoreKey,
		paramSubspace:   paramSubspace,
		accountKeeper:   nil,
		bankKeeper:      nil,
		cdpKeeper:       nil,
		hardKeeper:      nil,
		stakingKeeper:   nil,
		swapKeeper:      nil,
		savingsKeeper:   nil,
		liquidKeeper:    nil,
		earnKeeper:      nil,
		mintKeeper:      nil,
		distrKeeper:     nil,
		pricefeedKeeper: nil,
	}
}

func (tk *TestKeeperBuilder) WithPricefeedKeeper(k types.PricefeedKeeper) *TestKeeperBuilder {
	tk.pricefeedKeeper = k
	return tk
}

func (tk *TestKeeperBuilder) WithDistrKeeper(k types.DistrKeeper) *TestKeeperBuilder {
	tk.distrKeeper = k
	return tk
}

func (tk *TestKeeperBuilder) WithBankKeeper(k types.BankKeeper) *TestKeeperBuilder {
	tk.bankKeeper = k
	return tk
}

func (tk *TestKeeperBuilder) WithStakingKeeper(k types.StakingKeeper) *TestKeeperBuilder {
	tk.stakingKeeper = k
	return tk
}

func (tk *TestKeeperBuilder) WithMintKeeper(k types.MintKeeper) *TestKeeperBuilder {
	tk.mintKeeper = k
	return tk
}

func (tk *TestKeeperBuilder) WithEarnKeeper(k types.EarnKeeper) *TestKeeperBuilder {
	tk.earnKeeper = k
	return tk
}

func (tk *TestKeeperBuilder) WithLiquidKeeper(k types.LiquidKeeper) *TestKeeperBuilder {
	tk.liquidKeeper = k
	return tk
}

func (tk *TestKeeperBuilder) Build() keeper.Keeper {
	return keeper.NewKeeper(
		tk.cdc, tk.key, tk.paramSubspace,
		tk.bankKeeper, tk.cdpKeeper, tk.hardKeeper, tk.accountKeeper,
		tk.stakingKeeper, tk.swapKeeper, tk.savingsKeeper, tk.liquidKeeper,
		tk.earnKeeper, tk.mintKeeper, tk.distrKeeper, tk.pricefeedKeeper,
	)
}

// fakeParamSubspace is a stub paramSpace to simplify keeper unit test setup.
type fakeParamSubspace struct {
	params types.Params
}

func (subspace *fakeParamSubspace) GetParamSet(_ sdk.Context, ps paramtypes.ParamSet) {
	*(ps.(*types.Params)) = subspace.params
}

func (subspace *fakeParamSubspace) SetParamSet(_ sdk.Context, ps paramtypes.ParamSet) {
	subspace.params = *(ps.(*types.Params))
}

func (subspace *fakeParamSubspace) HasKeyTable() bool {
	// return true so the keeper does not try to call WithKeyTable, which does nothing
	return true
}

func (subspace *fakeParamSubspace) WithKeyTable(paramtypes.KeyTable) paramtypes.Subspace {
	// return an non-functional subspace to satisfy the interface
	return paramtypes.Subspace{}
}

// fakeSwapKeeper is a stub swap keeper.
// It can be used to return values to the incentive keeper without having to initialize a full swap keeper.
type fakeSwapKeeper struct {
	poolShares    map[string]sdk.Int
	depositShares map[string](map[string]sdk.Int)
}

var _ types.SwapKeeper = newFakeSwapKeeper()

func newFakeSwapKeeper() *fakeSwapKeeper {
	return &fakeSwapKeeper{
		poolShares:    map[string]sdk.Int{},
		depositShares: map[string](map[string]sdk.Int){},
	}
}

func (k *fakeSwapKeeper) addPool(id string, shares sdk.Int) *fakeSwapKeeper {
	k.poolShares[id] = shares
	return k
}

func (k *fakeSwapKeeper) addDeposit(poolID string, depositor sdk.AccAddress, shares sdk.Int) *fakeSwapKeeper {
	if k.depositShares[poolID] == nil {
		k.depositShares[poolID] = map[string]sdk.Int{}
	}
	k.depositShares[poolID][depositor.String()] = shares
	return k
}

func (k *fakeSwapKeeper) GetPoolShares(_ sdk.Context, poolID string) (sdk.Int, bool) {
	shares, ok := k.poolShares[poolID]
	return shares, ok
}

func (k *fakeSwapKeeper) GetDepositorSharesAmount(_ sdk.Context, depositor sdk.AccAddress, poolID string) (sdk.Int, bool) {
	shares, found := k.depositShares[poolID][depositor.String()]
	return shares, found
}

// fakeHardKeeper is a stub hard keeper.
// It can be used to return values to the incentive keeper without having to initialize a full hard keeper.
type fakeHardKeeper struct {
	borrows  fakeHardState
	deposits fakeHardState
}

type fakeHardState struct {
	total           sdk.Coins
	interestFactors map[string]sdk.Dec
}

func newFakeHardState() fakeHardState {
	return fakeHardState{
		total:           nil,
		interestFactors: map[string]sdk.Dec{}, // initialize map to avoid panics on read
	}
}

var _ types.HardKeeper = newFakeHardKeeper()

func newFakeHardKeeper() *fakeHardKeeper {
	return &fakeHardKeeper{
		borrows:  newFakeHardState(),
		deposits: newFakeHardState(),
	}
}

func (k *fakeHardKeeper) addTotalBorrow(coin sdk.Coin, factor sdk.Dec) *fakeHardKeeper {
	k.borrows.total = k.borrows.total.Add(coin)
	k.borrows.interestFactors[coin.Denom] = factor
	return k
}

func (k *fakeHardKeeper) addTotalSupply(coin sdk.Coin, factor sdk.Dec) *fakeHardKeeper {
	k.deposits.total = k.deposits.total.Add(coin)
	k.deposits.interestFactors[coin.Denom] = factor
	return k
}

func (k *fakeHardKeeper) GetBorrowedCoins(_ sdk.Context) (sdk.Coins, bool) {
	if k.borrows.total == nil {
		return nil, false
	}
	return k.borrows.total, true
}

func (k *fakeHardKeeper) GetSuppliedCoins(_ sdk.Context) (sdk.Coins, bool) {
	if k.deposits.total == nil {
		return nil, false
	}
	return k.deposits.total, true
}

func (k *fakeHardKeeper) GetBorrowInterestFactor(_ sdk.Context, denom string) (sdk.Dec, bool) {
	f, ok := k.borrows.interestFactors[denom]
	return f, ok
}

func (k *fakeHardKeeper) GetSupplyInterestFactor(_ sdk.Context, denom string) (sdk.Dec, bool) {
	f, ok := k.deposits.interestFactors[denom]
	return f, ok
}

func (k *fakeHardKeeper) GetBorrow(_ sdk.Context, _ sdk.AccAddress) (hardtypes.Borrow, bool) {
	panic("unimplemented")
}

func (k *fakeHardKeeper) GetDeposit(_ sdk.Context, _ sdk.AccAddress) (hardtypes.Deposit, bool) {
	panic("unimplemented")
}

// fakeStakingKeeper is a stub staking keeper.
// It can be used to return values to the incentive keeper without having to initialize a full staking keeper.
type fakeStakingKeeper struct {
	delegations stakingtypes.Delegations
	validators  stakingtypes.Validators
}

var _ types.StakingKeeper = newFakeStakingKeeper()

func newFakeStakingKeeper() *fakeStakingKeeper { return &fakeStakingKeeper{} }

func (k *fakeStakingKeeper) addBondedTokens(amount int64) *fakeStakingKeeper {
	if len(k.validators) != 0 {
		panic("cannot set total bonded if keeper already has validators set")
	}
	// add a validator with all the tokens
	k.validators = append(k.validators, stakingtypes.Validator{
		Status: stakingtypes.Bonded,
		Tokens: sdk.NewInt(amount),
	})
	return k
}

func (k *fakeStakingKeeper) TotalBondedTokens(_ sdk.Context) sdk.Int {
	total := sdk.ZeroInt()
	for _, val := range k.validators {
		if val.GetStatus() == stakingtypes.Bonded {
			total = total.Add(val.GetBondedTokens())
		}
	}
	return total
}

func (k *fakeStakingKeeper) GetDelegatorDelegations(_ sdk.Context, delegator sdk.AccAddress, maxRetrieve uint16) []stakingtypes.Delegation {
	return k.delegations
}

func (k *fakeStakingKeeper) GetValidator(_ sdk.Context, addr sdk.ValAddress) (stakingtypes.Validator, bool) {
	for _, val := range k.validators {
		if val.GetOperator().Equals(addr) {
			return val, true
		}
	}
	return stakingtypes.Validator{}, false
}

func (k *fakeStakingKeeper) GetValidatorDelegations(_ sdk.Context, valAddr sdk.ValAddress) []stakingtypes.Delegation {
	var delegations stakingtypes.Delegations
	for _, d := range k.delegations {
		if d.GetValidatorAddr().Equals(valAddr) {
			delegations = append(delegations, d)
		}
	}
	return delegations
}

// fakeCDPKeeper is a stub cdp keeper.
// It can be used to return values to the incentive keeper without having to initialize a full cdp keeper.
type fakeCDPKeeper struct {
	interestFactor *sdk.Dec
	totalPrincipal sdk.Int
}

var _ types.CdpKeeper = newFakeCDPKeeper()

func newFakeCDPKeeper() *fakeCDPKeeper {
	return &fakeCDPKeeper{
		interestFactor: nil,
		totalPrincipal: sdk.ZeroInt(),
	}
}

func (k *fakeCDPKeeper) addInterestFactor(f sdk.Dec) *fakeCDPKeeper {
	k.interestFactor = &f
	return k
}

func (k *fakeCDPKeeper) addTotalPrincipal(p sdk.Int) *fakeCDPKeeper {
	k.totalPrincipal = p
	return k
}

func (k *fakeCDPKeeper) GetInterestFactor(_ sdk.Context, collateralType string) (sdk.Dec, bool) {
	if k.interestFactor != nil {
		return *k.interestFactor, true
	}
	return sdk.Dec{}, false
}

func (k *fakeCDPKeeper) GetTotalPrincipal(_ sdk.Context, collateralType string, principalDenom string) sdk.Int {
	return k.totalPrincipal
}

func (k *fakeCDPKeeper) GetCdpByOwnerAndCollateralType(_ sdk.Context, owner sdk.AccAddress, collateralType string) (cdptypes.CDP, bool) {
	return cdptypes.CDP{}, false
}

func (k *fakeCDPKeeper) GetCollateral(_ sdk.Context, collateralType string) (cdptypes.CollateralParam, bool) {
	return cdptypes.CollateralParam{}, false
}

// fakeEarnKeeper is a stub earn keeper.
// It can be used to return values to the incentive keeper without having to initialize a full earn keeper.
type fakeEarnKeeper struct {
	vaultShares   map[string]earntypes.VaultShare
	depositShares map[string]earntypes.VaultShares
}

var _ types.EarnKeeper = newFakeEarnKeeper()

func newFakeEarnKeeper() *fakeEarnKeeper {
	return &fakeEarnKeeper{
		vaultShares:   map[string]earntypes.VaultShare{},
		depositShares: map[string]earntypes.VaultShares{},
	}
}

func (k *fakeEarnKeeper) addVault(vaultDenom string, shares earntypes.VaultShare) *fakeEarnKeeper {
	k.vaultShares[vaultDenom] = shares
	return k
}

func (k *fakeEarnKeeper) addDeposit(
	depositor sdk.AccAddress,
	shares earntypes.VaultShare,
) *fakeEarnKeeper {
	if k.depositShares[depositor.String()] == nil {
		k.depositShares[depositor.String()] = earntypes.NewVaultShares()
	}

	k.depositShares[depositor.String()] = k.depositShares[depositor.String()].Add(shares)

	return k
}

func (k *fakeEarnKeeper) GetVaultTotalShares(
	ctx sdk.Context,
	denom string,
) (shares earntypes.VaultShare, found bool) {
	vaultShares, found := k.vaultShares[denom]
	return vaultShares, found
}

func (k *fakeEarnKeeper) GetVaultTotalValue(ctx sdk.Context, denom string) (sdk.Coin, error) {
	vaultShares, found := k.vaultShares[denom]
	if !found {
		return sdk.NewCoin(denom, sdk.ZeroInt()), nil
	}

	return sdk.NewCoin(denom, vaultShares.Amount.RoundInt()), nil
}

func (k *fakeEarnKeeper) GetVaultAccountShares(
	ctx sdk.Context,
	acc sdk.AccAddress,
) (shares earntypes.VaultShares, found bool) {
	accShares, found := k.depositShares[acc.String()]
	return accShares, found
}

func (k *fakeEarnKeeper) IterateVaultRecords(
	ctx sdk.Context,
	cb func(record earntypes.VaultRecord) (stop bool),
) {
	for _, vaultShares := range k.vaultShares {
		cb(earntypes.VaultRecord{
			TotalShares: vaultShares,
		})
	}
}

// fakeLiquidKeeper is a stub liquid keeper.
// It can be used to return values to the incentive keeper without having to initialize a full liquid keeper.
type fakeLiquidKeeper struct {
	derivatives     map[string]sdk.Int
	lastRewardClaim map[string]time.Time
}

var _ types.LiquidKeeper = newFakeLiquidKeeper()

func newFakeLiquidKeeper() *fakeLiquidKeeper {
	return &fakeLiquidKeeper{
		derivatives:     map[string]sdk.Int{},
		lastRewardClaim: map[string]time.Time{},
	}
}

func (k *fakeLiquidKeeper) addDerivative(
	ctx sdk.Context,
	denom string,
	supply sdk.Int,
) *fakeLiquidKeeper {
	k.derivatives[denom] = supply
	k.lastRewardClaim[denom] = ctx.BlockTime()
	return k
}

func (k *fakeLiquidKeeper) IsDerivativeDenom(ctx sdk.Context, denom string) bool {
	return strings.HasPrefix(denom, "bkava-")
}

func (k *fakeLiquidKeeper) GetAllDerivativeDenoms(ctx sdk.Context) (denoms []string) {
	for denom := range k.derivatives {
		denoms = append(denoms, denom)
	}

	return denoms
}

func (k *fakeLiquidKeeper) GetTotalDerivativeValue(ctx sdk.Context) (sdk.Coin, error) {
	totalSupply := sdk.ZeroInt()
	for _, supply := range k.derivatives {
		totalSupply = totalSupply.Add(supply)
	}

	return sdk.NewCoin("ukava", totalSupply), nil
}

func (k *fakeLiquidKeeper) GetDerivativeValue(ctx sdk.Context, denom string) (sdk.Coin, error) {
	supply, found := k.derivatives[denom]
	if !found {
		return sdk.NewCoin("ukava", sdk.ZeroInt()), nil
	}

	return sdk.NewCoin("ukava", supply), nil
}

func (k *fakeLiquidKeeper) CollectStakingRewardsByDenom(
	ctx sdk.Context,
	derivativeDenom string,
	destinationModAccount string,
) (sdk.Coins, error) {
	amt := k.getRewardAmount(ctx, derivativeDenom)

	return sdk.NewCoins(sdk.NewCoin("ukava", amt)), nil
}

func (k *fakeLiquidKeeper) getRewardAmount(
	ctx sdk.Context,
	derivativeDenom string,
) sdk.Int {
	amt, found := k.derivatives[derivativeDenom]
	if !found {
		// No error
		return sdk.ZeroInt()
	}

	lastRewardClaim, found := k.lastRewardClaim[derivativeDenom]
	if !found {
		panic("last reward claim not found")
	}

	duration := int64(ctx.BlockTime().Sub(lastRewardClaim).Seconds())
	if duration <= 0 {
		return sdk.ZeroInt()
	}

	// Reward amount just set to 10% of the derivative supply per second
	return amt.QuoRaw(10).MulRaw(duration)
}

type fakeDistrKeeper struct {
	communityTax sdk.Dec
}

var _ types.DistrKeeper = newFakeDistrKeeper()

func newFakeDistrKeeper() *fakeDistrKeeper {
	return &fakeDistrKeeper{}
}

func (k *fakeDistrKeeper) setCommunityTax(percent sdk.Dec) *fakeDistrKeeper {
	k.communityTax = percent
	return k
}

func (k *fakeDistrKeeper) GetCommunityTax(ctx sdk.Context) (percent sdk.Dec) {
	return k.communityTax
}

type fakeMintKeeper struct {
	minter minttypes.Minter
}

var _ types.MintKeeper = newFakeMintKeeper()

func newFakeMintKeeper() *fakeMintKeeper {
	return &fakeMintKeeper{}
}

func (k *fakeMintKeeper) setMinter(minter minttypes.Minter) *fakeMintKeeper {
	k.minter = minter
	return k
}

func (k *fakeMintKeeper) GetMinter(ctx sdk.Context) (minter minttypes.Minter) {
	return k.minter
}

type fakePricefeedKeeper struct {
	prices map[string]pricefeedtypes.CurrentPrice
}

var _ types.PricefeedKeeper = newFakePricefeedKeeper()

func newFakePricefeedKeeper() *fakePricefeedKeeper {
	return &fakePricefeedKeeper{
		prices: map[string]pricefeedtypes.CurrentPrice{},
	}
}

func (k *fakePricefeedKeeper) setPrice(price pricefeedtypes.CurrentPrice) *fakePricefeedKeeper {
	k.prices[price.MarketID] = price
	return k
}

func (k *fakePricefeedKeeper) GetCurrentPrice(ctx sdk.Context, marketID string) (pricefeedtypes.CurrentPrice, error) {
	price, found := k.prices[marketID]
	if !found {
		return pricefeedtypes.CurrentPrice{}, fmt.Errorf("price not found for market %s", marketID)
	}

	return price, nil
}

type fakeBankKeeper struct {
	supply map[string]sdk.Int
}

var _ types.BankKeeper = newFakeBankKeeper()

func newFakeBankKeeper() *fakeBankKeeper {
	return &fakeBankKeeper{
		supply: map[string]sdk.Int{},
	}
}

func (k *fakeBankKeeper) setSupply(coins ...sdk.Coin) *fakeBankKeeper {
	for _, coin := range coins {
		k.supply[coin.Denom] = coin.Amount
	}

	return k
}

func (k *fakeBankKeeper) SendCoinsFromModuleToAccount(
	ctx sdk.Context,
	senderModule string,
	recipientAddr sdk.AccAddress,
	amt sdk.Coins,
) error {
	panic("not implemented")
}

func (k *fakeBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	panic("not implemented")
}

func (k *fakeBankKeeper) GetSupply(ctx sdk.Context, denom string) sdk.Coin {
	supply, found := k.supply[denom]
	if !found {
		return sdk.NewCoin(denom, sdk.ZeroInt())
	}

	return sdk.NewCoin(denom, supply)
}

// Assorted Testing Data

// note: amino panics when encoding times ≥ the start of year 10000.
var distantFuture = time.Date(9000, 1, 1, 0, 0, 0, 0, time.UTC)

func arbitraryCoins() sdk.Coins {
	return cs(c("btcb", 1))
}

func arbitraryAddress() sdk.AccAddress {
	_, addresses := app.GeneratePrivKeyAddressPairs(1)
	return addresses[0]
}

func arbitraryValidatorAddress() sdk.ValAddress {
	return generateValidatorAddresses(1)[0]
}

func generateValidatorAddresses(n int) []sdk.ValAddress {
	_, addresses := app.GeneratePrivKeyAddressPairs(n)
	var valAddresses []sdk.ValAddress
	for _, a := range addresses {
		valAddresses = append(valAddresses, sdk.ValAddress(a))
	}
	return valAddresses
}

var nonEmptyMultiRewardIndexes = types.MultiRewardIndexes{
	{
		CollateralType: "bnb",
		RewardIndexes: types.RewardIndexes{
			{
				CollateralType: "hard",
				RewardFactor:   d("0.02"),
			},
			{
				CollateralType: "ukava",
				RewardFactor:   d("0.04"),
			},
		},
	},
	{
		CollateralType: "btcb",
		RewardIndexes: types.RewardIndexes{
			{
				CollateralType: "hard",
				RewardFactor:   d("0.2"),
			},
			{
				CollateralType: "ukava",
				RewardFactor:   d("0.4"),
			},
		},
	},
}

func extractCollateralTypes(indexes types.MultiRewardIndexes) []string {
	var denoms []string
	for _, ri := range indexes {
		denoms = append(denoms, ri.CollateralType)
	}
	return denoms
}

func increaseAllRewardFactors(indexes types.MultiRewardIndexes) types.MultiRewardIndexes {
	increasedIndexes := make(types.MultiRewardIndexes, len(indexes))
	copy(increasedIndexes, indexes)

	for i := range increasedIndexes {
		increasedIndexes[i].RewardIndexes = increaseRewardFactors(increasedIndexes[i].RewardIndexes)
	}
	return increasedIndexes
}

func increaseRewardFactors(indexes types.RewardIndexes) types.RewardIndexes {
	increasedIndexes := make(types.RewardIndexes, len(indexes))
	copy(increasedIndexes, indexes)

	for i := range increasedIndexes {
		increasedIndexes[i].RewardFactor = increasedIndexes[i].RewardFactor.MulInt64(2)
	}
	return increasedIndexes
}

func appendUniqueMultiRewardIndex(indexes types.MultiRewardIndexes) types.MultiRewardIndexes {
	const uniqueDenom = "uniquedenom"

	for _, mri := range indexes {
		if mri.CollateralType == uniqueDenom {
			panic(fmt.Sprintf("tried to add unique multi reward index with denom '%s', but denom already existed", uniqueDenom))
		}
	}

	return append(indexes, types.NewMultiRewardIndex(
		uniqueDenom,
		types.RewardIndexes{
			{
				CollateralType: "hard",
				RewardFactor:   d("0.02"),
			},
			{
				CollateralType: "ukava",
				RewardFactor:   d("0.04"),
			},
		},
	),
	)
}

func appendUniqueEmptyMultiRewardIndex(indexes types.MultiRewardIndexes) types.MultiRewardIndexes {
	const uniqueDenom = "uniquedenom"

	for _, mri := range indexes {
		if mri.CollateralType == uniqueDenom {
			panic(fmt.Sprintf("tried to add unique multi reward index with denom '%s', but denom already existed", uniqueDenom))
		}
	}

	return append(indexes, types.NewMultiRewardIndex(uniqueDenom, nil))
}

func appendUniqueRewardIndexToFirstItem(indexes types.MultiRewardIndexes) types.MultiRewardIndexes {
	newIndexes := make(types.MultiRewardIndexes, len(indexes))
	copy(newIndexes, indexes)

	newIndexes[0].RewardIndexes = appendUniqueRewardIndex(newIndexes[0].RewardIndexes)
	return newIndexes
}

func appendUniqueRewardIndex(indexes types.RewardIndexes) types.RewardIndexes {
	const uniqueDenom = "uniquereward"

	for _, mri := range indexes {
		if mri.CollateralType == uniqueDenom {
			panic(fmt.Sprintf("tried to add unique reward index with denom '%s', but denom already existed", uniqueDenom))
		}
	}

	return append(
		indexes,
		types.NewRewardIndex(uniqueDenom, d("0.02")),
	)
}
