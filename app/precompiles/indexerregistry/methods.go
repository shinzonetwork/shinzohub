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
	nodeIdentityKeyPubkey, ok := args[0].([]byte)
	if !ok || len(nodeIdentityKeyPubkey) == 0 {
		return nil, fmt.Errorf("invalid nodeIdentityKeyPubkey")
	}
	nodeIdentityKeySignature, ok := args[1].([]byte)
	if !ok || len(nodeIdentityKeySignature) == 0 {
		return nil, fmt.Errorf("invalid nodeIdentityKeySignature")
	}
	message, ok := args[2].([]byte)
	if !ok || len(message) == 0 {
		return nil, fmt.Errorf("invalid message")
	}
	connectionString, ok := args[3].(string)
	if !ok || connectionString == "" {
		return nil, fmt.Errorf("invalid connectionString")
	}

	if err := p.sourcehubKeeper.CheckICAReady(ctx); err != nil {
		return nil, err
	}

	caller := contract.Caller().Bytes()
	callerBech32 := sdk.AccAddress(caller).String()
	result, err := p.indexerKeeper.RegisterIndexer(
		ctx,
		callerBech32,
		nodeIdentityKeyPubkey,
		nodeIdentityKeySignature,
		message,
		connectionString,
	)
	if err != nil {
		return nil, err
	}

	if result.Pending {
		topic0 := crypto.Keccak256Hash([]byte("Registered(address,bytes,string,string,uint64)"))
		event := p.ABI.Events["Registered"]
		data, perr := event.Inputs.NonIndexed().Pack(
			[]byte(result.Did),
			connectionString,
			result.SourceChain,
			result.SourceChainID,
		)
		if perr != nil {
			return nil, fmt.Errorf("failed to pack Registered event: %w", perr)
		}
		stateDB.AddLog(&gethtypes.Log{
			Address: contract.Address(),
			Topics: []common.Hash{
				topic0,
				common.BytesToHash(caller),
			},
			Data: data,
		})
	}

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
	row, found, err := p.indexerKeeper.GetIndexerByAddress(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(found && row.Registered)
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
	row, found, err := p.indexerKeeper.GetIndexerByAddress(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack([]byte{})
	}
	return method.Outputs.Pack([]byte(row.Did))
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
	row, found, err := p.indexerKeeper.GetIndexerByAddress(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack("")
	}
	return method.Outputs.Pack(row.ConnectionString)
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
	row, found, err := p.indexerKeeper.GetIndexerByAddress(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack(common.Hash{})
	}
	return method.Outputs.Pack(crypto.Keccak256Hash([]byte(row.SourceChain)))
}
