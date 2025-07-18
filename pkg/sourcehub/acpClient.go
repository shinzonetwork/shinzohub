package sourcehub

type AcpClient interface {
	AddToGroup(groupName string, did string) error
	RemoveFromGroup(groupName string, did string) error
	GiveQueryAccess(documentId string, did string) error
}
