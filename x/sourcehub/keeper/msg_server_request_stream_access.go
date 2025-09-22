package keeper

import (
	"context"
	"fmt"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (m msgServer) RequestStreamAccess(goCtx context.Context, msg *types.MsgRequestStreamAccess) (*types.MsgRequestStreamAccessResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	connectionID := m.Keeper.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return nil, fmt.Errorf("no connection ID set in module state")
	}

	portID := fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := m.Keeper.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return nil, fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	policyId := m.Keeper.GetPolicyId(ctx)
	if policyId == "" {
		return nil, fmt.Errorf("no policy ID set in module state")
	}

	actor := fmt.Sprintf("did:key:%s", msg.Did)

	cmd := acptypes.NewMsgDirectPolicyCmd(
		addr,
		policyId,
		acptypes.NewSetRelationshipCmd(coretypes.NewActorRelationship(msg.Resource, msg.StreamId, "subscriber", actor)),
	)

	anyMsg, err := codectypes.NewAnyWithValue(cmd)
	if err != nil {
		return &types.MsgRequestStreamAccessResponse{}, err
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return &types.MsgRequestStreamAccessResponse{}, err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())

	_, err = m.Keeper.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)

	return &types.MsgRequestStreamAccessResponse{}, err
}
