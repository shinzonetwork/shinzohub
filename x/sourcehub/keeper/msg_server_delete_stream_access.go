package keeper

import (
	"context"
	"fmt"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gogoproto "github.com/cosmos/gogoproto/proto"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (m msgServer) DeleteStreamAccess(goCtx context.Context, msg *types.MsgDeleteStreamAccess) (*types.MsgDeleteStreamAccessResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.adminKeeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}

	connectionID := m.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return nil, fmt.Errorf("no connection ID set in module state")
	}

	portID := fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := m.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return nil, fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	channelID, hasChannel := m.IcaCtrlKeeper.GetActiveChannelID(ctx, connectionID, portID)
	if !hasChannel || channelID == "" {
		return nil, fmt.Errorf("no active ICA channel for portID %s on connection %s", portID, connectionID)
	}

	policyId := m.GetPolicyId(ctx)
	if policyId == "" {
		return nil, fmt.Errorf("no policy ID set in module state")
	}

	actor := msg.Did

	resMap := map[uint]string{0: types.PrimitiveResourceName, 1: types.ViewResourceName}

	resource, ok := resMap[uint(msg.Resource)]
	if !ok {
		return nil, fmt.Errorf("invalid resource %q, expected \"0\" or \"1\"", msg.Resource)
	}

	cmd := acptypes.NewMsgDirectPolicyCmd(
		addr,
		policyId,
		acptypes.NewDeleteRelationshipCmd(coretypes.NewActorRelationship(resource, msg.StreamId, "subscriber", actor)),
	)

	anyMsg, err := codectypes.NewAnyWithValue(cmd)
	if err != nil {
		return &types.MsgDeleteStreamAccessResponse{}, err
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return &types.MsgDeleteStreamAccessResponse{}, err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())

	seq, err := m.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)
	if err != nil {
		return nil, err
	}

	// Meta carries the tuple identity (subject, object, resource). The
	// RequestKind on the pending record discriminates set vs delete; the
	// meta shape is shared because the tuple itself is the same.
	metaBz, _ := m.cdc.Marshal(&types.RequestStreamAccessMeta{Did: actor, StreamId: msg.StreamId, ResourceName: resource})
	req := NewPendingICARequest(portID, channelID, seq, types.RequestKind_REQUEST_KIND_DELETE_STREAM_ACCESS, msg.Signer, ctx.BlockTime(), metaBz)
	if err := m.SetPendingRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("record pending request: %w", err)
	}
	emitRequestPending(ctx, req)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeStreamAccessDeletePending,
		sdk.NewAttribute(types.AttrKeyDid, actor),
		sdk.NewAttribute(types.AttrKeyStreamID, msg.StreamId),
		sdk.NewAttribute(types.AttrKeySequence, fmt.Sprintf("%d", seq)),
	))

	return &types.MsgDeleteStreamAccessResponse{
		Sequence:  seq,
		PortId:    portID,
		ChannelId: channelID,
	}, nil
}
