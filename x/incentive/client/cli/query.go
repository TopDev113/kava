package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/kava-labs/kava/x/incentive/types"
)

const (
	flagOwner    = "owner"
	flagType     = "type"
	flagUnsynced = "unsynced"
	flagDenom    = "denom"

	typeDelegator   = "delegator"
	typeHard        = "hard"
	typeUSDXMinting = "usdx-minting"
	typeSwap        = "swap"
)

var rewardTypes = []string{typeDelegator, typeHard, typeUSDXMinting, typeSwap}

// GetQueryCmd returns the cli query commands for the incentive module
func GetQueryCmd(queryRoute string, cdc *codec.Codec) *cobra.Command {
	incentiveQueryCmd := &cobra.Command{
		Use:   types.ModuleName,
		Short: "Querying commands for the incentive module",
	}

	incentiveQueryCmd.AddCommand(flags.GetCommands(
		queryParamsCmd(queryRoute, cdc),
		queryRewardsCmd(queryRoute, cdc),
		queryRewardFactorsCmd(queryRoute, cdc),
	)...)

	return incentiveQueryCmd
}

func queryRewardsCmd(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rewards",
		Short: "query claimable rewards",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query rewards with optional flags for owner and type

			Example:
			$ %s query %s rewards
			$ %s query %s rewards --owner kava15qdefkmwswysgg4qxgqpqr35k3m49pkx2jdfnw
			$ %s query %s rewards --type hard
			$ %s query %s rewards --type usdx-minting
			$ %s query %s rewards --type delegator
			$ %s query %s rewards --type swap
			$ %s query %s rewards --type hard --owner kava15qdefkmwswysgg4qxgqpqr35k3m49pkx2jdfnw
			$ %s query %s rewards --type hard --unsynced
			`,
				version.ClientName, types.ModuleName, version.ClientName, types.ModuleName,
				version.ClientName, types.ModuleName, version.ClientName, types.ModuleName,
				version.ClientName, types.ModuleName, version.ClientName, types.ModuleName,
				version.ClientName, types.ModuleName, version.ClientName, types.ModuleName)),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			page := viper.GetInt(flags.FlagPage)
			limit := viper.GetInt(flags.FlagLimit)
			strOwner := viper.GetString(flagOwner)
			strType := viper.GetString(flagType)
			boolUnsynced := viper.GetBool(flagUnsynced)

			// Prepare params for querier
			owner, err := sdk.AccAddressFromBech32(strOwner)
			if err != nil {
				return err
			}

			switch strings.ToLower(strType) {
			case typeHard:
				params := types.NewQueryRewardsParams(page, limit, owner, boolUnsynced)
				claims, err := executeHardRewardsQuery(queryRoute, cdc, cliCtx, params)
				if err != nil {
					return err
				}
				return cliCtx.PrintOutput(claims)
			case typeUSDXMinting:
				params := types.NewQueryRewardsParams(page, limit, owner, boolUnsynced)
				claims, err := executeUSDXMintingRewardsQuery(queryRoute, cdc, cliCtx, params)
				if err != nil {
					return err
				}
				return cliCtx.PrintOutput(claims)
			case typeDelegator:
				params := types.NewQueryRewardsParams(page, limit, owner, boolUnsynced)
				claims, err := executeDelegatorRewardsQuery(queryRoute, cdc, cliCtx, params)
				if err != nil {
					return err
				}
				return cliCtx.PrintOutput(claims)
			case typeSwap:
				params := types.NewQueryRewardsParams(page, limit, owner, boolUnsynced)
				claims, err := executeSwapRewardsQuery(queryRoute, cdc, cliCtx, params)
				if err != nil {
					return err
				}
				return cliCtx.PrintOutput(claims)
			default:
				params := types.NewQueryRewardsParams(page, limit, owner, boolUnsynced)

				hardClaims, err := executeHardRewardsQuery(queryRoute, cdc, cliCtx, params)
				if err != nil {
					return err
				}
				usdxMintingClaims, err := executeUSDXMintingRewardsQuery(queryRoute, cdc, cliCtx, params)
				if err != nil {
					return err
				}
				delegatorClaims, err := executeDelegatorRewardsQuery(queryRoute, cdc, cliCtx, params)
				if err != nil {
					return err
				}
				swapClaims, err := executeSwapRewardsQuery(queryRoute, cdc, cliCtx, params)
				if err != nil {
					return err
				}
				if len(hardClaims) > 0 {
					cliCtx.PrintOutput(hardClaims)
				}
				if len(usdxMintingClaims) > 0 {
					cliCtx.PrintOutput(usdxMintingClaims)
				}
				if len(delegatorClaims) > 0 {
					cliCtx.PrintOutput(delegatorClaims)
				}
				if len(swapClaims) > 0 {
					cliCtx.PrintOutput(swapClaims)
				}
			}
			return nil
		},
	}
	cmd.Flags().String(flagOwner, "", "(optional) filter by owner address")
	cmd.Flags().String(flagType, "", fmt.Sprintf("(optional) filter by a reward type: %s", strings.Join(rewardTypes, "|")))
	cmd.Flags().Bool(flagUnsynced, false, "(optional) get unsynced claims")
	cmd.Flags().Int(flags.FlagPage, 1, "pagination page rewards of to to query for")
	cmd.Flags().Int(flags.FlagLimit, 100, "pagination limit of rewards to query for")
	return cmd
}

func queryParamsCmd(queryRoute string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "params",
		Short: "get the incentive module parameters",
		Long:  "Get the current global incentive module parameters.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			// Query
			route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryGetParams)
			res, height, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}
			cliCtx = cliCtx.WithHeight(height)

			// Decode and print results
			var params types.Params
			if err := cdc.UnmarshalJSON(res, &params); err != nil {
				return fmt.Errorf("failed to unmarshal params: %w", err)
			}
			return cliCtx.PrintOutput(params)
		},
	}
}

func queryRewardFactorsCmd(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reward-factors",
		Short: "get current global reward factors",
		Long:  `Get current global reward factors for all reward types.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			// Execute query
			route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryGetRewardFactors)
			res, height, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}
			cliCtx = cliCtx.WithHeight(height)

			// Decode and print results
			var response types.QueryGetRewardFactorsResponse
			if err := cdc.UnmarshalJSON(res, &response); err != nil {
				return fmt.Errorf("failed to unmarshal reward factors: %w", err)
			}
			return cliCtx.PrintOutput(response)
		},
	}
	cmd.Flags().String(flagDenom, "", "(optional) filter reward factors by denom")
	return cmd
}

