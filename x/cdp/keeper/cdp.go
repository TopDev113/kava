package keeper

import (
	"fmt"
	"sort"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/cdp/types"
)

// AddCdp adds a cdp for a specific owner and collateral type
func (k Keeper) AddCdp(ctx sdk.Context, owner sdk.AccAddress, collateral sdk.Coin, principal sdk.Coin, collateralType string) error {
	// validation
	err := k.ValidateCollateral(ctx, collateral, collateralType)
	if err != nil {
		return err
	}
	err = k.ValidateBalance(ctx, collateral, owner)
	if err != nil {
		return err
	}
	_, found := k.GetCdpByOwnerAndCollateralType(ctx, owner, collateralType)
	if found {
		return errorsmod.Wrapf(types.ErrCdpAlreadyExists, "owner %s, denom %s", owner, collateral.Denom)
	}
	err = k.ValidatePrincipalAdd(ctx, principal)
	if err != nil {
		return err
	}

	err = k.ValidateDebtLimit(ctx, collateralType, principal)
	if err != nil {
		return err
	}
	err = k.ValidateCollateralizationRatio(ctx, collateral, collateralType, principal, sdk.NewCoin(principal.Denom, sdk.ZeroInt()))
	if err != nil {
		return err
	}

	// send coins from the owners account to the cdp module
	id := k.GetNextCdpID(ctx)
	interestFactor, found := k.GetInterestFactor(ctx, collateralType)
	if !found {
		interestFactor = sdk.OneDec()
		k.SetInterestFactor(ctx, collateralType, interestFactor)

	}
	cdp := types.NewCDP(id, owner, collateral, collateralType, principal, ctx.BlockHeader().Time, interestFactor)
	deposit := types.NewDeposit(cdp.ID, owner, collateral)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, owner, types.ModuleName, sdk.NewCoins(collateral))
	if err != nil {
		return err
	}

	// mint the principal and send to the owners account
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(principal))
	if err != nil {
		panic(err)
	}
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, owner, sdk.NewCoins(principal))
	if err != nil {
		panic(err)
	}

	// mint the corresponding amount of debt coins
	err = k.MintDebtCoins(ctx, types.ModuleName, k.GetDebtDenom(ctx), principal)
	if err != nil {
		panic(err)
	}

	// update total principal for input collateral type
	k.IncrementTotalPrincipal(ctx, collateralType, principal)

	// set the cdp, deposit, and indexes in the store
	collateralToDebtRatio := k.CalculateCollateralToDebtRatio(ctx, collateral, cdp.Type, principal)
	err = k.SetCdpAndCollateralRatioIndex(ctx, cdp, collateralToDebtRatio)
	if err != nil {
		return err
	}
	k.IndexCdpByOwner(ctx, cdp)
	k.SetDeposit(ctx, deposit)
	k.SetNextCdpID(ctx, id+1)

	k.hooks.AfterCDPCreated(ctx, cdp)

	// emit events for cdp creation, deposit, and draw
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCreateCdp,
			sdk.NewAttribute(types.AttributeKeyCdpID, fmt.Sprintf("%d", cdp.ID)),
		),
	)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCdpDeposit,
			sdk.NewAttribute(sdk.AttributeKeyAmount, collateral.String()),
			sdk.NewAttribute(types.AttributeKeyCdpID, fmt.Sprintf("%d", cdp.ID)),
		),
	)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCdpDraw,
			sdk.NewAttribute(sdk.AttributeKeyAmount, principal.String()),
			sdk.NewAttribute(types.AttributeKeyCdpID, fmt.Sprintf("%d", cdp.ID)),
		),
	)

	return nil
}

// UpdateCdpAndCollateralRatioIndex updates the state of an existing cdp in the store by replacing the old index values and updating the store to the latest cdp object values
func (k Keeper) UpdateCdpAndCollateralRatioIndex(ctx sdk.Context, cdp types.CDP, ratio sdk.Dec) error {
	err := k.removeOldCollateralRatioIndex(ctx, cdp.Type, cdp.ID)
	if err != nil {
		return err
	}

	err = k.SetCDP(ctx, cdp)
	if err != nil {
		return err
	}
	k.IndexCdpByCollateralRatio(ctx, cdp.Type, cdp.ID, ratio)
	return nil
}

