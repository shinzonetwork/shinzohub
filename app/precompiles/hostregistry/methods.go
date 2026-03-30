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

	caller := contract.Caller().Bytes()

	did, err := p.hostKeeper.RegisterHost(
		ctx,
		connectionString,
		caller,
	)
	if err != nil {
		return nil, err
	}

	precompAddr := contract.Address()
	topic0 := crypto.Keccak256Hash([]byte("Registered(address,bytes,string)"))
	event := p.ABI.Events["Registered"]
	data, err := event.Inputs.NonIndexed().Pack(did, connectionString)
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
			"HostRegistered",
			sdk.NewAttribute("owner", sdk.AccAddress(caller).String()),
			sdk.NewAttribute("did", string(did)),
			sdk.NewAttribute("connection_string", connectionString),
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

	registered := p.hostKeeper.IsRegisteredHost(ctx, addr.Bytes())
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
