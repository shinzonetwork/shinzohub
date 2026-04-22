package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func emitRequestPending(ctx sdk.Context, req types.PendingICARequest) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRequestPending,
		sdk.NewAttribute(types.AttrKeySequence, fmt.Sprintf("%d", req.Sequence)),
		sdk.NewAttribute(types.AttrKeyPortID, req.PortId),
		sdk.NewAttribute(types.AttrKeyChannelID, req.ChannelId),
		sdk.NewAttribute(types.AttrKeyRequestKind, req.Kind.String()),
		sdk.NewAttribute(types.AttrKeyRequestor, req.Requestor),
	))
}

func emitRequestResolved(ctx sdk.Context, req types.PendingICARequest) {
	eventType := types.EventTypeRequestAcknowledged
	switch req.Status {
	case types.RequestStatus_REQUEST_STATUS_FAILURE:
		eventType = types.EventTypeRequestFailed
	case types.RequestStatus_REQUEST_STATUS_TIMEOUT:
		eventType = types.EventTypeRequestTimedOut
	}
	attrs := []sdk.Attribute{
		sdk.NewAttribute(types.AttrKeySequence, fmt.Sprintf("%d", req.Sequence)),
		sdk.NewAttribute(types.AttrKeyPortID, req.PortId),
		sdk.NewAttribute(types.AttrKeyChannelID, req.ChannelId),
		sdk.NewAttribute(types.AttrKeyRequestKind, req.Kind.String()),
		sdk.NewAttribute(types.AttrKeyRequestor, req.Requestor),
	}
	if req.Error != "" {
		attrs = append(attrs, sdk.NewAttribute(types.AttrKeyError, req.Error))
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(eventType, attrs...))
}