// DeleteCdpAndCollateralRatioIndex deletes an existing cdp in the store by removing the old index value and deleting the cdp object from the store
func (k Keeper) DeleteCdpAndCollateralRatioIndex(ctx sdk.Context, cdp types.CDP) error {
	err := k.removeOldCollateralRatioIndex(ctx, cdp.Type, cdp.ID)
	if err != nil {
		return err
	}

	return k.DeleteCDP(ctx, cdp)
}

// SetCdpAndCollateralRatioIndex sets the cdp and collateral ratio index in the store
func (k Keeper) SetCdpAndCollateralRatioIndex(ctx sdk.Context, cdp types.CDP, ratio sdk.Dec) error {
	err := k.SetCDP(ctx, cdp)
	if err != nil {
		return err
	}
	k.IndexCdpByCollateralRatio(ctx, cdp.Type, cdp.ID, ratio)
	return nil
}

func (k Keeper) removeOldCollateralRatioIndex(ctx sdk.Context, ctype string, id uint64) error {
	storedCDP, found := k.GetCDP(ctx, ctype, id)
	if !found {
		return errorsmod.Wrapf(types.ErrCdpNotFound, "%d", storedCDP.ID)
	}
	oldCollateralToDebtRatio := k.CalculateCollateralToDebtRatio(ctx, storedCDP.Collateral, storedCDP.Type, storedCDP.GetTotalPrincipal())
	k.RemoveCdpCollateralRatioIndex(ctx, storedCDP.Type, storedCDP.ID, oldCollateralToDebtRatio)
	return nil
}

// MintDebtCoins mints debt coins in the cdp module account
func (k Keeper) MintDebtCoins(ctx sdk.Context, moduleAccount string, denom string, principalCoins sdk.Coin) error {
	debtCoins := sdk.NewCoins(sdk.NewCoin(denom, principalCoins.Amount))
	return k.bankKeeper.MintCoins(ctx, moduleAccount, debtCoins)
}

// BurnDebtCoins burns debt coins from the cdp module account
func (k Keeper) BurnDebtCoins(ctx sdk.Context, moduleAccount string, denom string, paymentCoins sdk.Coin) error {
	macc := k.accountKeeper.GetModuleAccount(ctx, moduleAccount)
	maxBurnableAmount := k.bankKeeper.GetBalance(ctx, macc.GetAddress(), denom).Amount
	// check that the requested burn is not greater than the mod account balance
	debtCoins := sdk.NewCoins(sdk.NewCoin(denom, sdk.MinInt(paymentCoins.Amount, maxBurnableAmount)))
	return k.bankKeeper.BurnCoins(ctx, moduleAccount, debtCoins)
}

// GetCdpID returns the id of the cdp corresponding to a specific owner and collateral denom
func (k Keeper) GetCdpID(ctx sdk.Context, owner sdk.AccAddress, collateralType string) (uint64, bool) {
	cdpIDs, found := k.GetCdpIdsByOwner(ctx, owner)
	if !found {
		return 0, false
	}
	for _, id := range cdpIDs {
		_, found = k.GetCDP(ctx, collateralType, id)
		if found {
			return id, true
		}
	}
	return 0, false
}

// GetCdpIdsByOwner returns all the ids of cdps corresponding to a particular owner
func (k Keeper) GetCdpIdsByOwner(ctx sdk.Context, owner sdk.AccAddress) ([]uint64, bool) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CdpIDKeyPrefix)
	bz := store.Get(owner)
	if bz == nil {
		return []uint64{}, false
	}

	var index types.OwnerCDPIndex
	k.cdc.MustUnmarshal(bz, &index)
	return index.CdpIDs, true
}

// GetCdpByOwnerAndCollateralType queries cdps owned by owner and returns the cdp with matching denom
func (k Keeper) GetCdpByOwnerAndCollateralType(ctx sdk.Context, owner sdk.AccAddress, collateralType string) (types.CDP, bool) {
	cdpIDs, found := k.GetCdpIdsByOwner(ctx, owner)
	if !found {
		return types.CDP{}, false
	}
	for _, id := range cdpIDs {
		cdp, found := k.GetCDP(ctx, collateralType, id)
		if found {
			return cdp, true
		}
	}
	return types.CDP{}, false
}

