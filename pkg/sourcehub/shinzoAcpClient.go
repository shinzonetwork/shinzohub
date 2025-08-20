package sourcehub

import (
	"context"
	"crypto"
)

type ShinzoAcpClient interface {
	AddToGroup(ctx context.Context, groupName string, did string) error
	RemoveFromGroup(ctx context.Context, groupName string, did string) error
	BlockFromGroup(ctx context.Context, groupName, did string) error
	GiveQueryAccess(ctx context.Context, documentId string, did string) error
	BanUserFromResource(ctx context.Context, documentId string, did string) error
	CreateDataFeed(ctx context.Context, documentId string, creatorDid string, parentDocumentIds ...string) error
	VerifyAccessRequest(ctx context.Context, policyID, resourceName, objectID, permission, actorDID string) (bool, error)

	// Additional methods for test resource setup
	RegisterObject(ctx context.Context, resourceName, objectID string) error
	SetRelationship(ctx context.Context, resourceName, objectID, relation, subjectDID string) error
	SetGroupRelationship(ctx context.Context, resourceName, objectID, relation, groupName, groupRelation string) error
	SetParentRelationship(ctx context.Context, resourceName, objectID, parentResourceName, parentObjectID string) error
	GetSignerAddress() string
	GetSignerAccountAddress() string
	GetSigner() crypto.Signer
	SetActor(did string, signer crypto.Signer)
}
