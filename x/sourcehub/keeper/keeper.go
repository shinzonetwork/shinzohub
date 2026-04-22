package keeper

import (
	"fmt"
	"time"

	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeService  storetypes.KVStoreService
	IcaCtrlKeeper types.ICAControllerKeeper
	adminKeeper   types.AdminKeeper
	ackCallbacks  *ackCallbackRegistry
}

type ackCallbackRegistry struct {
	callbacks map[types.RequestKind][]types.PacketAckCallback
}

func (r *ackCallbackRegistry) register(kind types.RequestKind, cb types.PacketAckCallback) {
	r.callbacks[kind] = append(r.callbacks[kind], cb)
}

func (r *ackCallbackRegistry) lookup(kind types.RequestKind) []types.PacketAckCallback {
	return r.callbacks[kind]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	icaCtrlKeeper types.ICAControllerKeeper,
	adminKeeper types.AdminKeeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		IcaCtrlKeeper: icaCtrlKeeper,
		adminKeeper:   adminKeeper,
		ackCallbacks:  &ackCallbackRegistry{callbacks: make(map[types.RequestKind][]types.PacketAckCallback)},
	}
}

func (k Keeper) RegisterAckCallback(kind types.RequestKind, cb types.PacketAckCallback) {
	k.ackCallbacks.register(kind, cb)
}

func (k Keeper) GetAckCallbacks(kind types.RequestKind) []types.PacketAckCallback {
	return k.ackCallbacks.lookup(kind)
}

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) CheckICAReady(ctx sdk.Context) error {
	connectionID := k.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return fmt.Errorf("no connection ID set in module state")
	}
	portID := fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())
	if addr, _ := k.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID); addr == "" {
		return fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}
	if chanID, ok := k.IcaCtrlKeeper.GetActiveChannelID(ctx, connectionID, portID); !ok || chanID == "" {
		return fmt.Errorf("no active ICA channel for portID %s on connection %s", portID, connectionID)
	}
	if k.GetPolicyId(ctx) == "" {
		return fmt.Errorf("no policy ID set in module state")
	}
	return nil
}

func (k Keeper) SendICASetRelationship(ctx sdk.Context, did string, group string, requestor string) (seq uint64, portID, channelID string, err error) {
	connectionID := k.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return 0, "", "", fmt.Errorf("no connection ID set in module state")
	}

	portID = fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := k.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return 0, "", "", fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	channelID, hasChannel := k.IcaCtrlKeeper.GetActiveChannelID(ctx, connectionID, portID)
	if !hasChannel || channelID == "" {
		return 0, "", "", fmt.Errorf("no active ICA channel for portID %s on connection %s", portID, connectionID)
	}

	policyId := k.GetPolicyId(ctx)
	if policyId == "" {
		return 0, "", "", fmt.Errorf("no policy ID set in module state")
	}

	cmd := acptypes.NewMsgDirectPolicyCmd(
		addr,
		policyId,
		acptypes.NewSetRelationshipCmd(coretypes.NewActorRelationship("group", group, "guest", did)),
	)

	anyMsg, err := codectypes.NewAnyWithValue(cmd)
	if err != nil {
		return 0, "", "", err
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return 0, "", "", err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())

	seq, err = k.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)
	if err != nil {
		return 0, "", "", err
	}

	meta := &types.SetRelationshipMeta{Did: did, Group: group}
	metaBz, _ := k.cdc.Marshal(meta)
	req := NewPendingICARequest(portID, channelID, seq, types.RequestKind_REQUEST_KIND_SET_RELATIONSHIP, requestor, ctx.BlockTime(), metaBz)
	if err := k.SetPendingRequest(ctx, req); err != nil {
		return 0, "", "", fmt.Errorf("record pending request: %w", err)
	}

	emitRequestPending(ctx, req)
	return seq, portID, channelID, nil
}

func (k Keeper) RegisterObject(ctx sdk.Context, id string, requestor string) (seq uint64, portID, channelID string, err error) {
	connectionID := k.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return 0, "", "", fmt.Errorf("no connection ID set in module state")
	}

	portID = fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := k.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return 0, "", "", fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	channelID, hasChannel := k.IcaCtrlKeeper.GetActiveChannelID(ctx, connectionID, portID)
	if !hasChannel || channelID == "" {
		return 0, "", "", fmt.Errorf("no active ICA channel for portID %s on connection %s", portID, connectionID)
	}

	policyId := k.GetPolicyId(ctx)
	if policyId == "" {
		return 0, "", "", fmt.Errorf("no policy ID set in module state")
	}

	cmd := acptypes.NewMsgDirectPolicyCmd(
		addr,
		policyId,
		acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.ViewResourceName, id)),
	)

	anyMsg, err := codectypes.NewAnyWithValue(cmd)
	if err != nil {
		return 0, "", "", err
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return 0, "", "", err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())

	seq, err = k.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)
	if err != nil {
		return 0, "", "", err
	}

	meta := &types.RegisterObjectMeta{ResourceName: types.ViewResourceName, ObjectId: id}
	metaBz, _ := k.cdc.Marshal(meta)
	req := NewPendingICARequest(portID, channelID, seq, types.RequestKind_REQUEST_KIND_REGISTER_OBJECT, requestor, ctx.BlockTime(), metaBz)
	if err := k.SetPendingRequest(ctx, req); err != nil {
		return 0, "", "", fmt.Errorf("record pending request: %w", err)
	}

	emitRequestPending(ctx, req)
	return seq, portID, channelID, nil
}
