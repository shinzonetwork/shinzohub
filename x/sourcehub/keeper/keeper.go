package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
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

	// cached authority string for quick equality check
	authority string
	Params    collections.Item[types.Params]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	icaCtrlKeeper types.ICAControllerKeeper,
	authority string,
) Keeper {
	_, err := sdk.AccAddressFromBech32(authority)
	if err != nil {
		panic(err)
	}

	sb := collections.NewSchemaBuilder(storeService)

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		IcaCtrlKeeper: icaCtrlKeeper,
		authority:     authority,
		Params:        collections.NewItem(sb, types.KeyPrefixParams, "params", codec.CollValue[types.Params](cdc)),
	}
}

// msgServer is the concrete implementation of the MsgServer interface
type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) IsAdmin(ctx sdk.Context, address string) bool {
	for _, admin := range k.GetAdmins(ctx) {
		if admin == address {
			return true
		}
	}
	return false
}

func (k Keeper) GetAdmins(ctx sdk.Context) []string {
	p, err := k.GetParams(ctx)
	if err != nil {
		return []string{k.authority}
	}

	if p.Admin == "" || p.Admin == k.authority {
		return []string{k.authority}
	}

	return []string{p.Admin, k.authority}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) RegisterObject(ctx sdk.Context, id string) error {
	connectionID := k.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return fmt.Errorf("no connection ID set in module state")
	}

	portID := fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := k.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	policyId := k.GetPolicyId(ctx)
	if policyId == "" {
		return fmt.Errorf("no policy ID set in module state")
	}

	cmd := acptypes.NewMsgDirectPolicyCmd(
		addr,
		policyId,
		acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.ViewResourceName, id)),
	)

	anyMsg, err := codectypes.NewAnyWithValue(cmd)
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
