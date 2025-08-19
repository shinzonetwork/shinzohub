package sourcehub

import (
	"context"
	"crypto"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/sourcenetwork/acp_core/pkg/did"
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

func newAcpGoClient(acp *sdk.Client, txBuilder *sdk.TxBuilder, signer sdk.TxSigner, acpSigner crypto.Signer, acpDID string, policyID string) (*AcpGoClient, error) {
	return &AcpGoClient{
		acp:                acp,
		transactionBuilder: txBuilder,
		signer:             signer,
		acpSigner:          acpSigner,
		acpDID:             acpDID,
		policyId:           policyID,
	}, nil
}

func CreateAcpGoClient(chainId string) (AcpClient, error) {
	signer, err := NewApiSignerFromEnv()
	if err != nil {
		return nil, fmt.Errorf("Failed to load API signer: %v", err)
	}

	return createAcpGoClient(chainId, signer)
}

func createAcpGoClient(chainId string, signer sdk.TxSigner) (AcpClient, error) {
	acpClient, err := sdk.NewClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create ACP SDK client: %v", err)
	}
	txBuilder, err := sdk.NewTxBuilder(
		sdk.WithSDKClient(acpClient),
		sdk.WithChainID(chainId),
		sdk.WithFeeAmount(300),
		sdk.WithGasLimit(300000))
	if err != nil {
		return nil, fmt.Errorf("Failed to create TxBuilder: %v", err)
	}

	policyId := os.Getenv("POLICY_ID")
	if policyId == "" {
		return nil, fmt.Errorf("POLICY_ID environment variable is required")
	}

	acpDID, acpSigner, err := did.ProduceDID() // Todo figure out some way to fix this - it gives me a random did each time instead of one derived from our signer TxSigner
	if err != nil {
		return nil, fmt.Errorf("Failed to create ACP DID and signer: %v", err)
	}

	acpGoClient, err := newAcpGoClient(acpClient, &txBuilder, signer, acpSigner, acpDID, policyId)
	if err != nil {
		return nil, fmt.Errorf("Failed to create ACP Go client: %v", err)
	}
	return acpGoClient, nil
}

func CreateAcpGoClientWithValidatorSender(chainId string) (AcpClient, error) {
	signer, err := getValidatorSigner()
	if err != nil {
		return nil, fmt.Errorf("Failed to load API signer: %v", err)
	}

	return createAcpGoClient(chainId, signer)
}

func getValidatorSigner() (sdk.TxSigner, error) {
	// Create a keyring to access the validator account
	reg := cdctypes.NewInterfaceRegistry()
	cryptocdc.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)

	// Use the test keyring backend and the .sourcehub directory
	kr, err := keyring.New("sourcehub", keyring.BackendTest, os.Getenv("HOME")+"/.sourcehub", nil, cdc)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	// Get the validator signer
	validatorSigner, err := sdk.NewTxSignerFromKeyringKey(kr, "validator")
	if err != nil {
		return nil, fmt.Errorf("failed to get validator signer: %w", err)
	}

	return validatorSigner, nil
}

func (client *AcpGoClient) AddToGroup(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "guest", did)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return addToGroupError(did, groupName, e)
	})
}

func (client *AcpGoClient) MakeGroupAdmin(ctx context.Context, groupName string, did string) error {
	rel := coretypes.NewActorRelationship("group", groupName, "admin", did)
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

func (client *AcpGoClient) IsObjectRegistered(ctx context.Context, resourceName, objectID string) (bool, error) {
	// Create the object owner query request
	queryRequest := &acptypes.QueryObjectOwnerRequest{
		PolicyId: client.policyId,
		Object:   coretypes.NewObject(resourceName, objectID),
	}

	// Use the SourceHub ACP query client to check if the object is registered
	result, err := client.acp.ACPQueryClient().ObjectOwner(ctx, queryRequest)
	if err != nil {
		return false, fmt.Errorf("failed to query object owner: %w", err)
	}

	// Log the owner of the resource
	if result.IsRegistered {
		if result.Record != nil && result.Record.Metadata != nil {
			fmt.Printf("Object %s:%s is registered with owner: %s\n", resourceName, objectID, result.Record.Metadata.OwnerDid)
		} else {
			fmt.Printf("Object %s:%s is registered but owner information is not available\n", resourceName, objectID)
		}
	} else {
		fmt.Printf("Object %s:%s is not registered\n", resourceName, objectID)
	}

	return result.IsRegistered, nil
}

func (client *AcpGoClient) RegisterObject(ctx context.Context, resourceName, objectID string) error {
	// Check if the object is already registered
	isRegistered, err := client.IsObjectRegistered(ctx, resourceName, objectID)
	if err != nil {
		return fmt.Errorf("failed to check if object %s is registered: %w", objectID, err)
	}

	if isRegistered {
		// Object is already registered, exit early - attempting to register it again will throw an error
		return nil
	}

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

func (client *AcpGoClient) SetGroupRelationship(ctx context.Context, resourceName, objectID, relation, groupName, groupRelation string) error {
	rel := coretypes.NewActorSetRelationship(resourceName, objectID, relation, "group", groupName, groupRelation)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set group relationship %s on %s for group %s: %w", relation, objectID, groupName, e)
	})
}

func (client *AcpGoClient) SetParentRelationship(ctx context.Context, resourceName, objectID, parentResourceName, parentObjectID string) error {
	rel := coretypes.NewRelationship(resourceName, objectID, "parent", parentResourceName, parentObjectID)
	cmd := acptypes.NewSetRelationshipCmd(rel)

	return client.executePolicyCommand(ctx, []*acptypes.PolicyCmd{cmd}, func(e error) error {
		return fmt.Errorf("failed to set parent relationship on %s:%s -> %s:%s: %w", resourceName, objectID, parentResourceName, parentObjectID, e)
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
	return client.acpDID, nil
}
