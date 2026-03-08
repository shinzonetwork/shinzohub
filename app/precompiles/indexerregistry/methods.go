package indexerregistry

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	MethodRegister       = "register"
	MethodIsRegistered   = "isRegistered"
	MethodGetDid         = "getDid"
	MethodGetPid         = "getPid"
	MethodGetSourceChain = "getSourceChain"
)

func (p *Precompile) Register(
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

	assertion, found, err := p.indexerKeeper.GetIndexerAssertion(ctx, delegate, sourceChain, sourceChainId)
	if err != nil {
		return nil, fmt.Errorf("indexer assertion lookup failed: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("indexer not asserted for delegate %s on chain %s/%d", delegate, sourceChain, sourceChainId)
	}

	did, pid, err := p.indexerKeeper.RegisterIndexer(
		ctx,
		peerKeyPubkey,
		peerKeySignature,
		nodeIdentityKeyPubkey,
		nodeIdentityKeySignature,
		message,
		caller,
		assertion.SourceChain,
		assertion.SourceChainId,
	)
	if err != nil {
		return nil, err
	}

	precompAddr := contract.Address()
	topic0 := crypto.Keccak256Hash([]byte("Registered(address,bytes,bytes,string,uint64)"))
	event := p.ABI.Events["Registered"]
	data, err := event.Inputs.NonIndexed().Pack(did, pid, assertion.SourceChain, assertion.SourceChainId)
	if err != nil {
		return nil, fmt.Errorf("failed to pack Registered event: %w", err)
	}
	stateDB.AddLog(&gethtypes.Log{
		Address: precompAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(caller),
		},
		Data: data,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"IndexerRegistered",
			sdk.NewAttribute("owner", delegate),
			sdk.NewAttribute("did", string(did)),
			sdk.NewAttribute("pid", string(pid)),
			sdk.NewAttribute("source_chain", assertion.SourceChain),
			sdk.NewAttribute("source_chain_id", fmt.Sprintf("%d", assertion.SourceChainId)),
		),
	)

	return nil, nil
}

func (p *Precompile) IsRegistered(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid type for addr")
	}

	bech32Addr := sdk.AccAddress(addr.Bytes()).String()
	_, found, err := p.indexerKeeper.GetIndexer(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(found)
}

func (p *Precompile) GetDid(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid type for addr")
	}

	bech32Addr := sdk.AccAddress(addr.Bytes()).String()
	indexer, found, err := p.indexerKeeper.GetIndexer(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack([]byte{})
	}
	return method.Outputs.Pack([]byte(indexer.Did))
}

func (p *Precompile) GetPid(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid type for addr")
	}

	bech32Addr := sdk.AccAddress(addr.Bytes()).String()
	indexer, found, err := p.indexerKeeper.GetIndexer(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack([]byte{})
	}
	return method.Outputs.Pack([]byte(indexer.Pid))
}

func (p *Precompile) GetSourceChain(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid type for addr")
	}

	bech32Addr := sdk.AccAddress(addr.Bytes()).String()
	indexer, found, err := p.indexerKeeper.GetIndexer(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack(common.Hash{})
	}
	return method.Outputs.Pack(crypto.Keccak256Hash([]byte(indexer.SourceChain)))
}
