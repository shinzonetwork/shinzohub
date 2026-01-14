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

	if !m.Keeper.IsAdmin(ctx, msg.Signer) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("admin required")
	}

	connectionID := m.Keeper.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return &types.MsgRegisterShinzoPolicyResponse{}, fmt.Errorf("no connection ID set in module state")
	}

	portID := fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := m.Keeper.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return &types.MsgRegisterShinzoPolicyResponse{},
			fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	mt := coretypes.PolicyMarshalingType_YAML

	policyMsg := &acptypes.MsgCreatePolicy{
		Creator:     addr,
		Policy:      policy,
		MarshalType: mt,
	}

	anyMsg, err := codectypes.NewAnyWithValue(policyMsg)
	if err != nil {
		return &types.MsgRegisterShinzoPolicyResponse{}, err
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return &types.MsgRegisterShinzoPolicyResponse{}, err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())

	_, err = m.Keeper.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)

	return &types.MsgRegisterShinzoPolicyResponse{}, err
}
