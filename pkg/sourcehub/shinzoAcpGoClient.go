package sourcehub

import (
	"context"
	"fmt"
	"strings"
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
	documents := len(parentDocumentIds)
	if documents < 1 {
		return fmt.Errorf("Unable to create data feed: Must have at least one parent document")
	}

	err := client.Acp.SetActorRelationship(ctx, "view", documentId, "creator", creatorDid)
	if err != nil {
		return fmt.Errorf("Encountered error setting %s as the creator of view:%s: %v", creatorDid, documentId, err)
	}

	for _, parent := range parentDocumentIds {
		parentInfo := strings.Split(parent, ":")
		if len(parentInfo) != 2 {
			return fmt.Errorf("Received invalid parent document id: %s | expected in the form of resourceType:resourceName")
		}
		err = client.Acp.SetRelationship(ctx, "view", documentId, "parent", parentInfo[0], parentInfo[1])
		if err != nil {
			return fmt.Errorf("Encountered error setting %s as the parent of view:%s: %v", parent, documentId, err)
		}
	}

	return nil
}

func (client *ShinzoAcpGoClient) VerifyAccessRequest(ctx context.Context, policyID, resourceName, objectID, permission, actorDID string) (bool, error) {
	return client.Acp.VerifyAccessRequest(ctx, resourceName, objectID, permission, actorDID)
}

func (client *ShinzoAcpGoClient) GetActorDid() string {
	return client.Acp.GetActor().Did
}
