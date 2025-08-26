package sourcehub

import (
	"context"
)

type ShinzoAcpClient interface {
	AddToGroup(ctx context.Context, groupName string, did string) error
	RemoveFromGroup(ctx context.Context, groupName string, did string) error
	BlockFromGroup(ctx context.Context, groupName, did string) error
	GiveQueryAccess(ctx context.Context, documentId string, did string) error
	BanUserFromView(ctx context.Context, documentId string, did string) error
	CreateDataFeed(ctx context.Context, documentId string, creatorDid string, parentDocumentIds ...string) error
	VerifyAccessRequest(ctx context.Context, policyID, resourceName, objectID, permission, actorDID string) (bool, error)
	GetActorDid() string
}
