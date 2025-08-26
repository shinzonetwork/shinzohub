package sourcehub

import (
	"context"
	"crypto"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/sourcenetwork/sourcehub/sdk"
)

type AcpActor struct {
	Did    string
	Signer crypto.Signer
}

type AcpClient interface {
	RegisterObject(ctx context.Context, resourceType, resourceName string) error
	SetActorRelationship(ctx context.Context, resourceType, resourceName, relation, actorDid string) error
	SetRelationship(ctx context.Context, resourceType, resourceName, relation, subjectResourceType, subjectResourceName string) error
	SetActorSetRelationship(ctx context.Context, resourceType, resourceName, relation, subjectResourceType, subjectResourceName, subjectRelation string) error
	DeleteActorRelationship(ctx context.Context, resourceType, resourceName, relation, actorDid string) error
	DeleteRelationship(ctx context.Context, resourceType, resourceName, relation, subjectResourceType, subjectResourceName string) error
	DeleteActorSetRelationship(ctx context.Context, resourceType, resourceName, relation, subjectResourceType, subjectResourceName, subjectRelation string) error
	GetActor() AcpActor
	SetActor(actor *AcpActor)
	GetSigner() sdk.TxSigner
	VerifyAccessRequest(ctx context.Context, resourceType, resourceName, permission, actorDid string) (bool, error)
	GetBalanceInUOpen(ctx context.Context) (*banktypes.QueryBalanceResponse, error)
	FundAccount(ctx context.Context, fundingAccountAlias string, fundingAmount uint64) error
}
