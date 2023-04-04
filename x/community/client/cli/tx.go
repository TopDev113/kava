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

	"github.com/kava-labs/kava/x/community/client/utils"
	"github.com/kava-labs/kava/x/community/types"
)

const (
	flagDeposit = "deposit"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	communityTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "community module transactions subcommands",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmds := []*cobra.Command{
		getCmdFundCommunityPool(),
	}

	for _, cmd := range cmds {
		flags.AddTxFlagsToCmd(cmd)
	}

	communityTxCmd.AddCommand(cmds...)

	return communityTxCmd
}

func getCmdFundCommunityPool() *cobra.Command {
	return &cobra.Command{
		Use:   "fund-community-pool [coins]",
		Short: "funds the community pool",
		Long:  "Fund community pool removes the listed coins from the sender's account and send them to the community module account.",
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(
			`%s tx %s fund-community-module 10000000ukava --from <key>`, version.AppName, types.ModuleName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coins, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgFundCommunityPool(clientCtx.GetFromAddress(), coins)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}
}

// NewCmdSubmitCommunityPoolLendDepositProposal implements the command to submit a community-pool lend deposit proposal
func NewCmdSubmitCommunityPoolLendDepositProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "community-pool-lend-deposit [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a community pool lend deposit proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a community pool lend deposit proposal along with an initial deposit.
The proposal details must be supplied via a JSON file.
Note that --deposit below is the initial proposal deposit submitted along with the proposal.
Example:
$ %s tx gov submit-proposal community-pool-lend-deposit <path/to/proposal.json> --deposit 1000000000ukava --from=<key_or_address>
Where proposal.json contains:
{
  "title": "Community Pool Deposit",
  "description": "Deposit some KAVA from community pool!",
  "amount": [
    {
      "denom": "ukava",
      "amount": "100000000000"
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
			// parse proposal
			proposal, err := utils.ParseCommunityPoolLendDepositProposal(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			deposit, err := parseInitialDeposit(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			msg, err := govv1beta1.NewMsgSubmitProposal(&proposal, deposit, from)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagDeposit, "", "Initial deposit for the proposal")

	return cmd
}

// NewCmdSubmitCommunityPoolLendWithdrawProposal implements the command to submit a community-pool lend withdraw proposal
func NewCmdSubmitCommunityPoolLendWithdrawProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "community-pool-lend-withdraw [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a community pool lend withdraw proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a community pool lend withdraw proposal along with an initial deposit.
The proposal details must be supplied via a JSON file.
Note that --deposit below is the initial proposal deposit submitted along with the proposal.
Example:
$ %s tx gov submit-proposal community-pool-lend-withdraw <path/to/proposal.json> --deposit 1000000000ukava --from=<key_or_address>
Where proposal.json contains:
{
  "title": "Community Pool Withdrawal",
  "description": "Withdraw some KAVA from community pool!",
  "amount": [
    {
      "denom": "ukava",
      "amount": "100000000000"
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
			// parse proposal
			proposal, err := utils.ParseCommunityPoolLendWithdrawProposal(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			deposit, err := parseInitialDeposit(cmd)
			if err != nil {
				return err
			}
			from := clientCtx.GetFromAddress()
			msg, err := govv1beta1.NewMsgSubmitProposal(&proposal, deposit, from)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagDeposit, "", "Initial deposit for the proposal")

	return cmd
}

func parseInitialDeposit(cmd *cobra.Command) (sdk.Coins, error) {
	// parse initial deposit
	depositStr, err := cmd.Flags().GetString(flagDeposit)
	if err != nil {
		return nil, fmt.Errorf("no initial deposit found. did you set --deposit? %s", err)
	}
	deposit, err := sdk.ParseCoinsNormalized(depositStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse deposit: %s", err)
	}
	if !deposit.IsValid() || deposit.IsZero() {
		return nil, fmt.Errorf("no initial deposit set, use --deposit flag")
	}
	return deposit, nil
}
