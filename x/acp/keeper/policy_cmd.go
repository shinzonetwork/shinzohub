package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/sourcenetwork/acp_core/pkg/auth"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func dispatchPolicyCmd(ctx sdk.Context, engine coretypes.ACPEngineServer, policyId string, authenticatedActor string, ts *prototypes.Timestamp, cmd *types.PolicyCmd) (*types.PolicyCmdResult, error) {
	var err error
	result := &types.PolicyCmdResult{}

	principal, err := auth.NewDIDPrincipal(authenticatedActor)
	if err != nil {
		return nil, err
	}
	goCtx := auth.InjectPrincipal(ctx.Context(), principal)

	switch c := cmd.Cmd.(type) {
	case *types.PolicyCmd_SetRelationshipCmd:
		resp, respErr := engine.SetRelationship(goCtx, &coretypes.SetRelationshipRequest{
			PolicyId:     policyId,
			CreationTime: ts,
			Relationship: c.SetRelationshipCmd.Relationship,
		})
		if respErr != nil {
			err = respErr
			break
		}

		result.Result = &types.PolicyCmdResult_SetRelationshipResult{
			SetRelationshipResult: &types.SetRelationshipCmdResult{
				RecordExisted: resp.RecordExisted,
				Record:        resp.Record,
			},
		}
	case *types.PolicyCmd_DeleteRelationshipCmd:
		resp, respErr := engine.DeleteRelationship(goCtx, &coretypes.DeleteRelationshipRequest{
			PolicyId:     policyId,
			Relationship: c.DeleteRelationshipCmd.Relationship,
		})
		if respErr != nil {
			err = respErr
			break
		}

		result.Result = &types.PolicyCmdResult_DeleteRelationshipResult{
			DeleteRelationshipResult: &types.DeleteRelationshipCmdResult{
				RecordFound: resp.RecordFound,
			},
		}
	case *types.PolicyCmd_RegisterObjectCmd:
		resp, respErr := engine.RegisterObject(goCtx, &coretypes.RegisterObjectRequest{
			PolicyId:     policyId,
			CreationTime: ts,
			Object:       c.RegisterObjectCmd.Object,
		})
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_RegisterObjectResult{
			RegisterObjectResult: &types.RegisterObjectCmdResult{
				Result: resp.Result,
				Record: resp.Record,
			},
		}
	case *types.PolicyCmd_UnregisterObjectCmd:
		resp, respErr := engine.UnregisterObject(goCtx, &coretypes.UnregisterObjectRequest{
			PolicyId: policyId,
			Object:   c.UnregisterObjectCmd.Object,
		})
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_UnregisterObjectResult{
			UnregisterObjectResult: &types.UnregisterObjectCmdResult{
				Found:                resp.Found,
				RelationshipsRemoved: resp.RelationshipsRemoved,
			},
		}
	default:
		err = errors.Wrap("unsuported command", errors.ErrUnknownVariant, errors.Pair("command", c))
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}
