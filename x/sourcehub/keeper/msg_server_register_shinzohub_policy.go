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

func (m msgServer) RegisterShinzoPolicy(goCtx context.Context, msg *types.MsgRegisterShinzoPolicy) (*types.MsgRegisterShinzoPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.Keeper.adminKeeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}

	connectionID := m.Keeper.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return nil, fmt.Errorf("no connection ID set in module state")
	}

	portID := fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := m.Keeper.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return nil, fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	channelID, hasChannel := m.Keeper.IcaCtrlKeeper.GetActiveChannelID(ctx, connectionID, portID)
	if !hasChannel || channelID == "" {
		return nil, fmt.Errorf("no active ICA channel for portID %s on connection %s", portID, connectionID)
	}

	policyMsg := &acptypes.MsgCreatePolicy{
		Creator:     addr,
		Policy:      policy,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	anyMsg, err := codectypes.NewAnyWithValue(policyMsg)
	if err != nil {
		return nil, err
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return nil, err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}
	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())

	seq, err := m.Keeper.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)
	if err != nil {
		return nil, err
	}

	req := NewPendingICARequest(portID, channelID, seq, types.RequestKind_REQUEST_KIND_REGISTER_SHINZO_POLICY, msg.Signer, ctx.BlockTime(), nil)
	if err := m.Keeper.SetPendingRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("record pending request: %w", err)
	}
	emitRequestPending(ctx, req)

	return &types.MsgRegisterShinzoPolicyResponse{}, nil
}
