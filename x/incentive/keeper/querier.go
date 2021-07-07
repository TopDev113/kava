package keeper

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/kava-labs/kava/x/incentive/types"
)

// NewQuerier is the module level router for state queries
func NewQuerier(k Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err error) {
		switch path[0] {
		case types.QueryGetParams:
			return queryGetParams(ctx, req, k)
		case types.QueryGetHardRewards:
			return queryGetHardRewards(ctx, req, k)
		case types.QueryGetHardRewardsUnsynced:
			return queryGetHardRewardsUnsynced(ctx, req, k)
		case types.QueryGetUSDXMintingRewards:
			return queryGetUSDXMintingRewards(ctx, req, k)
		case types.QueryGetUSDXMintingRewardsUnsynced:
			return queryGetUSDXMintingRewardsUnsynced(ctx, req, k)
		case types.QueryGetDelegatorRewards:
			return queryGetDelegatorRewards(ctx, req, k)
		case types.QueryGetDelegatorRewardsUnsynced:
			return queryGetDelegatorRewardsUnsynced(ctx, req, k)
		case types.QueryGetRewardFactors:
			return queryGetRewardFactors(ctx, req, k)
		default:
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown %s query endpoint", types.ModuleName)
		}
	}
}

// query params in the store
func queryGetParams(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	// Get params
	params := k.GetParams(ctx)

	// Encode results
	bz, err := codec.MarshalJSONIndent(k.cdc, params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryGetHardRewards(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryHardRewardsParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	owner := len(params.Owner) > 0

	var hardClaims types.HardLiquidityProviderClaims
	switch {
	case owner:
		hardClaim, foundHardClaim := k.GetHardLiquidityProviderClaim(ctx, params.Owner)
		if foundHardClaim {
			hardClaims = append(hardClaims, hardClaim)
		}
	default:
		hardClaims = k.GetAllHardLiquidityProviderClaims(ctx)
	}

	var paginatedHardClaims types.HardLiquidityProviderClaims
	startH, endH := client.Paginate(len(hardClaims), params.Page, params.Limit, 100)
	if startH < 0 || endH < 0 {
		paginatedHardClaims = types.HardLiquidityProviderClaims{}
	} else {
		paginatedHardClaims = hardClaims[startH:endH]
	}

	var augmentedHardClaims types.HardLiquidityProviderClaims
	for _, claim := range paginatedHardClaims {
		augmentedClaim := k.SimulateHardSynchronization(ctx, claim)
		augmentedHardClaims = append(augmentedHardClaims, augmentedClaim)
	}

	// Marshal Hard claims
	bz, err := codec.MarshalJSONIndent(k.cdc, augmentedHardClaims)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryGetHardRewardsUnsynced(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryHardRewardsUnsyncedParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	owner := len(params.Owner) > 0

	var hardClaims types.HardLiquidityProviderClaims
	switch {
	case owner:
		hardClaim, foundHardClaim := k.GetHardLiquidityProviderClaim(ctx, params.Owner)
		if foundHardClaim {
			hardClaims = append(hardClaims, hardClaim)
		}
	default:
		hardClaims = k.GetAllHardLiquidityProviderClaims(ctx)
	}

	var paginatedHardClaims types.HardLiquidityProviderClaims
	startH, endH := client.Paginate(len(hardClaims), params.Page, params.Limit, 100)
	if startH < 0 || endH < 0 {
		paginatedHardClaims = types.HardLiquidityProviderClaims{}
	} else {
		paginatedHardClaims = hardClaims[startH:endH]
	}

	// Marshal Hard claims
	bz, err := codec.MarshalJSONIndent(k.cdc, paginatedHardClaims)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryGetUSDXMintingRewards(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryUSDXMintingRewardsParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	owner := len(params.Owner) > 0

	var usdxMintingClaims types.USDXMintingClaims
	switch {
	case owner:
		usdxMintingClaim, foundUsdxMintingClaim := k.GetUSDXMintingClaim(ctx, params.Owner)
		if foundUsdxMintingClaim {
			usdxMintingClaims = append(usdxMintingClaims, usdxMintingClaim)
		}
	default:
		usdxMintingClaims = k.GetAllUSDXMintingClaims(ctx)
	}

	var paginatedUsdxMintingClaims types.USDXMintingClaims
	startU, endU := client.Paginate(len(usdxMintingClaims), params.Page, params.Limit, 100)
	if startU < 0 || endU < 0 {
		paginatedUsdxMintingClaims = types.USDXMintingClaims{}
	} else {
		paginatedUsdxMintingClaims = usdxMintingClaims[startU:endU]
	}

	var augmentedUsdxMintingClaims types.USDXMintingClaims
	for _, claim := range paginatedUsdxMintingClaims {
		augmentedClaim := k.SimulateUSDXMintingSynchronization(ctx, claim)
		augmentedUsdxMintingClaims = append(augmentedUsdxMintingClaims, augmentedClaim)
	}

	// Marshal USDX minting claims
	bz, err := codec.MarshalJSONIndent(k.cdc, augmentedUsdxMintingClaims)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryGetUSDXMintingRewardsUnsynced(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryUSDXMintingRewardsUnsyncedParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	owner := len(params.Owner) > 0

	var usdxMintingClaims types.USDXMintingClaims
	switch {
	case owner:
		usdxMintingClaim, foundUsdxMintingClaim := k.GetUSDXMintingClaim(ctx, params.Owner)
		if foundUsdxMintingClaim {
			usdxMintingClaims = append(usdxMintingClaims, usdxMintingClaim)
		}
	default:
		usdxMintingClaims = k.GetAllUSDXMintingClaims(ctx)
	}

	var paginatedUsdxMintingClaims types.USDXMintingClaims
	startU, endU := client.Paginate(len(usdxMintingClaims), params.Page, params.Limit, 100)
	if startU < 0 || endU < 0 {
		paginatedUsdxMintingClaims = types.USDXMintingClaims{}
	} else {
		paginatedUsdxMintingClaims = usdxMintingClaims[startU:endU]
	}

	// Marshal USDX minting claims
	bz, err := codec.MarshalJSONIndent(k.cdc, paginatedUsdxMintingClaims)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryGetDelegatorRewards(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryDelegatorRewardsParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	owner := len(params.Owner) > 0

	var delegatorClaims types.DelegatorClaims
	switch {
	case owner:
		delegatorClaim, foundDelegatorClaim := k.GetDelegatorClaim(ctx, params.Owner)
		if foundDelegatorClaim {
			delegatorClaims = append(delegatorClaims, delegatorClaim)
		}
	default:
		delegatorClaims = k.GetAllDelegatorClaims(ctx)
	}

	var paginatedDelegatorClaims types.DelegatorClaims
	startH, endH := client.Paginate(len(delegatorClaims), params.Page, params.Limit, 100)
	if startH < 0 || endH < 0 {
		paginatedDelegatorClaims = types.DelegatorClaims{}
	} else {
		paginatedDelegatorClaims = delegatorClaims[startH:endH]
	}

	var augmentedDelegatorClaims types.DelegatorClaims
	for _, claim := range paginatedDelegatorClaims {
		augmentedClaim := k.SimulateDelegatorSynchronization(ctx, claim)
		augmentedDelegatorClaims = append(augmentedDelegatorClaims, augmentedClaim)
	}

	// Marshal Hard claims
	bz, err := codec.MarshalJSONIndent(k.cdc, augmentedDelegatorClaims)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryGetDelegatorRewardsUnsynced(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryDelegatorRewardsUnsyncedParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	owner := len(params.Owner) > 0

	var delegatorClaims types.DelegatorClaims
	switch {
	case owner:
		delegatorClaim, foundHardClaim := k.GetDelegatorClaim(ctx, params.Owner)
		if foundHardClaim {
			delegatorClaims = append(delegatorClaims, delegatorClaim)
		}
	default:
		delegatorClaims = k.GetAllDelegatorClaims(ctx)
	}

	var paginatedDelegatorClaims types.DelegatorClaims
	startH, endH := client.Paginate(len(delegatorClaims), params.Page, params.Limit, 100)
	if startH < 0 || endH < 0 {
		paginatedDelegatorClaims = types.DelegatorClaims{}
	} else {
		paginatedDelegatorClaims = delegatorClaims[startH:endH]
	}

	// Marshal Hard claims
	bz, err := codec.MarshalJSONIndent(k.cdc, paginatedDelegatorClaims)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryGetRewardFactors(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryRewardFactorsParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	var rewardFactors types.RewardFactors
	if len(params.Denom) > 0 {
		// Fetch reward factors for a single denom
		rewardFactor := types.RewardFactor{}
		rewardFactor.Denom = params.Denom

		usdxMintingRewardFactor, found := k.GetUSDXMintingRewardFactor(ctx, params.Denom)
		if found {
			rewardFactor.USDXMintingRewardFactor = usdxMintingRewardFactor
		}
		hardSupplyRewardIndexes, found := k.GetHardSupplyRewardIndexes(ctx, params.Denom)
		if found {
			rewardFactor.HardSupplyRewardFactors = hardSupplyRewardIndexes
		}
		hardBorrowRewardIndexes, found := k.GetHardBorrowRewardIndexes(ctx, params.Denom)
		if found {
			rewardFactor.HardBorrowRewardFactors = hardBorrowRewardIndexes
		}
		delegatorRewardIndexes, found := k.GetDelegatorRewardIndexes(ctx, params.Denom)
		if found {
			rewardFactor.DelegatorRewardFactors = delegatorRewardIndexes
		}
		rewardFactors = append(rewardFactors, rewardFactor)
	} else {
		rewardFactorMap := make(map[string]types.RewardFactor)

		// Populate mapping with usdx minting reward factors
		k.IterateUSDXMintingRewardFactors(ctx, func(denom string, factor sdk.Dec) (stop bool) {
			rewardFactor := types.RewardFactor{Denom: denom, USDXMintingRewardFactor: factor}
			rewardFactorMap[denom] = rewardFactor
			return false
		})

		// Populate mapping with Hard supply reward factors
		k.IterateHardSupplyRewardIndexes(ctx, func(denom string, indexes types.RewardIndexes) (stop bool) {
			rewardFactor, ok := rewardFactorMap[denom]
			if !ok {
				rewardFactor = types.RewardFactor{Denom: denom, HardSupplyRewardFactors: indexes}
			} else {
				rewardFactor.HardSupplyRewardFactors = indexes
			}
			rewardFactorMap[denom] = rewardFactor
			return false
		})

		// Populate mapping with Hard borrow reward factors
		k.IterateHardBorrowRewardIndexes(ctx, func(denom string, indexes types.RewardIndexes) (stop bool) {
			rewardFactor, ok := rewardFactorMap[denom]
			if !ok {
				rewardFactor = types.RewardFactor{Denom: denom, HardBorrowRewardFactors: indexes}
			} else {
				rewardFactor.HardBorrowRewardFactors = indexes
			}
			rewardFactorMap[denom] = rewardFactor
			return false
		})

		// Populate mapping with delegator reward factors
		k.IterateDelegatorRewardIndexes(ctx, func(denom string, indexes types.RewardIndexes) (stop bool) {
			rewardFactor, ok := rewardFactorMap[denom]
			if !ok {
				rewardFactor = types.RewardFactor{Denom: denom, DelegatorRewardFactors: indexes}
			} else {
				rewardFactor.DelegatorRewardFactors = indexes
			}
			rewardFactorMap[denom] = rewardFactor
			return false
		})

		// Translate mapping to slice
		for _, val := range rewardFactorMap {
			rewardFactors = append(rewardFactors, val)
		}
	}

	bz, err := codec.MarshalJSONIndent(types.ModuleCdc, rewardFactors)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}