// GetCDP returns the cdp associated with a particular collateral denom and id
func (k Keeper) GetCDP(ctx sdk.Context, collateralType string, cdpID uint64) (types.CDP, bool) {
	// get store
	store := prefix.NewStore(ctx.KVStore(k.key), types.CdpKeyPrefix)
	_, found := k.GetCollateral(ctx, collateralType)
	if !found {
		return types.CDP{}, false
	}
	// get CDP
	bz := store.Get(types.CdpKey(collateralType, cdpID))
	// unmarshal
	if bz == nil {
		return types.CDP{}, false
	}
	var cdp types.CDP
	k.cdc.MustUnmarshal(bz, &cdp)
	return cdp, true
}

// SetCDP sets a cdp in the store
func (k Keeper) SetCDP(ctx sdk.Context, cdp types.CDP) error {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CdpKeyPrefix)
	_, found := k.GetCollateral(ctx, cdp.Type)
	if !found {
		return errorsmod.Wrapf(types.ErrDenomPrefixNotFound, "%s", cdp.Collateral.Denom)
	}
	bz := k.cdc.MustMarshal(&cdp)
	store.Set(types.CdpKey(cdp.Type, cdp.ID), bz)
	return nil
}

// DeleteCDP deletes a cdp from the store
func (k Keeper) DeleteCDP(ctx sdk.Context, cdp types.CDP) error {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CdpKeyPrefix)
	_, found := k.GetCollateral(ctx, cdp.Type)
	if !found {
		return errorsmod.Wrapf(types.ErrDenomPrefixNotFound, "%s", cdp.Collateral.Denom)
	}
	store.Delete(types.CdpKey(cdp.Type, cdp.ID))
	return nil
}

// GetAllCdps returns all cdps from the store
func (k Keeper) GetAllCdps(ctx sdk.Context) (cdps types.CDPs) {
	k.IterateAllCdps(ctx, func(cdp types.CDP) bool {
		cdps = append(cdps, cdp)
		return false
	})
	return
}

// GetAllCdpsByCollateralType returns all cdps of a particular collateral type from the store
func (k Keeper) GetAllCdpsByCollateralType(ctx sdk.Context, collateralType string) (cdps types.CDPs) {
	k.IterateCdpsByCollateralType(ctx, collateralType, func(cdp types.CDP) bool {
		cdps = append(cdps, cdp)
		return false
	})
	return
}

// GetAllCdpsByCollateralTypeAndRatio returns all cdps of a particular collateral type and below a certain collateralization ratio
func (k Keeper) GetAllCdpsByCollateralTypeAndRatio(ctx sdk.Context, collateralType string, targetRatio sdk.Dec) (cdps types.CDPs) {
	k.IterateCdpsByCollateralRatio(ctx, collateralType, targetRatio, func(cdp types.CDP) bool {
		cdps = append(cdps, cdp)
		return false
	})
	return
}

// SetNextCdpID sets the highest cdp id in the store
func (k Keeper) SetNextCdpID(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CdpIDKey)
	store.Set(types.CdpIDKey, types.GetCdpIDBytes(id))
}

// GetNextCdpID returns the highest cdp id from the store
func (k Keeper) GetNextCdpID(ctx sdk.Context) (id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CdpIDKey)
	bz := store.Get(types.CdpIDKey)
	if bz == nil {
		panic("starting cdp id not set in genesis")
	}
	id = types.GetCdpIDFromBytes(bz)
	return
}

// IndexCdpByOwner sets the cdp id in the store, indexed by the owner
func (k Keeper) IndexCdpByOwner(ctx sdk.Context, cdp types.CDP) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CdpIDKeyPrefix)

	cdpIDs, found := k.GetCdpIdsByOwner(ctx, cdp.Owner)

	if found {
		cdpIDs = append(cdpIDs, cdp.ID)
		sort.Slice(cdpIDs, func(i, j int) bool { return cdpIDs[i] < cdpIDs[j] })
	} else {
		cdpIDs = []uint64{cdp.ID}
	}

	newIndex := types.OwnerCDPIndex{CdpIDs: cdpIDs}
	store.Set(cdp.Owner, k.cdc.MustMarshal(&newIndex))
}

