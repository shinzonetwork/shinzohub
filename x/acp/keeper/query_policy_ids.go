package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/acp_core/pkg/utils"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k Keeper) PolicyIds(goCtx context.Context, req *types.QueryPolicyIdsRequest) (*types.QueryPolicyIdsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine, err := k.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := engine.ListPolicies(ctx, &coretypes.ListPoliciesRequest{})
	if err != nil {
		return nil, err
	}

	return &types.QueryPolicyIdsResponse{
		Ids: utils.MapSlice(resp.Policies, func(p *coretypes.Policy) string { return p.Id }),
	}, nil
}
