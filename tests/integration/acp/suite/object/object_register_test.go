package object

import (
	"testing"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var policyDef string = `
name: policy
resources:
  resource:
    relations:
      owner:
        types:
          - actor
`

func TestRegisterObject_RegisteringNewObjectIsSucessful(t *testing.T) {
	ctx := test.NewTestCtx(t)

	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	bob := ctx.GetActor("bob")
	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   types.NewObject("resource", "foo"),
		Expected: &types.RegisterObjectCmdResult{
			Result: types.RegistrationResult_Registered,
			Record: &types.RelationshipRecord{
				//TxId:         "",
				Creator:      ctx.TxSigner.SourceHubAddr,
				PolicyId:     ctx.State.PolicyId,
				Actor:        bob.DID,
				Relationship: types.NewActorRelationship("resource", "foo", "owner", bob.DID),
				Archived:     false,
				CreationTime: test.TimeToProto(ctx.Timestamp),
			},
		},
	}
	a2.Run(ctx)

	/*
		event := &types.EventObjectRegistered{
			Actor:          did,
			PolicyId:       pol.Id,
			ObjectId:       "foo",
			ObjectResource: "resource",
		}
		testutil.AssertEventEmmited(t, ctx, event)
	*/
}

func TestRegisterObject_RegisteringObjectRegisteredToAnotherUserErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)

	// Given Bob as owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   types.NewObject("resource", "foo"),
	}
	a2.Run(ctx)

	// when Alice tries to register foo
	// then she is denied
	a2 = test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Actor:       ctx.GetActor("alice"),
		Object:      types.NewObject("resource", "foo"),
		ExpectedErr: types.ErrNotAuthorized,
	}
	a2.Run(ctx)
}

func TestRegisterObject_ReregisteringObjectOwnedByUserIsNoop(t *testing.T) {
	ctx := test.NewTestCtx(t)

	// Given Bob as owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   types.NewObject("resource", "foo"),
	}
	a2.Run(ctx)

	// when Bob tries to reregister foo
	// then the result is a noop
	bob := ctx.GetActor("bob")
	a2 = test.RegisterObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   types.NewObject("resource", "foo"),
		Expected: &types.RegisterObjectCmdResult{
			Result: types.RegistrationResult_NoOp,
			Record: &types.RelationshipRecord{
				//TxId:         "",
				Creator:      ctx.TxSigner.SourceHubAddr,
				PolicyId:     ctx.State.PolicyId,
				Actor:        bob.DID,
				Relationship: types.NewActorRelationship("resource", "foo", "owner", bob.DID),
				Archived:     false,
				CreationTime: test.TimeToProto(ctx.Timestamp),
			},
		},
	}
	a2.Run(ctx)
}

func TestRegisterObject_RegisteringAnotherUsersArchivedObjectErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)

	// Given Bob as previous owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   types.NewObject("resource", "foo"),
	}
	a2.Run(ctx)

	a3 := test.UnregisterObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   types.NewObject("resource", "foo"),
	}
	a3.Run(ctx)

	// when Alice tries to register foo
	// then she is denied
	action := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Actor:       ctx.GetActor("alice"),
		Object:      types.NewObject("resource", "foo"),
		ExpectedErr: types.ErrNotAuthorized,
	}
	action.Run(ctx)
}

func TestRegisterObject_RegisteringArchivedUserObjectUnarchivesObject(t *testing.T) {
	ctx := test.NewTestCtx(t)

	// Given Bob as previous owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)
	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   types.NewObject("resource", "foo"),
	}
	a2.Run(ctx)
	a3 := test.UnregisterObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   types.NewObject("resource", "foo"),
	}
	a3.Run(ctx)

	// when Bob tries to register foo, it is unarchived
	bob := ctx.GetActor("bob")
	action := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Actor:    bob,
		Object:   types.NewObject("resource", "foo"),
		Expected: &types.RegisterObjectCmdResult{
			Result: types.RegistrationResult_Unarchived,
			Record: &types.RelationshipRecord{
				//TxId:         "",
				Creator:      ctx.TxSigner.SourceHubAddr,
				PolicyId:     ctx.State.PolicyId,
				Actor:        bob.DID,
				Relationship: types.NewActorRelationship("resource", "foo", "owner", bob.DID),
				Archived:     false,
				CreationTime: test.TimeToProto(ctx.Timestamp),
			},
		},
	}
	action.Run(ctx)

	/*
		event := &types.EventObjectRegistered{
			Actor:          bobDID,
			PolicyId:       pol.Id,
			ObjectId:       "foo",
			ObjectResource: "resource",
		}
		testutil.AssertEventEmmited(t, ctx, event)
	*/
}

func TestRegisterObject_RegisteringObjectInAnUndefinedResourceErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)

	// Given Bob as previous owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)
	a2 := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Actor:       ctx.GetActor("bob"),
		Object:      types.NewObject("abc", "foo"),
		ExpectedErr: types.ErrAcpInput,
	}
	a2.Run(ctx)
}

func TestRegisterObject_RegisteringToUnknownPolicyReturnsError(t *testing.T) {
	ctx := test.NewTestCtx(t)

	// Given Bob as previous owner of foo
	a2 := test.RegisterObjectAction{
		PolicyId:    "abc1234",
		Actor:       ctx.GetActor("bob"),
		Object:      types.NewObject("resource", "foo"),
		ExpectedErr: types.ErrPolicyNotFound,
	}
	a2.Run(ctx)
}

func TestRegisterObject_BlankResourceErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)

	// Given Bob as previous owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)
	a2 := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Actor:       ctx.GetActor("bob"),
		Object:      types.NewObject("abc", "foo"),
		ExpectedErr: types.ErrAcpInput,
	}
	a2.Run(ctx)
}

func TestRegisterObject_BlankObjectIdErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)

	// Given Bob as previous owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)
	a2 := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Actor:       ctx.GetActor("bob"),
		Object:      types.NewObject("abc", ""),
		ExpectedErr: types.ErrAcpInput,
	}
	a2.Run(ctx)
}
