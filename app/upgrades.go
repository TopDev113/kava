package app

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	committeekeeper "github.com/kava-labs/kava/x/committee/keeper"
	committeetypes "github.com/kava-labs/kava/x/committee/types"
	communitytypes "github.com/kava-labs/kava/x/community/types"
	hardkeeper "github.com/kava-labs/kava/x/hard/keeper"
	incentivekeeper "github.com/kava-labs/kava/x/incentive/keeper"
)

const (
	MainnetUpgradeName = "v0.23.0"
	TestnetUpgradeName = "v0.23.0-alpha.0"

	MainnetStabilityCommitteeId = uint64(1)
	TestnetStabilityCommitteeId = uint64(1)
)

func (app App) RegisterUpgradeHandlers() {
	// register upgrade handler for mainnet
	app.upgradeKeeper.SetUpgradeHandler(MainnetUpgradeName,
		func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info("running mainnet upgrade handler")

			toVM, err := app.mm.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return toVM, err
			}

			app.Logger().Info("move all community pool funds from x/distribution to x/community")
			FundCommunityPoolModule(ctx, app.distrKeeper, app.bankKeeper, app)

			app.Logger().Info("granting x/gov module account x/community module authz messages")
			GrantGovCommunityPoolMessages(ctx, app.authzKeeper, app.accountKeeper)

			app.Logger().Info(fmt.Sprintf(
				"adding lend & cdp committee permissions to stability committee (id=%d)",
				MainnetStabilityCommitteeId,
			))
			AddNewPermissionsToStabilityCommittee(ctx, app.committeeKeeper, MainnetStabilityCommitteeId)

			app.Logger().Info(fmt.Sprintf(
				"removing x/evm AllowedParamsChange from stability committee (id=%d)",
				MainnetStabilityCommitteeId,
			))
			RemoveEVMCommitteePermissions(ctx, app.committeeKeeper, MainnetStabilityCommitteeId)

			app.Logger().Info("enabling community pool incentive tracking")
			EnableCommunityPoolIncentiveTracking(ctx, app.hardKeeper, app.incentiveKeeper)

			return toVM, nil
		},
	)

	// register upgrade handler for testnet. This only runs the module
	// migrations, as testnet already ran the other upgrades in v0.22.0.
	app.upgradeKeeper.SetUpgradeHandler(TestnetUpgradeName,
		func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info("running testnet upgrade handler")

			toVM, err := app.mm.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return toVM, err
			}

			return toVM, nil
		},
	)

	upgradeInfo, err := app.upgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	// note: no store updates
	doUpgrade := upgradeInfo.Name == MainnetUpgradeName || upgradeInfo.Name == TestnetUpgradeName
	if doUpgrade && !app.upgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

// GrantGovCommunityPoolMessages grants x/gov module account access to submit x/authz messages from the community pool module account.
func GrantGovCommunityPoolMessages(
	ctx sdk.Context,
	authzKeeper authzkeeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
) {
	communityAddr := accountKeeper.GetModuleAddress(communitytypes.ModuleName)
	govAddr := accountKeeper.GetModuleAddress(govtypes.ModuleName)
	allowedMsgs := GetCommunityPoolAllowedMsgs()
	for _, msg := range allowedMsgs {
		auth := authz.NewGenericAuthorization(msg)
		if err := authzKeeper.SaveGrant(ctx, govAddr, communityAddr, auth, nil); err != nil {
			panic(fmt.Errorf("failed to grant msg %s to x/gov account: %w", msg, err))
		}
	}
}

// MoveCommunityPoolFunds takes the full balance of the original community pool (the auth fee pool)
// and transfers them to the new community pool (the x/community module account)
func FundCommunityPoolModule(
	ctx sdk.Context,
	distKeeper distrkeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	app App,
) {
	// get balance of original community pool
	balance, leftoverDust := distKeeper.GetFeePoolCommunityCoins(ctx).TruncateDecimal()
	app.Logger().Info(fmt.Sprintf("community pool balance: %v, dust: %v", balance, leftoverDust))

	// the balance of the community fee pool is held by the distribution module.
	// transfer whole pool balance from distribution module to new community pool module account
	err := bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		distrtypes.ModuleName,
		communitytypes.ModuleAccountName,
		balance,
	)
	if err != nil {
		panic(fmt.Errorf(
			"failed to transfer community pool funds to new community pool module account: %w",
			err,
		))
	}

	// make sure x/distribution knows that there're no funds in the community pool.
	// we keep the leftover decimal change in the account to ensure all funds are accounted for.
	feePool := distKeeper.GetFeePool(ctx)
	feePool.CommunityPool = leftoverDust
	distKeeper.SetFeePool(ctx, feePool)
}