// RemoveCdpOwnerIndex deletes the cdp id from the store's index of cdps by owner
func (k Keeper) RemoveCdpOwnerIndex(ctx sdk.Context, cdp types.CDP) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CdpIDKeyPrefix)

	cdpIDs, found := k.GetCdpIdsByOwner(ctx, cdp.Owner)
	if !found {
		return
	}
	updatedCdpIds := []uint64{}
	for _, id := range cdpIDs {
		if id != cdp.ID {
			updatedCdpIds = append(updatedCdpIds, id)
		}
	}
	if len(updatedCdpIds) == 0 {
		store.Delete(cdp.Owner)
		return
	}

	updatedIndex := types.OwnerCDPIndex{CdpIDs: updatedCdpIds}
	updatedBytes := k.cdc.MustMarshal(&updatedIndex)
	store.Set(cdp.Owner, updatedBytes)
}

// IndexCdpByCollateralRatio sets the cdp id in the store, indexed by the collateral type and collateral to debt ratio
func (k Keeper) IndexCdpByCollateralRatio(ctx sdk.Context, collateralType string, id uint64, collateralRatio sdk.Dec) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CollateralRatioIndexPrefix)
	_, found := k.GetCollateral(ctx, collateralType)
	if !found {
		panic(fmt.Sprintf("denom %s prefix not found", collateralType))
	}
	store.Set(types.CollateralRatioKey(collateralType, id, collateralRatio), types.GetCdpIDBytes(id))
}

// RemoveCdpCollateralRatioIndex deletes the cdp id from the store's index of cdps by collateral type and collateral to debt ratio
func (k Keeper) RemoveCdpCollateralRatioIndex(ctx sdk.Context, collateralType string, id uint64, collateralRatio sdk.Dec) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.CollateralRatioIndexPrefix)
	_, found := k.GetCollateral(ctx, collateralType)
	if !found {
		panic(fmt.Sprintf("denom %s prefix not found", collateralType))
	}
	store.Delete(types.CollateralRatioKey(collateralType, id, collateralRatio))
}

// GetDebtDenom returns the denom of debt in the system
func (k Keeper) GetDebtDenom(ctx sdk.Context) string {
	store := prefix.NewStore(ctx.KVStore(k.key), types.DebtDenomKey)
	bz := store.Get(types.DebtDenomKey)
	return string(bz)
}

// GetGovDenom returns the denom of the governance token
func (k Keeper) GetGovDenom(ctx sdk.Context) string {
	store := prefix.NewStore(ctx.KVStore(k.key), types.GovDenomKey)
	bz := store.Get(types.GovDenomKey)
	return string(bz)
}

// SetDebtDenom set the denom of debt in the system
func (k Keeper) SetDebtDenom(ctx sdk.Context, denom string) {
	if denom == "" {
		panic("debt denom not set in genesis")
	}
	store := prefix.NewStore(ctx.KVStore(k.key), types.DebtDenomKey)
	store.Set(types.DebtDenomKey, []byte(denom))
}

// SetGovDenom set the denom of the governance token in the system
func (k Keeper) SetGovDenom(ctx sdk.Context, denom string) {
	if denom == "" {
		panic("gov denom not set in genesis")
	}
	store := prefix.NewStore(ctx.KVStore(k.key), types.GovDenomKey)
	store.Set(types.GovDenomKey, []byte(denom))
}

// ValidateCollateral validates that a collateral is valid for use in cdps
func (k Keeper) ValidateCollateral(ctx sdk.Context, collateral sdk.Coin, collateralType string) error {
	cp, found := k.GetCollateral(ctx, collateralType)
	if !found {
		return errorsmod.Wrap(types.ErrCollateralNotSupported, collateral.Denom)
	}
	if cp.Denom != collateral.Denom {
		return errorsmod.Wrapf(types.ErrInvalidCollateral, "collateral type: %s expected denom: %s got: %s", collateralType, cp.Denom, collateral.Denom)
	}
	ok := k.GetMarketStatus(ctx, cp.SpotMarketID)
	if !ok {
		return errorsmod.Wrap(types.ErrPricefeedDown, collateral.Denom)
	}
	ok = k.GetMarketStatus(ctx, cp.LiquidationMarketID)
	if !ok {
		return errorsmod.Wrap(types.ErrPricefeedDown, collateral.Denom)
	}
	return nil
}

