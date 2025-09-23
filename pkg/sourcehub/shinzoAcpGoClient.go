package sourcehub

import (
	"context"
	"fmt"
	"strings"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

type ShinzoAcpGoClient struct {
	Acp AcpClient
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
		Acp: acp,
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
	return client.Acp.SetActorRelationship(ctx, "group", groupName, "guest", did)
}

func (client *ShinzoAcpGoClient) RemoveFromGroup(ctx context.Context, groupName string, did string) error {
	return client.Acp.DeleteActorRelationship(ctx, "group", groupName, "guest", did)
}

func (client *ShinzoAcpGoClient) BlockFromGroup(ctx context.Context, groupName string, did string) error {
	return client.Acp.SetActorRelationship(ctx, "group", groupName, "blocked", did)
}

func (client *ShinzoAcpGoClient) GiveQueryAccess(ctx context.Context, documentId string, did string) error {
	return client.Acp.SetActorRelationship(ctx, "view", documentId, "subscriber", did)
}

func (client *ShinzoAcpGoClient) BanUserFromView(ctx context.Context, documentId string, did string) error {
	return client.Acp.SetActorRelationship(ctx, "view", documentId, "banned", did)
}

func (client *ShinzoAcpGoClient) CreateDataFeed(ctx context.Context, documentId string, creatorDid string, parentDocumentIds ...string) error {
	if len(parentDocumentIds) < 1 {
		return createDataFeedError(documentId, creatorDid, fmt.Errorf("Must provide at lease one parent document id"))
	}

	creatorRel := coretypes.NewActorRelationship("view", documentId, "creator", creatorDid)
	creatorCmd := acptypes.NewSetRelationshipCmd(creatorRel)

	var parentCmds []*acptypes.PolicyCmd
	for _, parentId := range parentDocumentIds {
		parent := strings.Split(parentId, ":")
		if len(parent) != 2 {
			return createDataFeedError(documentId, creatorDid, fmt.Errorf("Invalid parentDocumentId encountered: %s ; must be in the form of resourceType:resourceName", parentId))
		}
		parentRel := coretypes.NewRelationship("view", documentId, "parent", parent[0], parent[1])
		parentCmd := acptypes.NewSetRelationshipCmd(parentRel)
		parentCmds = append(parentCmds, parentCmd)
	}

	allCmds := []*acptypes.PolicyCmd{creatorCmd}
	allCmds = append(allCmds, parentCmds...)

	return client.Acp.ExecutePolicyCommands(ctx, allCmds, func(e error) error {
		return createDataFeedError(documentId, creatorDid, e)
	})
}

func (client *ShinzoAcpGoClient) VerifyAccessRequest(ctx context.Context, policyID, resourceName, objectID, permission, actorDID string) (bool, error) {
	return client.Acp.VerifyAccessRequest(ctx, resourceName, objectID, permission, actorDID)
}

func (client *ShinzoAcpGoClient) GetActorDid() string {
	return client.Acp.GetActor().Did
}
