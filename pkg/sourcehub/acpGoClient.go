package sourcehub

import (
	"context"
	"crypto"
	"fmt"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/sdk"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

type AcpGoClient struct {
	acp                *sdk.Client
	transactionBuilder *sdk.TxBuilder
	signer             sdk.TxSigner
	acpSigner          crypto.Signer
	acpDID             string
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

func (client *AcpGoClient) executePolicyCommand(ctx context.Context, cmds []*acptypes.PolicyCmd, decorateError func(error) error) error {
	if len(cmds) == 0 {
		return decorateError(fmt.Errorf("no policy commands provided"))
	}

	signerDID, err := client.GetSignerDid()
	if err != nil {
		return decorateError(fmt.Errorf("failed to get signer DID: %w", err))
	}

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client.acp)
	if err != nil {
		return decorateError(err)
	}

	cmdBuilder.Actor(signerDID)
	cmdBuilder.PolicyID(client.policyId)

	// Add all commands
	for _, cmd := range cmds {
		cmdBuilder.PolicyCmd(cmd)
	}

	cmdBuilder.SetSigner(client.acpSigner)

	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		return decorateError(err)
	}

	msg := acptypes.NewMsgSignedPolicyCmdFromJWS(client.signer.GetAccAddress(), jws)
	msgSet := sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(msg)

	return sendAndConfirmTx(ctx, client.acp, client.transactionBuilder, client.signer, &msgSet, decorateError)
}

func NewAcpGoClient(acp *sdk.Client, txBuilder *sdk.TxBuilder, signer sdk.TxSigner, acpSigner crypto.Signer, acpDID string, policyID string) (*AcpGoClient, error) {
	return &AcpGoClient{
		acp:                acp,
		transactionBuilder: txBuilder,
		signer:             signer,
		acpSigner:          acpSigner,
		acpDID:             acpDID,
		policyId:           policyID,
	}, nil
}

func (client *AcpGoClient) AddToGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "guest", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return addToGroupError(did, groupName, e)
	})
}

func (client *AcpGoClient) RemoveFromGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "guest", did)
	cmd := acptypes.NewDeleteRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return removeFromGroupError(did, groupName, e)
	})
}

func (client *AcpGoClient) BlockFromGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "blocked", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return addToGroupError(did, groupName, e)
	})
}

func (client *AcpGoClient) GiveQueryAccess(ctx context.Context, documentId string, did string) error {
	rel := coretypes.NewActorRelationship("file", documentId, "subscriber", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return giveQueryAccessError(did, documentId, e)
	})
}

func (client *AcpGoClient) BanUserFromResource(ctx context.Context, documentId string, did string) error {
	rel := coretypes.NewActorRelationship("file", documentId, "banned", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return giveQueryAccessError(did, documentId, e)
	})
}

func (client *AcpGoClient) CreateDataFeed(ctx context.Context, documentId string, creatorDid string, parentDocumentIds ...string) error {
	// Create the main creator relationship command
	creatorRel := coretypes.NewActorRelationship("file", documentId, "creator", creatorDid)
	creatorCmd := acptypes.NewSetRelationshipCmd(creatorRel)

	// Create parent relationship commands if any
	var parentCmds []*acptypes.PolicyCmd
	for _, parentId := range parentDocumentIds {
		parentRel := coretypes.NewActorRelationship("file", documentId, "parent", parentId)
		parentCmd := acptypes.NewSetRelationshipCmd(parentRel)
		parentCmds = append(parentCmds, parentCmd)
	}

	// Combine all commands into a single slice
	allCmds := []*acptypes.PolicyCmd{creatorCmd}
	allCmds = append(allCmds, parentCmds...)

	return client.executePolicyCommand(ctx, allCmds, func(e error) error {
		return createDataFeedError(documentId, creatorDid, e)
	})
}

func (client *AcpGoClient) VerifyAccessRequest(ctx context.Context, policyID, resourceName, objectID, permission, actorDID string) (bool, error) {
	// Create the access request using SourceHub types
	accessRequest := &acptypes.QueryVerifyAccessRequestRequest{
		PolicyId: policyID,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject(resourceName, objectID),
					Permission: permission,
				},
			},
			Actor: &coretypes.Actor{
				Id: actorDID,
			},
		},
	}

	// Use the SourceHub ACP query client to verify the access request
	result, err := client.acp.ACPQueryClient().VerifyAccessRequest(ctx, accessRequest)
	if err != nil {
		return false, fmt.Errorf("failed to verify access request: %w", err)
	}

	return result.Valid, nil
}

func (client *AcpGoClient) RegisterObject(ctx context.Context, resourceName, objectID string) error {
	// Create a register object command
	cmd := acptypes.NewRegisterObjectCmd(coretypes.NewObject(resourceName, objectID))

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to register object %s: %w", objectID, e)
	})
}

func (client *AcpGoClient) SetRelationship(ctx context.Context, resourceName, objectID, relation, subjectDID string) error {
	// Create a set relationship command
	rel := coretypes.NewActorRelationship(resourceName, objectID, relation, subjectDID)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set relationship %s on %s: %w", relation, objectID, e)
	})
}

func (client *AcpGoClient) GetSignerAddress() string {
	// Return the ACP DID directly
	return client.acpDID
}

// GetSignerAccountAddress returns the Cosmos account address of the signer
func (client *AcpGoClient) GetSignerAccountAddress() string {
	return client.signer.GetAccAddress()
}

func (client *AcpGoClient) GetSignerDid() (string, error) {
	// Return the stored ACP DID
	return client.acpDID, nil
}