// ValidatePrincipalAdd validates that an asset is valid for use as debt when creating a new cdp
func (k Keeper) ValidatePrincipalAdd(ctx sdk.Context, principal sdk.Coin) error {
	dp, found := k.GetDebtParam(ctx, principal.Denom)
	if !found {
		return errorsmod.Wrap(types.ErrDebtNotSupported, principal.Denom)
	}
	if principal.Amount.LT(dp.DebtFloor) {
		return errorsmod.Wrapf(types.ErrBelowDebtFloor, "proposed %s < minimum %s", principal, dp.DebtFloor)
	}
	return nil
}

// ValidatePrincipalDraw validates that an asset is valid for use as debt when drawing debt off an existing cdp
func (k Keeper) ValidatePrincipalDraw(ctx sdk.Context, principal sdk.Coin, expectedDenom string) error {
	if principal.Denom != expectedDenom {
		return errorsmod.Wrapf(types.ErrInvalidDebtRequest, "proposed %s, expected %s", principal.Denom, expectedDenom)
	}
	_, found := k.GetDebtParam(ctx, principal.Denom)
	if !found {
		return errorsmod.Wrap(types.ErrDebtNotSupported, principal.Denom)
	}
	return nil
}

// ValidateDebtLimit validates that the input debt amount does not exceed the global debt limit or the debt limit for that collateral
func (k Keeper) ValidateDebtLimit(ctx sdk.Context, collateralType string, principal sdk.Coin) error {
	cp, found := k.GetCollateral(ctx, collateralType)
	if !found {
		return errorsmod.Wrap(types.ErrCollateralNotSupported, collateralType)
	}
	totalPrincipal := k.GetTotalPrincipal(ctx, collateralType, principal.Denom).Add(principal.Amount)
	collateralLimit := cp.DebtLimit.Amount
	if totalPrincipal.GT(collateralLimit) {
		return errorsmod.Wrapf(types.ErrExceedsDebtLimit, "debt increase %s > collateral debt limit %s", sdk.NewCoins(sdk.NewCoin(principal.Denom, totalPrincipal)), sdk.NewCoins(sdk.NewCoin(principal.Denom, collateralLimit)))
	}
	globalLimit := k.GetParams(ctx).GlobalDebtLimit.Amount
	if totalPrincipal.GT(globalLimit) {
		return errorsmod.Wrapf(types.ErrExceedsDebtLimit, "debt increase %s > global debt limit  %s", sdk.NewCoin(principal.Denom, totalPrincipal), sdk.NewCoin(principal.Denom, globalLimit))
	}
	return nil
}

// ValidateCollateralizationRatio validate that adding the input principal doesn't put the cdp below the liquidation ratio
func (k Keeper) ValidateCollateralizationRatio(ctx sdk.Context, collateral sdk.Coin, collateralType string, principal sdk.Coin, fees sdk.Coin) error {
	collateralizationRatio, err := k.CalculateCollateralizationRatio(ctx, collateral, collateralType, principal, fees, spot)
	if err != nil {
		return err
	}
	liquidationRatio := k.getLiquidationRatio(ctx, collateralType)
	if collateralizationRatio.LT(liquidationRatio) {
		return errorsmod.Wrapf(types.ErrInvalidCollateralRatio, "collateral %s, collateral ratio %s, liquidation ratio %s", collateral.Denom, collateralizationRatio, liquidationRatio)
	}
	return nil
}

// ValidateBalance validates that the input account has sufficient spendable funds
func (k Keeper) ValidateBalance(ctx sdk.Context, amount sdk.Coin, sender sdk.AccAddress) error {
	acc := k.accountKeeper.GetAccount(ctx, sender)
	if acc == nil {
		return errorsmod.Wrapf(types.ErrAccountNotFound, "address: %s", sender)
	}
	spendableBalance := k.bankKeeper.SpendableCoins(ctx, acc.GetAddress()).AmountOf(amount.Denom)
	if spendableBalance.LT(amount.Amount) {
		return errorsmod.Wrapf(types.ErrInsufficientBalance, "%s < %s", sdk.NewCoin(amount.Denom, spendableBalance), amount)
	}

	return nil
}

// CalculateCollateralToDebtRatio returns the collateral to debt ratio of the input collateral and debt amounts
func (k Keeper) CalculateCollateralToDebtRatio(ctx sdk.Context, collateral sdk.Coin, collateralType string, debt sdk.Coin) sdk.Dec {
	debtTotal := k.convertDebtToBaseUnits(ctx, debt)

	if debtTotal.IsZero() || debtTotal.GTE(types.MaxSortableDec) {
		return types.MaxSortableDec.Sub(sdk.SmallestDec())
	}

	collateralBaseUnits := k.convertCollateralToBaseUnits(ctx, collateral, collateralType)
	return collateralBaseUnits.Quo(debtTotal)
}

