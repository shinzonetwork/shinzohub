package keeper

import (
	"encoding/base64"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
	"github.com/shinzonetwork/shinzohub/x/view/types"
)

type AckCallback struct {
	keeper Keeper
}

func NewAckCallback(k Keeper) AckCallback {
	return AckCallback{keeper: k}
}

func (c AckCallback) OnPacketAck(ctx sdk.Context, req sourcehubtypes.PendingICARequest) error {
	if req.Kind != sourcehubtypes.RequestKind_REQUEST_KIND_REGISTER_OBJECT {
		return nil
	}

	var meta sourcehubtypes.RegisterObjectMeta
	if err := c.keeper.cdc.Unmarshal(req.Meta, &meta); err != nil {
		return fmt.Errorf("decode RegisterObjectMeta: %w", err)
	}
	if meta.ResourceName != sourcehubtypes.ViewResourceName {
		return nil
	}

	address := meta.ObjectId
	pending, found, err := c.keeper.GetPendingView(ctx, address)
	if err != nil {
		return fmt.Errorf("read pending view %s: %w", address, err)
	}
	if !found {
		return nil
	}

	switch req.Status {
	case sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS:
		if err := c.keeper.SetView(ctx, pending); err != nil {
			return fmt.Errorf("promote pending view: %w", err)
		}
		_ = c.keeper.DeletePendingView(ctx, address)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeViewRegistered,
			sdk.NewAttribute(types.AttrKeyAddress, address),
			sdk.NewAttribute(types.AttrKeyCreator, pending.Creator),
			sdk.NewAttribute(types.AttrKeyName, pending.Name),
			sdk.NewAttribute(types.AttrKeyData, base64.StdEncoding.EncodeToString(pending.Data)),
		))

	case sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE, sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT:
		_ = c.keeper.DeletePendingView(ctx, address)

		eventType := types.EventTypeViewRegistrationFailed
		if req.Status == sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT {
			eventType = types.EventTypeViewRegistrationTimedOut
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			eventType,
			sdk.NewAttribute(types.AttrKeyAddress, address),
			sdk.NewAttribute(types.AttrKeyCreator, pending.Creator),
			sdk.NewAttribute(types.AttrKeyName, pending.Name),
			sdk.NewAttribute(types.AttrKeyError, req.Error),
		))
	}
	return nil
}
