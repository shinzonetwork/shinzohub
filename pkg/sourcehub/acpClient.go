package sourcehub

import "context"

type AcpClient interface {
	AddToGroup(ctx context.Context, groupName string, did string) error
	RemoveFromGroup(ctx context.Context, groupName string, did string) error
	BlockFromGroup(ctx context.Context, groupName, did string) error
	GiveQueryAccess(ctx context.Context, documentId string, did string) error
	BanUserFromResource(ctx context.Context, documentId string, did string) error
	CreateDataFeed(ctx context.Context, documentId string, creatorDid string, parentDocumentIds ...string) error
}