func executeHardRewardsQuery(queryRoute string, cdc *codec.Codec, cliCtx context.CLIContext, params types.QueryRewardsParams) (types.HardLiquidityProviderClaims, error) {
	bz, err := cdc.MarshalJSON(params)
	if err != nil {
		return types.HardLiquidityProviderClaims{}, err
	}

	route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryGetHardRewards)
	res, height, err := cliCtx.QueryWithData(route, bz)
	if err != nil {
		return types.HardLiquidityProviderClaims{}, err
	}

	cliCtx = cliCtx.WithHeight(height)

	var claims types.HardLiquidityProviderClaims
	if err := cdc.UnmarshalJSON(res, &claims); err != nil {
		return types.HardLiquidityProviderClaims{}, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return claims, nil
}

func executeUSDXMintingRewardsQuery(queryRoute string, cdc *codec.Codec, cliCtx context.CLIContext, params types.QueryRewardsParams) (types.USDXMintingClaims, error) {
	bz, err := cdc.MarshalJSON(params)
	if err != nil {
		return types.USDXMintingClaims{}, err
	}

	route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryGetUSDXMintingRewards)
	res, height, err := cliCtx.QueryWithData(route, bz)
	if err != nil {
		return types.USDXMintingClaims{}, err
	}

	cliCtx = cliCtx.WithHeight(height)

	var claims types.USDXMintingClaims
	if err := cdc.UnmarshalJSON(res, &claims); err != nil {
		return types.USDXMintingClaims{}, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return claims, nil
}

func executeDelegatorRewardsQuery(queryRoute string, cdc *codec.Codec, cliCtx context.CLIContext, params types.QueryRewardsParams) (types.DelegatorClaims, error) {
	bz, err := cdc.MarshalJSON(params)
	if err != nil {
		return types.DelegatorClaims{}, err
	}

	route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryGetDelegatorRewards)
	res, height, err := cliCtx.QueryWithData(route, bz)
	if err != nil {
		return types.DelegatorClaims{}, err
	}

	cliCtx = cliCtx.WithHeight(height)

	var claims types.DelegatorClaims
	if err := cdc.UnmarshalJSON(res, &claims); err != nil {
		return types.DelegatorClaims{}, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return claims, nil
}

func executeSwapRewardsQuery(queryRoute string, cdc *codec.Codec, cliCtx context.CLIContext, params types.QueryRewardsParams) (types.SwapClaims, error) {
	bz, err := cdc.MarshalJSON(params)
	if err != nil {
		return types.SwapClaims{}, err
	}

	route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryGetSwapRewards)
	res, height, err := cliCtx.QueryWithData(route, bz)
	if err != nil {
		return types.SwapClaims{}, err
	}

	cliCtx = cliCtx.WithHeight(height)

	var claims types.SwapClaims
	if err := cdc.UnmarshalJSON(res, &claims); err != nil {
		return types.SwapClaims{}, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return claims, nil
}
