package sourcehub

import (
	"context"
	"fmt"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/sdk"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

type AcpGoClient struct {
	acp                *sdk.Client
	transactionBuilder *sdk.TxBuilder
}

// Helper: look up policy ID by group name (assumes group name == policy name)
func (client *AcpGoClient) lookupPolicyIdByGroupName(ctx context.Context, groupName string) (string, error) {
	acpQuery := client.acp.ACPQueryClient()
	resp, err := acpQuery.PolicyIds(ctx, &acptypes.QueryPolicyIdsRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to query policy ids: %w", err)
	}
	for _, id := range resp.Ids {
		// Fetch policy by ID and check name
		polResp, err := acpQuery.Policy(ctx, &acptypes.QueryPolicyRequest{Id: id})
		if err != nil {
			continue
		}
		if polResp.Record != nil && polResp.Record.Policy != nil && polResp.Record.Policy.Name == groupName {
			return id, nil
		}
	}
	return "", fmt.Errorf("policy with group name '%s' not found", groupName)
}

func addToGroupError(did, groupName string, err error) error {
	return fmt.Errorf("Encountered an error adding %s to %s group: %w", did, groupName, err)
}

func removeFromGroupError(did, groupName string, err error) error {
	return fmt.Errorf("Encountered an error removing %s from %s group: %w", did, groupName, err)
}

func giveQueryAccessError(did, documentId string, err error) error {
	return fmt.Errorf("Encountered an error giving query access to %s for document %s: %w", did, documentId, err)
}

func sendAndConfirmTx(ctx context.Context, acp *sdk.Client, txBuilder *sdk.TxBuilder, msgSet *sdk.MsgSet, decorateError func(error) error) error {
	tx, err := txBuilder.Build(ctx, nil, msgSet) // nil signer placeholder
	if err != nil {
		return decorateError(err)
	}
	resp, err := acp.BroadcastTx(ctx, tx)
	if err != nil {
		return decorateError(fmt.Errorf("Error sending transaction: %w", err))
	}
	result, err := acp.AwaitTx(ctx, resp.TxHash)
	if err != nil {
		return decorateError(fmt.Errorf("Error waiting for transaction: %w", err))
	}
	if execErr := result.Error(); execErr != nil {
		return decorateError(fmt.Errorf("Transaction failed: %w", execErr))
	}
	return nil
}

func (client *AcpGoClient) AddToGroup(groupName string, did string) error {
	ctx := context.Background()
	policyId, err := client.lookupPolicyIdByGroupName(ctx, groupName)
	if err != nil {
		return addToGroupError(did, groupName, err)
	}

	rel := coretypes.NewActorRelationship("group", groupName, "member", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return addToGroupError(did, groupName, err)
	}
	cmdBuilder.Actor(did)
	cmdBuilder.PolicyID(policyId)
	cmdBuilder.PolicyCmd(cmd)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return addToGroupError(did, groupName, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(did, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)
	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, &msgSet, func(e error) error { return addToGroupError(did, groupName, e) })
}

func (client *AcpGoClient) RemoveFromGroup(groupName string, did string) error {
	ctx := context.Background()
	policyId, err := client.lookupPolicyIdByGroupName(ctx, groupName)
	if err != nil {
		return removeFromGroupError(did, groupName, err)
	}

	rel := coretypes.NewActorRelationship("group", groupName, "member", did)
	cmd := acptypes.NewDeleteRelationshipCmd(rel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return removeFromGroupError(did, groupName, err)
	}
	cmdBuilder.Actor(did)
	cmdBuilder.PolicyID(policyId)
	cmdBuilder.PolicyCmd(cmd)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return removeFromGroupError(did, groupName, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(did, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)
	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, &msgSet, func(e error) error { return removeFromGroupError(did, groupName, e) })
}

func (client *AcpGoClient) GiveQueryAccess(documentId string, did string) error {
	ctx := context.Background()
	policyId, err := client.lookupPolicyIdByGroupName(ctx, "default-document-policy")
	if err != nil {
		return giveQueryAccessError(did, documentId, err)
	}

	rel := coretypes.NewActorRelationship("file", documentId, "reader", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return giveQueryAccessError(did, documentId, err)
	}
	cmdBuilder.Actor(did)
	cmdBuilder.PolicyID(policyId)
	cmdBuilder.PolicyCmd(cmd)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return giveQueryAccessError(did, documentId, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(did, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)
	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, &msgSet, func(e error) error { return giveQueryAccessError(did, documentId, e) })
}
