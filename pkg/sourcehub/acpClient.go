package sourcehub

import "context"

type AcpClient interface {
	AddToGroup(ctx context.Context, groupName string, did string) error
	RemoveFromGroup(ctx context.Context, groupName string, did string) error
	GiveQueryAccess(ctx context.Context, documentId string, did string) error
}
