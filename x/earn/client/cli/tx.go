package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/kava-labs/kava/x/earn/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	earnTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmds := []*cobra.Command{
		getCmdDeposit(),
		getCmdWithdraw(),
	}

	for _, cmd := range cmds {
		flags.AddTxFlagsToCmd(cmd)
	}

	earnTxCmd.AddCommand(cmds...)

	return earnTxCmd
}

func getCmdDeposit() *cobra.Command {
	return &cobra.Command{
		Use:   "deposit [amount] [strategy]",
		Short: "deposit coins to an earn vault",
		Example: fmt.Sprintf(
			`%s tx %s deposit 10000000ukava hard --from <key>`,
			version.AppName, types.ModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			strategy := types.NewStrategyTypeFromString(args[1])
			if !strategy.IsValid() {
				return fmt.Errorf("invalid strategy type: %s", args[1])
			}

			signer := clientCtx.GetFromAddress()
			msg := types.NewMsgDeposit(signer.String(), amount, strategy)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
}

func getCmdWithdraw() *cobra.Command {
	return &cobra.Command{
		Use:   "withdraw [amount] [strategy]",
		Short: "withdraw coins from an earn vault",
		Example: fmt.Sprintf(
			`%s tx %s withdraw 10000000ukava hard --from <key>`,
			version.AppName, types.ModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			strategy := types.NewStrategyTypeFromString(args[1])
			if !strategy.IsValid() {
				return fmt.Errorf("invalid strategy type: %s", args[1])
			}

			fromAddr := clientCtx.GetFromAddress()
			msg := types.NewMsgWithdraw(fromAddr.String(), amount, strategy)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
}

// GetCmdSubmitCommunityPoolDepositProposal implements the command to submit a community-pool deposit proposal
func GetCmdSubmitCommunityPoolDepositProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "community-pool-deposit [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a community pool deposit proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a community pool deposit proposal along with an initial deposit.
The proposal details must be supplied via a JSON file.
Example:
$ %s tx gov submit-proposal community-pool-deposit <path/to/proposal.json> --from=<key_or_address>
Where proposal.json contains:
{
  "title": "Community Pool Deposit",
  "description": "Deposit some KAVA from community pool!",
  "amount": 
  	{
			"denom": "ukava",
			"amount": "100000000000"
	},
	"deposit": [
		{
			"denom": "ukava",
			"amount": "1000000000"
		}
	]
}
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposal, err := ParseCommunityPoolDepositProposalJSON(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			content := types.NewCommunityPoolDepositProposal(proposal.Title, proposal.Description, proposal.Amount)
			msg, err := govv1beta1.NewMsgSubmitProposal(content, proposal.Deposit, from)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	return cmd
}

// GetCmdSubmitCommunityPoolWithdrawProposal implements the command to submit a community-pool withdraw proposal
func GetCmdSubmitCommunityPoolWithdrawProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "community-pool-withdraw [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a community pool withdraw proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a community pool withdraw proposal along with an initial deposit.
The proposal details must be supplied via a JSON file.
Example:
$ %s tx gov submit-proposal community-pool-withdraw <path/to/proposal.json> --from=<key_or_address>
Where proposal.json contains:
{
  "title": "Community Pool Withdraw",
  "description": "Withdraw some KAVA from community pool!",
  "amount": 
  	{
			"denom": "ukava",
			"amount": "100000000000"
	},
	"deposit": [
		{
			"denom": "ukava",
			"amount": "1000000000"
		}
	]
}
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposal, err := ParseCommunityPoolWithdrawProposalJSON(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			content := types.NewCommunityPoolWithdrawProposal(proposal.Title, proposal.Description, proposal.Amount)
			msg, err := govv1beta1.NewMsgSubmitProposal(content, proposal.Deposit, from)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	return cmd
}
