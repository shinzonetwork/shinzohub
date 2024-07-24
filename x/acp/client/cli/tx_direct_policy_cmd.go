package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var _ = strconv.Itoa(0)

func directDispatcher(cmd *cobra.Command, polId string, polCmd *types.PolicyCmd) error {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	creator := clientCtx.GetFromAddress().String()
	msg := types.NewMsgDirectPolicyCmdNow(creator, polId, polCmd)
	return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
}

func CmdDirectPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "direct-policy-cmd",
		Short: "Broadcast a DirectPolicyCommand msg",
	}

	cmd.AddCommand(CmdRegisterObject(directDispatcher))
	cmd.AddCommand(CmdUnregisterObject(directDispatcher))
	cmd.AddCommand(CmdSetRelationship(directDispatcher))
	cmd.AddCommand(CmdDeleteRelationship(directDispatcher))
	return cmd
}
