package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	hubtypes "github.com/sourcenetwork/sourcehub/types"
	"github.com/sourcenetwork/sourcehub/x/acp/auth_engine"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/relationship"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k msgServer) DirectPolicyCmd(goCtx context.Context, msg *types.MsgDirectPolicyCmd) (*types.MsgDirectPolicyCmdResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine, err := k.GetZanziEngine(ctx)
	if err != nil {
		return nil, err
	}
	addr, err := hubtypes.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("DirectPolicyCmd: %v: %w", err, types.ErrInvalidAccAddr)
	}

	acc := k.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return nil, fmt.Errorf("DirectPolicyCmd: %w", types.ErrAccNotFound)
	}

	actorID, err := did.IssueDID(acc)
	if err != nil {
		return nil, fmt.Errorf("DirectPolicyCmd: could not issue did to creator %v: %w", msg.Creator, types.ErrAcpInput)
	}

	rec, err := engine.GetPolicy(goCtx, msg.PolicyId)
	if err != nil {
		return nil, err
	} else if rec == nil {
		return nil, fmt.Errorf("DirectPolicyCmd: policy %v: %w", msg.PolicyId, types.ErrPolicyNotFound)
	}

	authorizer := relationship.NewRelationshipAuthorizer(engine)

	cmdResult := &types.PolicyCmdResult{}
	policy := rec.Policy

	switch c := msg.Cmd.Cmd.(type) {
	case *types.PolicyCmd_SetRelationshipCmd:
		var found auth_engine.RecordFound
		var record *types.RelationshipRecord

		cmd := relationship.SetRelationshipCommand{
			Policy:       policy,
			CreationTs:   msg.CreationTime,
			Creator:      msg.Creator,
			Relationship: c.SetRelationshipCmd.Relationship,
			Actor:        actorID,
		}
		found, record, err = cmd.Execute(ctx, engine, authorizer)
		if err != nil {
			err = fmt.Errorf("set relationship cmd: %w", err)
			break
		}

		cmdResult.Result = &types.PolicyCmdResult_SetRelationshipResult{
			SetRelationshipResult: &types.SetRelationshipCmdResult{
				RecordExisted: bool(found),
				Record:        record,
			},
		}
	case *types.PolicyCmd_DeleteRelationshipCmd:
		var found auth_engine.RecordFound

		cmd := relationship.DeleteRelationshipCommand{
			Policy:       policy,
			Actor:        actorID,
			Relationship: c.DeleteRelationshipCmd.Relationship,
		}
		found, err = cmd.Execute(ctx, engine, authorizer)
		if err != nil {
			err = fmt.Errorf("delete relationship cmd: %w", err)
			break
		}

		cmdResult.Result = &types.PolicyCmdResult_DeleteRelationshipResult{
			DeleteRelationshipResult: &types.DeleteRelationshipCmdResult{
				RecordFound: bool(found),
			},
		}
	case *types.PolicyCmd_RegisterObjectCmd:
		var registrationResult types.RegistrationResult
		var record *types.RelationshipRecord

		cmd := relationship.RegisterObjectCommand{
			Policy:     policy,
			CreationTs: msg.CreationTime,
			Creator:    msg.Creator,
			Registration: &types.Registration{
				Object: c.RegisterObjectCmd.Object,
				Actor: &types.Actor{
					Id: actorID,
				},
			},
		}
		registrationResult, record, err = cmd.Execute(ctx, engine, ctx.EventManager())
		if err != nil {
			err = fmt.Errorf("register object cmd: %w", err)
			break
		}

		cmdResult.Result = &types.PolicyCmdResult_RegisterObjectResult{
			RegisterObjectResult: &types.RegisterObjectCmdResult{
				Result: registrationResult,
				Record: record,
			},
		}
	case *types.PolicyCmd_UnregisterObjectCmd:
		var count uint

		cmd := relationship.UnregisterObjectCommand{
			Policy: policy,
			Object: c.UnregisterObjectCmd.Object,
			Actor:  actorID,
		}
		count, err = cmd.Execute(ctx, engine, authorizer)
		if err != nil {
			err = fmt.Errorf("unregister object cmd: %w", err)
			break
		}

		cmdResult.Result = &types.PolicyCmdResult_UnregisterObjectResult{
			UnregisterObjectResult: &types.UnregisterObjectCmdResult{
				Found:                true, //TODO true,
				RelationshipsRemoved: uint64(count),
			},
		}
	default:
		err = fmt.Errorf("unsuported command %v: %w", c, types.ErrInvalidVariant)
	}

	if err != nil {
		return nil, fmt.Errorf("DirectPolicyCmd failed: %w", err)

	}

	return &types.MsgDirectPolicyCmdResponse{
		Result: cmdResult,
	}, nil
}
