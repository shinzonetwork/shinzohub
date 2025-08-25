package sourcehub

import (
	"context"
	"crypto"
	"fmt"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

type ShinzoAcpGoClient struct {
	acp AcpClient
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

func newShinzoAcpGoClient(acp AcpClient) (*ShinzoAcpGoClient, error) {
	return &ShinzoAcpGoClient{
		acp: acp,
	}, nil
}

func CreateShinzoAcpGoClient(chainId string) (ShinzoAcpClient, error) {
	acp, err := CreateAcpGoClientFromEnvironmentVariable(chainId, "SHINZOHUB_PRIVATE_KEY")
	if err != nil {
		return nil, fmt.Errorf("Failed to create Acp Client: %v", err)
	}

	return createShinzoAcpGoClient(acp)
}

func createShinzoAcpGoClient(acp AcpClient) (ShinzoAcpClient, error) {
	return newShinzoAcpGoClient(acp)
}

func CreateShinzoAcpGoClientWithValidatorSender(chainId string) (ShinzoAcpClient, error) {
	acp, err := CreateAcpGoClientFromKeychain(chainId, "validator")
	if err != nil {
		return nil, fmt.Errorf("Failed to create Acp Client: %v", err)
	}

	return createShinzoAcpGoClient(acp)
}

func (client *ShinzoAcpGoClient) AddToGroup(ctx context.Context, groupName string, did string) error {
	return client.acp.SetActorRelationship(ctx, "group", groupName, "guest", did)
}

func (client *ShinzoAcpGoClient) MakeGroupAdmin(ctx context.Context, groupName string, did string) error {
	return client.acp.SetActorRelationship(ctx, "group", groupName, "admin", did)
}

func (client *ShinzoAcpGoClient) RemoveFromGroup(ctx context.Context, groupName string, did string) error {
	return client.acp.DeleteActorRelationship(ctx, "group", groupName, "guest", did)
}

func (client *ShinzoAcpGoClient) BlockFromGroup(ctx context.Context, groupName string, did string) error {
	return client.acp.SetActorRelationship(ctx, "group", groupName, "blocked", did)
}

func (client *ShinzoAcpGoClient) GiveQueryAccess(ctx context.Context, documentId string, did string) error {
	return client.acp.SetActorRelationship(ctx, "view", documentId, "subscriber", did)
}

func (client *ShinzoAcpGoClient) BanUserFromView(ctx context.Context, documentId string, did string) error {
	return client.acp.SetActorRelationship(ctx, "view", documentId, "banned", did)
}

func (client *ShinzoAcpGoClient) CreateDataFeed(ctx context.Context, documentId string, creatorDid string, parentDocumentIds ...string) error {
	// Create the main creator relationship command
	creatorRel := coretypes.NewActorRelationship("view", documentId, "creator", creatorDid)
	creatorCmd := acptypes.NewSetRelationshipCmd(creatorRel)

	// Create parent relationship commands if any
	var parentCmds []*acptypes.PolicyCmd
	for _, parentId := range parentDocumentIds {
		parentRel := coretypes.NewActorRelationship("view", documentId, "parent", parentId)
		parentCmd := acptypes.NewSetRelationshipCmd(parentRel)
		parentCmds = append(parentCmds, parentCmd)
	}

	// Combine all commands into a single slice
	allCmds := []*acptypes.PolicyCmd{creatorCmd}
	allCmds = append(allCmds, parentCmds...)

	return client.acp.ExecutePolicyCommands(ctx, allCmds, func(e error) error {
		return createDataFeedError(documentId, creatorDid, e)
	})
}

func (client *ShinzoAcpGoClient) VerifyAccessRequest(ctx context.Context, policyID, resourceName, objectID, permission, actorDID string) (bool, error) {
	return client.acp.VerifyAccessRequest(ctx, resourceName, objectID, permission, actorDID)
}

func (client *ShinzoAcpGoClient) RegisterObject(ctx context.Context, resourceName, objectID string) error {
	return client.acp.RegisterObject(ctx, resourceName, objectID)
}

func (client *ShinzoAcpGoClient) SetRelationship(ctx context.Context, resourceName, objectID, relation, subjectDID string) error {
	return client.acp.SetActorRelationship(ctx, resourceName, objectID, relation, subjectDID)
}

func (client *ShinzoAcpGoClient) SetGroupRelationship(ctx context.Context, resourceName, objectID, relation, groupName, groupRelation string) error {
	return client.acp.SetActorSetRelationship(ctx, resourceName, objectID, relation, "group", groupName, groupRelation)
}

func (client *ShinzoAcpGoClient) SetParentRelationship(ctx context.Context, resourceName, objectID, parentResourceName, parentObjectID string) error {
	return client.acp.SetRelationship(ctx, resourceName, objectID, "parent", parentResourceName, parentObjectID)
}

func (client *ShinzoAcpGoClient) GetSignerAddress() string {
	// Return the ACP DID directly
	return client.acp.GetActor().Did
}

// GetSignerAccountAddress returns the Cosmos account address of the signer
func (client *ShinzoAcpGoClient) GetSignerAccountAddress() string {
	return client.acp.GetSigner().GetAccAddress()
}

func (client *ShinzoAcpGoClient) GetSignerDid() (string, error) {
	return client.acp.GetActor().Did, nil
}

func (client *ShinzoAcpGoClient) GetSigner() crypto.Signer {
	return client.acp.GetActor().Signer
}

func (client *ShinzoAcpGoClient) SetActor(did string, signer crypto.Signer) {
	client.acp.SetActor(&AcpActor{Did: did, Signer: signer})
}
