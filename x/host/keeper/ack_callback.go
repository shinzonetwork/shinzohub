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

	addr, found := c.keeper.GetPendingAddressForDID(ctx, meta.Did)
	if !found {
		return nil
	}
	bech32Addr := addr.String()

	store := runtime.KVStoreAdapter(c.keeper.storeService.OpenKVStore(ctx))
	pendingAddrKey := addrIndexKey(types.PendingAddrDIDPrefix, bech32Addr)
	pendingDidKey := didIndexKey(types.PendingDIDAddrPrefix, meta.Did)

	pending, foundPending, err := c.keeper.GetPendingHost(ctx, bech32Addr)
	if err != nil {
		return fmt.Errorf("read pending host %s: %w", bech32Addr, err)
	}

	switch req.Status {
	case sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS:
		if foundPending {
			if err := c.keeper.SetHost(ctx, pending); err != nil {
				return fmt.Errorf("promote pending host: %w", err)
			}
			_ = c.keeper.DeletePendingHost(ctx, bech32Addr)
		}
		store.Set(addrIndexKey(types.AddrDIDPrefix, bech32Addr), []byte(meta.Did))
		store.Set(didIndexKey(types.DIDAddrPrefix, meta.Did), []byte(bech32Addr))
		store.Delete(pendingAddrKey)
		store.Delete(pendingDidKey)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHostRegistered,
			sdk.NewAttribute(types.AttrKeyAddress, bech32Addr),
			sdk.NewAttribute(types.AttrKeyDID, meta.Did),
		))

	case sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE, sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT:
		_ = c.keeper.DeletePendingHost(ctx, bech32Addr)
		store.Delete(pendingAddrKey)
		store.Delete(pendingDidKey)

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
