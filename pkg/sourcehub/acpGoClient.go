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
	signer             sdk.TxSigner
	policyId           string
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

func createDataFeedError(documentId, creatorDid string, err error) error {
	return fmt.Errorf("Encountered an error creating data view for document %s by creator %s: %w", documentId, creatorDid, err)
}

func sendAndConfirmTx(ctx context.Context, acp *sdk.Client, txBuilder *sdk.TxBuilder, signer sdk.TxSigner, msgSet *sdk.MsgSet, decorateError func(error) error) error {
	tx, err := txBuilder.Build(ctx, signer, msgSet)
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

func NewAcpGoClient(acp *sdk.Client, txBuilder *sdk.TxBuilder, signer sdk.TxSigner, policyId string) *AcpGoClient {
	return &AcpGoClient{
		acp:                acp,
		transactionBuilder: txBuilder,
		signer:             signer,
		policyId:           policyId,
	}
}

func (client *AcpGoClient) AddToGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "guest", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return addToGroupError(did, groupName, err)
	}
	cmdBuilder.Actor(did)
	cmdBuilder.PolicyID(client.policyId)
	cmdBuilder.PolicyCmd(cmd)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return addToGroupError(did, groupName, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(did, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)
	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, client.signer, &msgSet, func(e error) error { return addToGroupError(did, groupName, e) })
}

func (client *AcpGoClient) RemoveFromGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "guest", did)
	cmd := acptypes.NewDeleteRelationshipCmd(rel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return removeFromGroupError(did, groupName, err)
	}
	cmdBuilder.Actor(did)
	cmdBuilder.PolicyID(client.policyId)
	cmdBuilder.PolicyCmd(cmd)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return removeFromGroupError(did, groupName, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(did, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)
	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, client.signer, &msgSet, func(e error) error { return removeFromGroupError(did, groupName, e) })
}

func (client *AcpGoClient) BlockFromGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "blocked", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return addToGroupError(did, groupName, err)
	}
	cmdBuilder.Actor(did)
	cmdBuilder.PolicyID(client.policyId)
	cmdBuilder.PolicyCmd(cmd)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return addToGroupError(did, groupName, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(did, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)
	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, client.signer, &msgSet, func(e error) error { return addToGroupError(did, groupName, e) })
}

func (client *AcpGoClient) GiveQueryAccess(ctx context.Context, documentId string, did string) error {
	rel := coretypes.NewActorRelationship("file", documentId, "subscriber", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return giveQueryAccessError(did, documentId, err)
	}
	cmdBuilder.Actor(did)
	cmdBuilder.PolicyID(client.policyId)
	cmdBuilder.PolicyCmd(cmd)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return giveQueryAccessError(did, documentId, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(did, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)
	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, client.signer, &msgSet, func(e error) error { return giveQueryAccessError(did, documentId, e) })
}

func (client *AcpGoClient) BanUserFromResource(ctx context.Context, documentId string, did string) error {
	rel := coretypes.NewActorRelationship("file", documentId, "banned", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return giveQueryAccessError(did, documentId, err)
	}
	cmdBuilder.Actor(did)
	cmdBuilder.PolicyID(client.policyId)
	cmdBuilder.PolicyCmd(cmd)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return giveQueryAccessError(did, documentId, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(did, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)
	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, client.signer, &msgSet, func(e error) error { return giveQueryAccessError(did, documentId, e) })
}

func (client *AcpGoClient) CreateDataFeed(ctx context.Context, documentId string, creatorDid string, parentDocumentIds ...string) error {
	creatorRel := coretypes.NewActorRelationship("file", documentId, "creator", creatorDid)
	creatorCmd := acptypes.NewSetRelationshipCmd(creatorRel)

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return createDataFeedError(documentId, creatorDid, err)
	}
	cmdBuilder.Actor(creatorDid)
	cmdBuilder.PolicyID(client.policyId)
	cmdBuilder.PolicyCmd(creatorCmd)

	for _, parentId := range parentDocumentIds {
		parentRel := coretypes.NewActorRelationship("file", documentId, "parent", parentId)
		parentCmd := acptypes.NewSetRelationshipCmd(parentRel)
		cmdBuilder.PolicyCmd(parentCmd)
	}

	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return createDataFeedError(documentId, creatorDid, err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(creatorDid, jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)

	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, client.signer, &msgSet, func(e error) error {
		return createDataFeedError(documentId, creatorDid, e)
	})
}