// LoadAugmentedCDP creates a new augmented CDP from an existing CDP
func (k Keeper) LoadAugmentedCDP(ctx sdk.Context, cdp types.CDP) types.AugmentedCDP {
	// sync the latest interest of the cdp
	interestAccumulated := k.CalculateNewInterest(ctx, cdp)
	cdp.AccumulatedFees = cdp.AccumulatedFees.Add(interestAccumulated)
	// update cdp fields to match synced accumulated fees
	prevAccrualTime, found := k.GetPreviousAccrualTime(ctx, cdp.Type)
	if found {
		cdp.FeesUpdated = prevAccrualTime
	}
	globalInterestFactor, found := k.GetInterestFactor(ctx, cdp.Type)
	if found {
		cdp.InterestFactor = globalInterestFactor
	}
	// calculate collateralization ratio
	collateralizationRatio, err := k.CalculateCollateralizationRatio(ctx, cdp.Collateral, cdp.Type, cdp.Principal, cdp.AccumulatedFees, liquidation)
	if err != nil {
		return types.AugmentedCDP{CDP: cdp}
	}
	// convert collateral value to debt coin
	totalDebt := cdp.GetTotalPrincipal().Amount
	collateralValueInDebtDenom := sdk.NewDecFromInt(totalDebt).Mul(collateralizationRatio)
	collateralValueInDebt := sdk.NewCoin(cdp.Principal.Denom, collateralValueInDebtDenom.RoundInt())
	// create new augmuented cdp
	augmentedCDP := types.NewAugmentedCDP(cdp, collateralValueInDebt, collateralizationRatio)
	return augmentedCDP
}

// LoadCDPResponse creates a new CDPResponse from an existing CDP
func (k Keeper) LoadCDPResponse(ctx sdk.Context, cdp types.CDP) types.CDPResponse {
	// sync the latest interest of the cdp
	interestAccumulated := k.CalculateNewInterest(ctx, cdp)
	cdp.AccumulatedFees = cdp.AccumulatedFees.Add(interestAccumulated)
	// update cdp fields to match synced accumulated fees
	prevAccrualTime, found := k.GetPreviousAccrualTime(ctx, cdp.Type)
	if found {
		cdp.FeesUpdated = prevAccrualTime
	}
	globalInterestFactor, found := k.GetInterestFactor(ctx, cdp.Type)
	if found {
		cdp.InterestFactor = globalInterestFactor
	}
	// calculate collateralization ratio
	collateralizationRatio, err := k.CalculateCollateralizationRatio(ctx, cdp.Collateral, cdp.Type, cdp.Principal, cdp.AccumulatedFees, liquidation)
	if err != nil {
		return types.CDPResponse{
			ID:              cdp.ID,
			Owner:           cdp.Owner.String(),
			Type:            cdp.Type,
			Collateral:      cdp.Collateral,
			Principal:       cdp.Principal,
			AccumulatedFees: cdp.AccumulatedFees,
			FeesUpdated:     cdp.FeesUpdated,
			InterestFactor:  cdp.InterestFactor.String(),
		}
	}
	// convert collateral value to debt coin
	totalDebt := cdp.GetTotalPrincipal().Amount
	collateralValueInDebtDenom := sdk.NewDecFromInt(totalDebt).Mul(collateralizationRatio)
	collateralValueInDebt := sdk.NewCoin(cdp.Principal.Denom, collateralValueInDebtDenom.RoundInt())
	// create new cdp response
	return types.NewCDPResponse(cdp, collateralValueInDebt, collateralizationRatio)
}

