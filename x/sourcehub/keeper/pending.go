package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (k Keeper) SetPendingRequest(ctx sdk.Context, req types.PendingICARequest) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&req)
	if err != nil {
		return fmt.Errorf("marshal PendingICARequest: %w", err)
	}
	store.Set(types.PendingRequestKey(req.PortId, req.ChannelId, req.Sequence), bz)
	if req.Requestor != "" {
		store.Set(types.PendingByRequestorKey(req.Requestor, req.Sequence), []byte(fmt.Sprintf("%s/%s/%d", req.PortId, req.ChannelId, req.Sequence)))
	}
	return nil
}

func (k Keeper) GetPendingRequest(ctx sdk.Context, portID, channelID string, sequence uint64) (types.PendingICARequest, bool, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.PendingRequestKey(portID, channelID, sequence))
	if len(bz) == 0 {
		return types.PendingICARequest{}, false, nil
	}
	var req types.PendingICARequest
	if err := k.cdc.Unmarshal(bz, &req); err != nil {
		return types.PendingICARequest{}, false, fmt.Errorf("unmarshal PendingICARequest: %w", err)
	}
	return req, true, nil
}

func (k Keeper) ResolvePendingRequest(
	ctx sdk.Context,
	portID, channelID string,
	sequence uint64,
	status types.RequestStatus,
	errMsg string,
	msgResponses [][]byte,
) (types.PendingICARequest, bool, error) {
	req, found, err := k.GetPendingRequest(ctx, portID, channelID, sequence)
	if err != nil || !found {
		return types.PendingICARequest{}, found, err
	}
	req.Status = status
	req.Error = errMsg
	req.ResolvedAt = ctx.BlockTime().Unix()
	req.MsgResponses = msgResponses

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&req)
	if err != nil {
		return types.PendingICARequest{}, true, fmt.Errorf("marshal resolved PendingICARequest: %w", err)
	}
	store.Set(types.PendingRequestKey(portID, channelID, sequence), bz)
	return req, true, nil
}

func (k Keeper) IteratePendingByRequestor(ctx sdk.Context, requestor string, cb func(types.PendingICARequest) bool) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	idx := prefix.NewStore(store, types.PendingByRequestorPrefixKey(requestor))
	iter := idx.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var portID, channelID string
		var sequence uint64
		if _, err := fmt.Sscanf(string(iter.Value()), "%[^/]/%[^/]/%d", &portID, &channelID, &sequence); err != nil {
			continue
		}
		req, found, err := k.GetPendingRequest(ctx, portID, channelID, sequence)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if stop := cb(req); stop {
			return nil
		}
	}
	return nil
}

func NewPendingICARequest(
	portID, channelID string,
	sequence uint64,
	kind types.RequestKind,
	requestor string,
	submittedAt time.Time,
	meta []byte,
) types.PendingICARequest {
	return types.PendingICARequest{
		Sequence:    sequence,
		PortId:      portID,
		ChannelId:   channelID,
		Kind:        kind,
		Requestor:   requestor,
		SubmittedAt: submittedAt.Unix(),
		Status:      types.RequestStatus_REQUEST_STATUS_PENDING,
		Meta:        meta,
	}
}
