package sourcehub

import (
	"errors"

	"github.com/sourcenetwork/sourcehub/sdk"
)

type AcpGoClient struct {
	acp                *sdk.Client
	transactionBuilder *sdk.TxBuilder
}

func (client *AcpGoClient) AddToGroup(groupName string, did string) error {
	return errors.New("Not implemented")
}

func (client *AcpGoClient) RemoveFromGroup(groupName string, did string) error {
	return errors.New("Not implemented")
}

func (client *AcpGoClient) GiveQueryAccess(documentId string, did string) error {
	return errors.New("Not implemented")
}
