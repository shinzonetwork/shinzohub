package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/acp/bearer_token"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k msgServer) BearerPolicyCmd(goCtx context.Context, msg *types.MsgBearerPolicyCmd) (*types.MsgBearerPolicyCmdResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine, err := k.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	resolver := &did.KeyResolver{}
	actorID, err := bearer_token.AuthorizeMsg(ctx, resolver, msg, ctx.BlockTime())
	if err != nil {
		return nil, err
	}

	result, err := dispatchPolicyCmd(ctx, engine, msg.PolicyId, actorID, msg.CreationTime, msg.Cmd)

	if err != nil {
		return nil, fmt.Errorf("PolicyCmd failed: %w", err)
	}

	return &types.MsgBearerPolicyCmdResponse{
		Result: result,
	}, nil
}
