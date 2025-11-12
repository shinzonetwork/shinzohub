package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
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

func (k Keeper) RegisterEntity(
	ctx sdk.Context,
	peerKeyPubkey []byte,
	peerKeySignature []byte,
	nodeIdentityKeyPubkey []byte,
	nodeIdentityKeySignature []byte,
	message []byte,
	entity uint8, // 0 = indexer, 1 = host
	address []byte, // tx signer address (raw bytes, not bech32 string)
) ([]byte, []byte, error) {

	connectionID := k.GetControllerConnectionID(ctx)
	if connectionID == "" {
		return nil, nil, fmt.Errorf("no connection ID set in module state")
	}

	portID := fmt.Sprintf("icacontroller-%s", types.ModuleAddress.String())

	addr, _ := k.IcaCtrlKeeper.GetInterchainAccountAddress(ctx, connectionID, portID)
	if addr == "" {
		return nil, nil, fmt.Errorf("ICA address not found for portID %s on connection %s", portID, connectionID)
	}

	policyId := k.GetPolicyId(ctx)
	if policyId == "" {
		return nil, nil, fmt.Errorf("no policy ID set in module state")
	}

	role := entity

	if err := verifyPeerKeySignature(peerKeyPubkey, message, peerKeySignature); err != nil {
		return nil, nil, err
	}

	if err := verifynodeIdentityKeySignature(nodeIdentityKeyPubkey, message, nodeIdentityKeySignature); err != nil {
		return nil, nil, err
	}

	pid, err := derivePIDFromPeerKeyPublicKey(peerKeyPubkey)
	if err != nil {
		return nil, nil, err
	}

	did, err := deriveDIDFromNodeIdentityPublicKey(nodeIdentityKeyPubkey)
	if err != nil {
		return nil, nil, err
	}

	didBytes := []byte(did)
	pidBytes := []byte(pid)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	addrKey := addrRoleKey(address, role)
	didKey := didRoleKey(didBytes, role)

	existingDidForAddr := store.Get(addrKey)
	if len(existingDidForAddr) > 0 {
		if !bytesEqual(existingDidForAddr, didBytes) {
			return nil, nil, fmt.Errorf("address already registered for this role with a different DID")
		}
	}

	existingAddrForDid := store.Get(didKey)
	if len(existingAddrForDid) > 0 {
		if !bytesEqual(existingAddrForDid, address) {
			return nil, nil, fmt.Errorf("DID already registered for this role with a different address")
		}
	}

	cmd := acptypes.NewMsgDirectPolicyCmd(
		addr,
		policyId,
		acptypes.NewSetRelationshipCmd(coretypes.NewActorRelationship("group", RoleToString(role), "guest", did)),
	)

	anyMsg, err := codectypes.NewAnyWithValue(cmd)
	if err != nil {
		return nil, nil, err
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		return nil, nil, err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	timeout := uint64(ctx.BlockTime().Add(5 * time.Minute).UnixNano())

	_, err = k.IcaCtrlKeeper.SendTx(ctx, connectionID, portID, packetData, timeout)
	if err != nil {
		return nil, nil, err
	}

	store.Set(addrKey, didBytes)
	store.Set(didKey, address)

	return didBytes, pidBytes, nil
}

func (k Keeper) GetDidForAddressRole(ctx sdk.Context, address []byte, role uint8) ([]byte, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	v := store.Get(addrRoleKey(address, role))
	if len(v) == 0 {
		return nil, false
	}
	return v, true
}

func (k Keeper) GetDidByAddress(ctx sdk.Context, address []byte) ([]byte, error) {
	return []byte{}, fmt.Errorf("")
}

func (k Keeper) GetPidByAddress(ctx sdk.Context, address []byte) ([]byte, error) {
	return []byte{}, fmt.Errorf("")
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
