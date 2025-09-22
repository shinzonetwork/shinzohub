package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdRegisterSourcehubICA())
	cmd.AddCommand(CmdRequestStreamAccess())
	cmd.AddCommand(CmdRegisterShinzoPolicy())

	return cmd
}

func CmdRegisterSourcehubICA() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-ica [controller-connection-id] [host-connection-id]",
		Short: "Register an interchain account for the sourcehub module",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			controllerConnectionID := args[0]
			hostConnectionID := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRegisterSourcehubICA{
				Signer:                 clientCtx.GetFromAddress().String(),
				ControllerConnectionId: controllerConnectionID,
				HostConnectionId:       hostConnectionID,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRequestStreamAccess() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-stream [resource] [stream-id] [did]",
		Short: "Request access to a stream by providing a stream ID and a DID",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[0]
			streamID := args[1]
			did := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRequestStreamAccess{
				Signer:   clientCtx.GetFromAddress().String(),
				Resource: resource,
				StreamId: streamID,
				Did:      did,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRegisterShinzoPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-policy",
		Short: "Register shinzohub default policy to sourcehub",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRegisterShinzoPolicy{
				Signer: clientCtx.GetFromAddress().String(),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