// CalculateCollateralizationRatio returns the collateralization ratio of the input collateral to the input debt plus fees
func (k Keeper) CalculateCollateralizationRatio(ctx sdk.Context, collateral sdk.Coin, collateralType string, principal sdk.Coin, fees sdk.Coin, pfType pricefeedType) (sdk.Dec, error) {
	if collateral.IsZero() {
		return sdk.ZeroDec(), nil
	}
	var marketID string
	switch pfType {
	case spot:
		marketID = k.getSpotMarketID(ctx, collateralType)
	case liquidation:
		marketID = k.getliquidationMarketID(ctx, collateralType)
	default:
		return sdk.Dec{}, pfType.IsValid()
	}

	price, err := k.pricefeedKeeper.GetCurrentPrice(ctx, marketID)
	if err != nil {
		return sdk.Dec{}, err
	}
	collateralBaseUnits := k.convertCollateralToBaseUnits(ctx, collateral, collateralType)
	collateralValue := collateralBaseUnits.Mul(price.Price)

	prinicpalBaseUnits := k.convertDebtToBaseUnits(ctx, principal)
	principalTotal := prinicpalBaseUnits
	feeBaseUnits := k.convertDebtToBaseUnits(ctx, fees)
	principalTotal = principalTotal.Add(feeBaseUnits)

	collateralRatio := collateralValue.Quo(principalTotal)
	return collateralRatio, nil
}

// CalculateCollateralizationRatioFromAbsoluteRatio takes a coin's denom and an absolute ratio and returns the respective collateralization ratio
func (k Keeper) CalculateCollateralizationRatioFromAbsoluteRatio(ctx sdk.Context, collateralType string, absoluteRatio sdk.Dec, pfType pricefeedType) (sdk.Dec, error) {
	// get price of collateral
	var marketID string
	switch pfType {
	case spot:
		marketID = k.getSpotMarketID(ctx, collateralType)
	case liquidation:
		marketID = k.getliquidationMarketID(ctx, collateralType)
	default:
		return sdk.Dec{}, pfType.IsValid()
	}

	price, err := k.pricefeedKeeper.GetCurrentPrice(ctx, marketID)
	if err != nil {
		return sdk.Dec{}, err
	}
	// convert absolute ratio to collateralization ratio
	respectiveCollateralRatio := absoluteRatio.Quo(price.Price)
	return respectiveCollateralRatio, nil
}

// SetMarketStatus sets the status of the input market, true means the market is up and running, false means it is down
func (k Keeper) SetMarketStatus(ctx sdk.Context, marketID string, up bool) {
	store := prefix.NewStore(ctx.KVStore(k.key), types.PricefeedStatusKeyPrefix)
	if up {
		store.Set([]byte(marketID), []byte{})
	} else {
		store.Delete([]byte(marketID))
	}
}

// GetMarketStatus returns true if the market has a price, otherwise false
func (k Keeper) GetMarketStatus(ctx sdk.Context, marketID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.key), types.PricefeedStatusKeyPrefix)
	bz := store.Get([]byte(marketID))
	return bz != nil
}

// UpdatePricefeedStatus determines if the price of an asset is available and updates the global status of the market
func (k Keeper) UpdatePricefeedStatus(ctx sdk.Context, marketID string) (ok bool) {
	_, err := k.pricefeedKeeper.GetCurrentPrice(ctx, marketID)
	if err != nil {
		k.SetMarketStatus(ctx, marketID, false)
		return false
	}
	k.SetMarketStatus(ctx, marketID, true)
	return true
}

// converts the input collateral to base units (ie multiplies the input by 10^(-ConversionFactor))
func (k Keeper) convertCollateralToBaseUnits(ctx sdk.Context, collateral sdk.Coin, collateralType string) (baseUnits sdk.Dec) {
	cp, _ := k.GetCollateral(ctx, collateralType)
	return sdk.NewDecFromInt(collateral.Amount).Mul(sdk.NewDecFromIntWithPrec(sdk.OneInt(), cp.ConversionFactor.Int64()))
}

// converts the input debt to base units (ie multiplies the input by 10^(-ConversionFactor))
func (k Keeper) convertDebtToBaseUnits(ctx sdk.Context, debt sdk.Coin) (baseUnits sdk.Dec) {
	dp, _ := k.GetDebtParam(ctx, debt.Denom)
	return sdk.NewDecFromInt(debt.Amount).Mul(sdk.NewDecFromIntWithPrec(sdk.OneInt(), dp.ConversionFactor.Int64()))
}

type pricefeedType string

const (
	spot        pricefeedType = "spot"
	liquidation pricefeedType = "liquidation"
)

func (pft pricefeedType) IsValid() error {
	switch pft {
	case spot, liquidation:
		return nil
	}
	return fmt.Errorf("invalid pricefeed type: %s", pft)
}
