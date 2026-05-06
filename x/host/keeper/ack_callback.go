package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/host/types"
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

type AckCallback struct {
	keeper Keeper
}

func NewAckCallback(k Keeper) AckCallback {
	return AckCallback{keeper: k}
}

func (c AckCallback) OnPacketAck(ctx sdk.Context, req sourcehubtypes.PendingICARequest) error {
	if req.Kind != sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP {
		return nil
	}

	var meta sourcehubtypes.SetRelationshipMeta
	if err := c.keeper.cdc.Unmarshal(req.Meta, &meta); err != nil {
		return fmt.Errorf("decode SetRelationshipMeta: %w", err)
	}
	if meta.Group != sourcehubtypes.GroupHostName {
		return nil
	}

	store := runtime.KVStoreAdapter(c.keeper.storeService.OpenKVStore(ctx))
	didBytes := []byte(meta.Did)
	callerAddr := store.Get(append([]byte(types.PendingDIDAddrPrefix), didBytes...))
	if len(callerAddr) == 0 {
		return nil
	}
	bech32Addr := sdk.AccAddress(callerAddr).String()

	pending, found, err := c.keeper.GetPendingHost(ctx, bech32Addr)
	if err != nil {
		return fmt.Errorf("read pending host %s: %w", bech32Addr, err)
	}

	switch req.Status {
	case sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS:
		if found {
			if err := c.keeper.SetHost(ctx, pending); err != nil {
				return fmt.Errorf("promote pending host: %w", err)
			}
			_ = c.keeper.DeletePendingHost(ctx, bech32Addr)
		}
		store.Set(append([]byte(types.AddrDIDPrefix), callerAddr...), didBytes)
		store.Set(append([]byte(types.DIDAddrPrefix), didBytes...), callerAddr)
		store.Delete(append([]byte(types.PendingAddrDIDPrefix), callerAddr...))
		store.Delete(append([]byte(types.PendingDIDAddrPrefix), didBytes...))

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHostRegistered,
			sdk.NewAttribute(types.AttrKeyAddress, bech32Addr),
			sdk.NewAttribute(types.AttrKeyDID, meta.Did),
		))

	case sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE, sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT:
		_ = c.keeper.DeletePendingHost(ctx, bech32Addr)
		store.Delete(append([]byte(types.PendingAddrDIDPrefix), callerAddr...))
		store.Delete(append([]byte(types.PendingDIDAddrPrefix), didBytes...))

		eventType := types.EventTypeHostRegistrationFailed
		if req.Status == sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT {
			eventType = types.EventTypeHostRegistrationTimedOut
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			eventType,
			sdk.NewAttribute(types.AttrKeyAddress, bech32Addr),
			sdk.NewAttribute(types.AttrKeyDID, meta.Did),
			sdk.NewAttribute(types.AttrKeyError, req.Error),
		))
	}
	return nil
}
