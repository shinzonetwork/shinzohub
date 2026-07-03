package hostregistry

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
	MethodGetEndpointAddress  = "getEndpointAddress"
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

	endpointAddress, ok := args[4].(string)
	if !ok || endpointAddress == "" {
		return nil, fmt.Errorf("invalid endpointAddress")
	}

	if err := p.sourcehubKeeper.CheckICAReady(ctx); err != nil {
		return nil, err
	}

	callerEVM := contract.Caller()
	caller := sdk.AccAddress(callerEVM.Bytes())

	did, err := p.hostKeeper.RegisterHost(
		ctx,
		nodeIdentityKeyPubkey,
		nodeIdentityKeySignature,
		message,
		connectionString,
		endpointAddress,
		caller,
	)
	if err != nil {
		return nil, err
	}

	precompAddr := contract.Address()
	topic0 := crypto.Keccak256Hash([]byte("Registered(address,bytes,string,string)"))
	event := p.ABI.Events["Registered"]
	data, err := event.Inputs.NonIndexed().Pack(did, connectionString, endpointAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to pack Registered event: %w", err)
	}
	stateDB.AddLog(&gethtypes.Log{
		Address: precompAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(callerEVM.Bytes()),
		},
		Data: data,
	})

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

	registered := p.hostKeeper.IsRegisteredHost(ctx, sdk.AccAddress(addr.Bytes()))
	return method.Outputs.Pack(registered)
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
	host, found, err := p.hostKeeper.GetHost(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack([]byte{})
	}
	return method.Outputs.Pack([]byte(host.Did))
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
	host, found, err := p.hostKeeper.GetHost(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack("")
	}
	return method.Outputs.Pack(host.ConnectionString)
}

func (p *Precompile) GetEndpointAddress(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid type for addr")
	}

	bech32Addr := sdk.AccAddress(addr.Bytes()).String()
	host, found, err := p.hostKeeper.GetHost(ctx, bech32Addr)
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack("")
	}
	return method.Outputs.Pack(host.EndpointAddress)
}
