package object

import (
	"testing"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var unregisterTestPol = `
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
`

func setupUnregister(t *testing.T) *test.TestCtx {
	ctx := test.NewTestCtx(t)
	a1 := test.CreatePolicyAction{
		Creator: ctx.TxSigner,
		Policy:  unregisterTestPol,
	}
	a1.Run(ctx)
	a2 := test.RegisterObjectAction{
		PolicyId: ctx.State.PolicyId,
		Object:   types.NewObject("file", "foo"),
		Actor:    ctx.GetActor("alice"),
	}
	a2.Run(ctx)
	a3 := test.SetRelationshipAction{
		Actor:        ctx.GetActor("alice"),
		PolicyId:     ctx.State.PolicyId,
		Relationship: types.NewActorRelationship("file", "foo", "reader", ctx.GetActor("alice").DID),
	}
	a3.Run(ctx)
	return ctx
}

func TestUnregisterObject_RegisteredObjectCanBeUnregisteredByAuthor(t *testing.T) {
	ctx := setupUnregister(t)

	action := test.UnregisterObjectAction{
		PolicyId: ctx.State.PolicyId,
		Object:   types.NewObject("file", "foo"),
		Actor:    ctx.GetActor("alice"),
		Expected: &types.UnregisterObjectCmdResult{
			Found:                true,
			RelationshipsRemoved: 2,
		},
	}
	action.Run(ctx)
}

func TestUnregisterObject_ActorCannotUnregisterObjectTheyDoNotOwn(t *testing.T) {
	ctx := setupUnregister(t)

	action := test.UnregisterObjectAction{
		PolicyId:    ctx.State.PolicyId,
		Object:      types.NewObject("file", "foo"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: types.ErrNotAuthorized,
	}
	action.Run(ctx)
}

func TestUnregisterObject_UnregisteringAnObjectThatDoesNotExistReturnsUnauthorized(t *testing.T) {
	ctx := setupUnregister(t)

	action := test.UnregisterObjectAction{
		PolicyId:    ctx.State.PolicyId,
		Object:      types.NewObject("file", "file-isnt-registerd"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: types.ErrNotAuthorized,
	}
	action.Run(ctx)
}

func TestUnregisterObject_UnregisteringAnAlreadyArchivedObjectIsANoop(t *testing.T) {
	ctx := setupUnregister(t)

	action := test.UnregisterObjectAction{
		PolicyId: ctx.State.PolicyId,
		Object:   types.NewObject("file", "foo"),
		Actor:    ctx.GetActor("alice"),
	}
	action.Run(ctx)

	action = test.UnregisterObjectAction{
		PolicyId: ctx.State.PolicyId,
		Object:   types.NewObject("file", "foo"),
		Actor:    ctx.GetActor("alice"),
		Expected: &types.UnregisterObjectCmdResult{
			Found: true,
		},
	}
	action.Run(ctx)
}

func TestUnregisterObject_SendingInvalidPolicyIdErrors(t *testing.T) {
	ctx := setupUnregister(t)

	action := test.UnregisterObjectAction{
		PolicyId:    "abc1234",
		Object:      types.NewObject("file", "foo"),
		Actor:       ctx.GetActor("alice"),
		ExpectedErr: types.ErrPolicyNotFound,
	}
	action.Run(ctx)
}
