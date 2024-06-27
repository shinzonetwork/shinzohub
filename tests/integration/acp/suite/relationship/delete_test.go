package relationship

import (
	"testing"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var deletePolicy = `
name: policy
resources:
  file:
    relations:
      owner:
        types:
          - actor
      reader:
        types:
          - actor
      writer:
        types:
          - actor
      admin:
        types:
          - actor
        manages:
          - reader
`

func setupDelete(t *testing.T) *test.TestCtx {
	ctx := test.NewTestCtx(t)

	reader := ctx.GetActor("reader")
	writer := ctx.GetActor("writer")
	admin := ctx.GetActor("admin")
	action := test.PolicySetupAction{
		Policy:        deletePolicy,
		PolicyCreator: ctx.TxSigner,
		ObjectsPerActor: map[string][]*types.Object{
			"alice": []*types.Object{
				types.NewObject("file", "foo"),
			},
		},
		RelationshipsPerActor: map[string][]*types.Relationship{
			"alice": []*types.Relationship{
				types.NewActorRelationship("file", "foo", "reader", reader.DID),
				types.NewActorRelationship("file", "foo", "writer", writer.DID),
				types.NewActorRelationship("file", "foo", "admin", admin.DID),
			},
		},
	}
	action.Run(ctx)

	return ctx
}

func TestDeleteRelationship_ObjectOwnerCanRemoveRelationship(t *testing.T) {
	ctx := setupDelete(t)

	action := test.DeleteRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: types.NewActorRelationship("file", "foo", "reader", ctx.GetActor("reader").DID),
		Actor:        ctx.GetActor("alice"),
		Expected: &types.DeleteRelationshipCmdResult{
			RecordFound: true,
		},
	}
	action.Run(ctx)
}

func TestDeleteRelationship_ObjectManagerCanRemoveRelationshipsForRelationTheyManage(t *testing.T) {
	ctx := setupDelete(t)

	action := test.DeleteRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: types.NewActorRelationship("file", "foo", "reader", ctx.GetActor("reader").DID),
		Actor:        ctx.GetActor("admin"),
		Expected: &types.DeleteRelationshipCmdResult{
			RecordFound: true,
		},
	}
	action.Run(ctx)
}

func TestDeleteRelationship_ObjectManagerCannotRemoveRelationshipForRelationTheyDontManage(t *testing.T) {
	ctx := setupDelete(t)
	action := test.DeleteRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: types.NewActorRelationship("file", "foo", "writer", ctx.GetActor("writer").DID),
		Actor:        ctx.GetActor("admin"),
		ExpectedErr:  types.ErrNotAuthorized,
	}
	action.Run(ctx)
}
