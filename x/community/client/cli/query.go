package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/kava-labs/kava/x/community/types"
)

// GetQueryCmd returns the cli query commands for the community module.
func GetQueryCmd() *cobra.Command {
	communityQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the community module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	commands := []*cobra.Command{
		GetCmdQueryBalance(),
	}

	for _, cmd := range commands {
		flags.AddQueryFlagsToCmd(cmd)
	}

	communityQueryCmd.AddCommand(commands...)

	return communityQueryCmd
}

// GetCmdQueryBalance implements a command to return the current community pool balance.
func GetCmdQueryBalance() *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Query the current balance of the community module account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Balance(cmd.Context(), &types.QueryBalanceRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
}
