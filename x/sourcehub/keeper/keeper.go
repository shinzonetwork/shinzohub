package keeper

import (
	"fmt"
	"time"

	storetypes "cosmossdk.io/core/store"
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
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	icaCtrlKeeper types.ICAControllerKeeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		IcaCtrlKeeper: icaCtrlKeeper,
	}
}

func (k Keeper) SendPolicy(ctx sdk.Context, policy string) error {
	connectionID := k.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return fmt.Errorf("no connection ID set in module state")
	}

	portID := fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := k.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	mt := coretypes.PolicyMarshalingType_SHORT_YAML

	msg := &acptypes.MsgCreatePolicy{
		Creator:     addr,
		Policy:      policy,
		MarshalType: mt,
	}

	anyMsg, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return err
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())

	_, err = k.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)
	return err
}
