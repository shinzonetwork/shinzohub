package keeper

import (
	"context"
	"fmt"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"

	gogoproto "github.com/cosmos/gogoproto/proto"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (m msgServer) RegisterShinzoObjects(
	goCtx context.Context,
	msg *types.MsgRegisterShinzoObjects,
) (*types.MsgRegisterShinzoObjectsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.Keeper.IsAdmin(ctx, msg.Signer) {
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

	policyID := m.Keeper.GetPolicyId(ctx)
	if policyID == "" {
		return nil, fmt.Errorf("no policy ID set in module state")
	}

	var anyMsgs []*codectypes.Any

	for _, r := range msg.Resources {
		cmd := acptypes.NewMsgDirectPolicyCmd(
			addr,
			policyID,
			acptypes.NewRegisterObjectCmd(
				coretypes.NewObject(types.PrimitiveResourceName, r),
			),
		)

		anyMsg, err := codectypes.NewAnyWithValue(cmd)
		if err != nil {
			return nil, fmt.Errorf("wrap primitive %q: %w", r, err)
		}

		anyMsgs = append(anyMsgs, anyMsg)
	}

	{
		cmd := acptypes.NewMsgDirectPolicyCmd(
			addr,
			policyID,
			acptypes.NewRegisterObjectCmd(
				coretypes.NewObject(types.GroupObjectName, types.GroupIndexerName),
			),
		)
		anyMsg, err := codectypes.NewAnyWithValue(cmd)
		if err != nil {
			return nil, fmt.Errorf("wrap group indexer: %w", err)
		}
		anyMsgs = append(anyMsgs, anyMsg)
	}
	{
		cmd := acptypes.NewMsgDirectPolicyCmd(
			addr,
			policyID,
			acptypes.NewRegisterObjectCmd(
				coretypes.NewObject(types.GroupObjectName, types.GroupHostName),
			),
		)
		anyMsg, err := codectypes.NewAnyWithValue(cmd)
		if err != nil {
			return nil, fmt.Errorf("wrap group host: %w", err)
		}
		anyMsgs = append(anyMsgs, anyMsg)
	}

	cosmosTx := &icatypes.CosmosTx{Messages: anyMsgs}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return nil, fmt.Errorf("marshal CosmosTx: %w", err)
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())
	_, err = m.Keeper.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)
	if err != nil {
		return nil, fmt.Errorf("ICA SendTx: %w", err)
	}

	return &types.MsgRegisterShinzoObjectsResponse{}, nil
}
