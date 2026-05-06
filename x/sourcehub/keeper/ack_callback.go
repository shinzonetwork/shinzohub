package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

type AckCallback struct {
	keeper Keeper
}

func NewAckCallback(k Keeper) AckCallback {
	return AckCallback{keeper: k}
}

func (c AckCallback) OnPacketAck(ctx sdk.Context, req types.PendingICARequest) error {
	if req.Kind != types.RequestKind_REQUEST_KIND_REGISTER_SHINZO_POLICY {
		return nil
	}

	if req.Status != types.RequestStatus_REQUEST_STATUS_SUCCESS {
		return nil
	}

	if len(req.MsgResponses) == 0 {
		return fmt.Errorf("CreatePolicy ack missing MsgResponses")
	}
	var resp acptypes.MsgCreatePolicyResponse
	if err := gogoproto.Unmarshal(req.MsgResponses[0], &resp); err != nil {
		return fmt.Errorf("decode MsgCreatePolicyResponse: %w", err)
	}
	if resp.Record == nil || resp.Record.Policy == nil || resp.Record.Policy.Id == "" {
		return fmt.Errorf("CreatePolicy ack missing policy id")
	}
	c.keeper.SetPolicyId(ctx, resp.Record.Policy.Id)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypePolicyCreated,
		sdk.NewAttribute(types.AttrKeyPolicyID, resp.Record.Policy.Id),
		sdk.NewAttribute(types.AttrKeyRequestor, req.Requestor),
	))
	return nil
}