// AddNewPermissionsToStabilityCommittee adds the following permissions to the committee with the passed in id:
// - CommunityCDPRepayDebtPermission
// - CommunityPoolLendWithdrawPermission
// - CommunityCDPWithdrawCollateralPermission
func AddNewPermissionsToStabilityCommittee(ctx sdk.Context, committeeKeeper committeekeeper.Keeper, committeeId uint64) {
	// get committee
	comm, found := committeeKeeper.GetCommittee(ctx, committeeId)
	if !found {
		panic(fmt.Sprintf("expected to find committee with id %d but found none", committeeId))
	}
	// set new permissions
	perms := comm.GetPermissions()
	perms = append(perms,
		&committeetypes.CommunityCDPRepayDebtPermission{},
		&committeetypes.CommunityPoolLendWithdrawPermission{},
		&committeetypes.CommunityCDPWithdrawCollateralPermission{},
	)
	comm.SetPermissions(perms)
	// save permission changes
	committeeKeeper.SetCommittee(ctx, comm)
}

// RemoveEVMCommitteePermissions removes the following permissions to the committee with the passed in id:
// - AllowedParamsChange for x/evm subspace (no longer stored in params)
func RemoveEVMCommitteePermissions(ctx sdk.Context, committeeKeeper committeekeeper.Keeper, committeeId uint64) {
	// get committee
	comm, found := committeeKeeper.GetCommittee(ctx, committeeId)
	if !found {
		panic(fmt.Sprintf("expected to find committee with id %d but found none", committeeId))
	}
	perms := comm.GetPermissions()

	// Remove the x/evm param permissions
	for i, permission := range perms {
		// Must be a pointer
		paramPerm, ok := permission.(*committeetypes.ParamsChangePermission)
		if !ok {
			continue
		}

		var newAllowedParamsChanges committeetypes.AllowedParamsChanges
		for _, param := range paramPerm.AllowedParamsChanges {
			// Exclude the x/evm allowed param
			if param.Subspace == evmtypes.ModuleName {
				continue
			}

			newAllowedParamsChanges = append(newAllowedParamsChanges, param)
		}

		// Update the allowed params
		paramPerm.AllowedParamsChanges = newAllowedParamsChanges

		// Update the permission
		perms[i] = paramPerm
	}

	comm.SetPermissions(perms)
	// save permission changes
	committeeKeeper.SetCommittee(ctx, comm)
}

func GetCommunityPoolAllowedMsgs() []string {
	return []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.bank.v1beta1.MsgMultiSend",
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.staking.v1beta1.MsgBeginRedelegate",
		"/cosmos.staking.v1beta1.MsgUndelegate",
		"/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation",
		"/cosmos.distribution.v1beta1.MsgSetWithdrawAddress",
		"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
		"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission",
		"/cosmos.distribution.v1beta1.MsgFundCommunityPool",
		"/kava.cdp.v1beta1.MsgCreateCDP",
		"/kava.cdp.v1beta1.MsgDeposit",
		"/kava.cdp.v1beta1.MsgWithdraw",
		"/kava.cdp.v1beta1.MsgDrawDebt",
		"/kava.cdp.v1beta1.MsgRepayDebt",
		"/kava.hard.v1beta1.MsgDeposit",
		"/kava.hard.v1beta1.MsgWithdraw",
		"/kava.hard.v1beta1.MsgBorrow",
		"/kava.hard.v1beta1.MsgRepay",
		"/kava.swap.v1beta1.MsgDeposit",
		"/kava.swap.v1beta1.MsgWithdraw",
		"/kava.swap.v1beta1.MsgSwapExactForTokens",
		"/kava.swap.v1beta1.MsgSwapForExactTokens",
		"/kava.liquid.v1beta1.MsgMintDerivative",
		"/kava.liquid.v1beta1.MsgBurnDerivative",
	}
}

// EnableCommunityPoolIncentiveTracking fixes an issue in v0.21.0 where the community account was not tracked by incentive.
// It only updates tracking for deposits and assumes there have been no borrows.
func EnableCommunityPoolIncentiveTracking(ctx sdk.Context, hardkeeper hardkeeper.Keeper, incentiveKeeper incentivekeeper.Keeper) {
	communityAddr := authtypes.NewModuleAddress(communitytypes.ModuleName)
	deposit, found := hardkeeper.GetDeposit(ctx, communityAddr)
	if !found {
		return
	}

	_, found = incentiveKeeper.GetHardLiquidityProviderClaim(ctx, communityAddr)
	if found {
		return // skip if claim already exists
	}
	incentiveKeeper.InitializeHardSupplyReward(ctx, deposit)
}
