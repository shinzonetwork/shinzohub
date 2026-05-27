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
