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
	MethodRegister            = "register"
	MethodIsRegistered        = "isRegistered"
	MethodGetDid              = "getDid"
	MethodGetConnectionString = "getConnectionString"
	MethodGetSourceChain      = "getSourceChain"
)

func (p *Precompile) Register(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	_ *abi.Method,
	args []interface{},
) ([]byte, error) {
	connectionString, ok := args[0].(string)
	if !ok || connectionString == "" {
		return nil, fmt.Errorf("invalid connectionString")
	}

	sourceChain, ok := args[1].(string)
	if !ok || sourceChain == "" {
		return nil, fmt.Errorf("invalid sourceChain")
	}

	sourceChainId, ok := args[2].(uint64)
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

	did, err := p.indexerKeeper.RegisterIndexer(
		ctx,
		connectionString,
		caller,
		assertion.SourceChain,
		assertion.SourceChainId,
	)
	if err != nil {
		return nil, err
	}

	precompAddr := contract.Address()
	topic0 := crypto.Keccak256Hash([]byte("Registered(address,bytes,string,string,uint64)"))
	event := p.ABI.Events["Registered"]
	data, err := event.Inputs.NonIndexed().Pack(did, connectionString, assertion.SourceChain, assertion.SourceChainId)
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
			sdk.NewAttribute("connection_string", connectionString),
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

func (p *Precompile) GetConnectionString(
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
		return method.Outputs.Pack("")
	}
	return method.Outputs.Pack(indexer.ConnectionString)
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
