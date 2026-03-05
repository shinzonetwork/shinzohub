package entityregistry

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"

	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

const (
	EntityRegistryRegisterIndexerMethod = "registerIndexer"
	EntityRegistryRegisterHostMethod    = "registerHost"
)

// EntityRegistryRegisterIndexer handles precompile calls to registerIndexer(…).
// The caller must have a prior assertion record keyed on (delegate, sourceChain,
// sourceChainId).  The chain info is sourced from the assertion and emitted in
// the EntityRegistered cosmos event.
func (p Precompile) EntityRegistryRegisterIndexer(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	_ *abi.Method,
	args []interface{},
) ([]byte, error) {
	peerKeyPubkey, ok := args[0].([]byte)
	if !ok || len(peerKeyPubkey) == 0 {
		return nil, fmt.Errorf("invalid peerKeyPubkey")
	}

	peerKeySignature, ok := args[1].([]byte)
	if !ok || len(peerKeySignature) == 0 {
		return nil, fmt.Errorf("invalid peerKeySignature")
	}

	nodeIdentityKeyPubkey, ok := args[2].([]byte)
	if !ok || len(nodeIdentityKeyPubkey) == 0 {
		return nil, fmt.Errorf("invalid nodeIdentityKeyPubkey")
	}

	nodeIdentityKeySignature, ok := args[3].([]byte)
	if !ok || len(nodeIdentityKeySignature) == 0 {
		return nil, fmt.Errorf("invalid nodeIdentityKeySignature")
	}

	message, ok := args[4].([]byte)
	if !ok || len(message) == 0 {
		return nil, fmt.Errorf("invalid message")
	}

	sourceChain, ok := args[5].(string)
	if !ok || sourceChain == "" {
		return nil, fmt.Errorf("invalid sourceChain")
	}

	sourceChainId, ok := args[6].(uint64)
	if !ok || sourceChainId == 0 {
		return nil, fmt.Errorf("invalid sourceChainId: must be non-zero")
	}

	caller := contract.Caller().Bytes()
	delegate := sdk.AccAddress(caller).String()

	att, found, err := p.sourcehubKeeper.GetIndexerAssertion(ctx, delegate, sourceChain, sourceChainId)
	if err != nil {
		return nil, fmt.Errorf("indexer assertion lookup failed: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("indexer not asserted for delegate %s on chain %s/%d", delegate, sourceChain, sourceChainId)
	}

	did, pid, err := p.sourcehubKeeper.RegisterEntity(
		ctx,
		peerKeyPubkey,
		peerKeySignature,
		nodeIdentityKeyPubkey,
		nodeIdentityKeySignature,
		message,
		sourcehubtypes.RoleIndexer,
		caller,
	)
	if err != nil {
		return nil, err
	}

	return emitEntityRegistered(ctx, contract, stateDB, p, caller, did, pid, sourcehubtypes.RoleIndexer, att.SourceChain, fmt.Sprintf("%d", att.SourceChainId))
}

// EntityRegistryRegisterHost handles precompile calls to registerHost(…).
// Hosts do not require a prior assertion; they register directly with their
// cryptographic key proofs.
func (p Precompile) EntityRegistryRegisterHost(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	_ *abi.Method,
	args []interface{},
) ([]byte, error) {
	peerKeyPubkey, ok := args[0].([]byte)
	if !ok || len(peerKeyPubkey) == 0 {
		return nil, fmt.Errorf("invalid peerKeyPubkey")
	}

	peerKeySignature, ok := args[1].([]byte)
	if !ok || len(peerKeySignature) == 0 {
		return nil, fmt.Errorf("invalid peerKeySignature")
	}

	nodeIdentityKeyPubkey, ok := args[2].([]byte)
	if !ok || len(nodeIdentityKeyPubkey) == 0 {
		return nil, fmt.Errorf("invalid nodeIdentityKeyPubkey")
	}

	nodeIdentityKeySignature, ok := args[3].([]byte)
	if !ok || len(nodeIdentityKeySignature) == 0 {
		return nil, fmt.Errorf("invalid nodeIdentityKeySignature")
	}

	message, ok := args[4].([]byte)
	if !ok || len(message) == 0 {
		return nil, fmt.Errorf("invalid message")
	}

	caller := contract.Caller().Bytes()

	did, pid, err := p.sourcehubKeeper.RegisterEntity(
		ctx,
		peerKeyPubkey,
		peerKeySignature,
		nodeIdentityKeyPubkey,
		nodeIdentityKeySignature,
		message,
		sourcehubtypes.RoleHost,
		caller,
	)
	if err != nil {
		return nil, err
	}

	// sourceChain / sourceChainId are intentionally blank for hosts.
	return emitEntityRegistered(ctx, contract, stateDB, p, caller, did, pid, sourcehubtypes.RoleHost, "", "")
}

// emitEntityRegistered writes the EVM log and the cosmos event that both
// registration paths share.
func emitEntityRegistered(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	p Precompile,
	caller, did, pid []byte,
	entity uint8,
	sourceChain, sourceChainId string,
) ([]byte, error) {
	key := crypto.Keccak256Hash(caller, did)

	topic0 := crypto.Keccak256Hash([]byte("EntityRegistered(bytes32,address,bytes,bytes,uint8)"))

	event := p.ABI.Events["EntityRegistered"]

	data, err := event.Inputs.NonIndexed().Pack(did, pid, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to pack EntityRegistered event: %w", err)
	}

	stateDB.AddLog(&gethtypes.Log{
		Address: contract.Address(),
		Topics: []common.Hash{
			topic0,
			key,
			common.BytesToHash(caller),
		},
		Data: data,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"EntityRegistered",
			sdk.NewAttribute("key", key.Hex()),
			sdk.NewAttribute("owner", sdk.AccAddress(caller).String()),
			sdk.NewAttribute("did", string(did)),
			sdk.NewAttribute("pid", string(pid)),
			sdk.NewAttribute("entity", string(entity)),
			sdk.NewAttribute("source_chain", sourceChain),
			sdk.NewAttribute("source_chain_id", sourceChainId),
		),
	)

	return nil, nil
}
