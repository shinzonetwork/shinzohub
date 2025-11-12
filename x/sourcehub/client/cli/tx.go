package cli

import (
	"fmt"
	"strconv"
	"strings"

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
	cmd.AddCommand(CmdRegisterObjects())

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
			streamID := args[1]
			did := args[2]

			resourceInt, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid resource: %w", err)
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRequestStreamAccess{
				Signer:   clientCtx.GetFromAddress().String(),
				Resource: types.Resource(resourceInt),
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

func CmdRegisterObjects() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-objects [resources...]",
		Short: "Register one or more shinzo objects by resource name(s)",
		Long:  "Examples:\n  shinzohubd tx sourcehub register-objects block logs event --from acc0 --yes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Normalize + dedupe
			seen := map[string]struct{}{}
			var resources []string
			for _, a := range args {
				r := strings.TrimSpace(strings.ToLower(a))
				if r == "" {
					continue
				}
				if _, ok := seen[r]; ok {
					continue
				}
				seen[r] = struct{}{}
				resources = append(resources, r)
			}

			if len(resources) == 0 {
				return fmt.Errorf("at least one resource is required")
			}

			msg := &types.MsgRegisterShinzoObjects{
				Signer:    clientCtx.GetFromAddress().String(),
				Resources: resources,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	// add standard tx flags: --from, --fees, --gas, --gas-adjustment, --yes, etc.
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
