package poolregistry

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/shinzonetwork/shinzohub/x/pool/types"
)

const (
	MethodRegisterDemandForView = "registerDemandForView"
	MethodPoolsOf               = "poolsOf"
	MethodViewOfPool            = "viewOfPool"
	MethodGetPool               = "getPool"
	MethodGetPoolFor            = "getPoolFor"
	MethodGetPoolDetail         = "getPoolDetail"
	MethodJoinPool              = "joinPool"
	MethodLeavePool             = "leavePool"
)

// poolConfigInput mirrors the Solidity PoolConfig struct for ABI decoding.
type poolConfigInput struct {
	WindowSize uint64
}

// poolOutput mirrors the Solidity Pool struct for ABI encoding.
type poolOutput struct {
	PoolAddress common.Address
	ViewAddress common.Address
	Config      poolConfigInput
	IsActive    bool
	Price       *big.Int
}

// hostEntryOutput mirrors the Solidity PoolHostEntry struct.
type hostEntryOutput struct {
	HostAddress common.Address
	JoinedAt    int64
}

// demandEntryOutput mirrors the Solidity PoolDemandEntry struct.
type demandEntryOutput struct {
	Registrant common.Address
	Bond       *big.Int
	PricePref  *big.Int
	Binding    bool
	ExpiresAt  int64
}

// poolDetailOutput mirrors the Solidity PoolDetail struct.
type poolDetailOutput struct {
	Pool    poolOutput
	Hosts   []hostEntryOutput
	Demands []demandEntryOutput
}

// parseInt parses a base-10 string into *big.Int. Empty or invalid strings
// become zero. Used to translate the keeper's string-encoded amounts to
// uint256 on the wire.
func parseInt(s string) *big.Int {
	if s == "" {
		return new(big.Int)
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return new(big.Int)
	}
	return n
}

// poolSalt is the deterministic 32-byte CREATE2 salt for the (view, config) pair.
// keccak256(viewAddress || bigEndian(windowSize)).
func poolSalt(viewAddr common.Address, cfg poolConfigInput) [32]byte {
	var buf [20 + 8]byte
	copy(buf[:20], viewAddr.Bytes())
	binary.BigEndian.PutUint64(buf[20:], cfg.WindowSize)
	return crypto.Keccak256Hash(buf[:])
}

// buildPoolInitCode returns Pool.sol's constructor init code (bytecode + packed args).
// Only viewAddress is encoded; config differentiates pools via the CREATE2 salt,
// not the constructor.
func buildPoolInitCode(viewAddr common.Address, _ poolConfigInput) ([]byte, error) {
	args, err := PoolConstructorArgs.Pack(viewAddr)
	if err != nil {
		return nil, err
	}
	return append(append([]byte{}, PoolBytecode...), args...), nil
}

// derivePoolAddress predicts the CREATE2 address that Pool.sol will deploy at.
// keccak256(0xff || precompileAddr || salt || keccak256(initCode))[12:].
func derivePoolAddress(viewAddr common.Address, cfg poolConfigInput) common.Address {
	initCode, _ := buildPoolInitCode(viewAddr, cfg)
	salt := poolSalt(viewAddr, cfg)
	deployer := common.HexToAddress(PrecompileAddress)
	return crypto.CreateAddress2(deployer, salt, crypto.Keccak256(initCode))
}

func (p Precompile) RegisterDemandForView(
	ctx sdk.Context,
	evm *vm.EVM,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	viewAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid view address")
	}

	cfg := *abi.ConvertType(args[1], new(poolConfigInput)).(*poolConfigInput)

	bond := contract.Value()
	if bond == nil || bond.Sign() == 0 {
		return nil, fmt.Errorf("must send a non-zero bond")
	}

	poolAddr := derivePoolAddress(viewAddress, cfg)
	registrant := contract.Caller()
	poolAddrHex := poolAddr.Hex()
	viewAddrHex := viewAddress.Hex()

	created := false
	if !p.poolKeeper.PoolExists(ctx, poolAddrHex) {
		// Deploy Pool.sol via CREATE2 so the EVM-addressable contract sits at
		// the same deterministic address the keeper records.
		// Skipped if PoolBytecode is empty (placeholder build).
		if len(PoolBytecode) > 0 {
			initCode, err := buildPoolInitCode(viewAddress, cfg)
			if err != nil {
				return nil, fmt.Errorf("build pool init code: %w", err)
			}
			salt := poolSalt(viewAddress, cfg)
			saltU256 := new(uint256.Int).SetBytes(salt[:])

			_, deployedAddr, leftoverGas, createErr := evm.Create2(
				contract.Address(),
				initCode,
				contract.Gas,
				new(uint256.Int),
				saltU256,
			)
			if createErr != nil {
				return nil, fmt.Errorf("deploy pool contract: %w", createErr)
			}
			contract.Gas = leftoverGas

			if deployedAddr != poolAddr {
				return nil, fmt.Errorf(
					"create2 address mismatch: expected %s got %s",
					poolAddr.Hex(), deployedAddr.Hex(),
				)
			}
		}

		if err := p.poolKeeper.CreatePool(ctx, poolAddrHex, viewAddrHex, types.PoolConfig{
			WindowSize: cfg.WindowSize,
		}); err != nil {
			return nil, fmt.Errorf("create pool: %w", err)
		}
		created = true
	}

	demand := types.PoolDemand{
		Bond:      math.NewIntFromBigInt(bond.ToBig()).String(),
		PricePref: "0",
		Binding:   false,
		ExpiresAt: 0,
	}
	if err := p.poolKeeper.RegisterDemand(ctx, poolAddrHex, registrant.Hex(), demand); err != nil {
		return nil, fmt.Errorf("register demand: %w", err)
	}

	if created {
		emitPoolCreated(stateDB, contract.Address(), poolAddr, viewAddress, cfg)
	}
	emitDemandRegistered(stateDB, contract.Address(), poolAddr, registrant, bond.ToBig())

	out := poolOutput{
		PoolAddress: poolAddr,
		ViewAddress: viewAddress,
		Config:      cfg,
		IsActive:    p.poolKeeper.IsActive(ctx, poolAddrHex),
		Price:       p.poolKeeper.GetPoolPrice(ctx, poolAddrHex).BigInt(),
	}
	return method.Outputs.Pack(out)
}

func (p Precompile) PoolsOf(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	viewAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid view address")
	}

	pools, err := p.poolKeeper.GetPoolsForView(ctx, viewAddress.Hex())
	if err != nil {
		return nil, err
	}

	out := make([]common.Address, 0, len(pools))
	for _, addr := range pools {
		out = append(out, common.HexToAddress(addr))
	}
	return method.Outputs.Pack(out)
}

func (p Precompile) ViewOfPool(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	poolAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid pool address")
	}

	pool, found, err := p.poolKeeper.GetPool(ctx, poolAddress.Hex())
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack(common.Address{})
	}
	return method.Outputs.Pack(common.HexToAddress(pool.ViewAddress))
}

func (p Precompile) GetPool(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	poolAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid pool address")
	}

	pool, found, err := p.poolKeeper.GetPool(ctx, poolAddress.Hex())
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack(poolOutput{})
	}

	out := poolOutput{
		PoolAddress: common.HexToAddress(pool.PoolAddress),
		ViewAddress: common.HexToAddress(pool.ViewAddress),
		Config:      poolConfigInput{WindowSize: pool.Config.WindowSize},
		IsActive:    p.poolKeeper.IsActive(ctx, poolAddress.Hex()),
		Price:       p.poolKeeper.GetPoolPrice(ctx, poolAddress.Hex()).BigInt(),
	}
	return method.Outputs.Pack(out)
}

func (p Precompile) GetPoolFor(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	viewAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid view address")
	}

	cfg := *abi.ConvertType(args[1], new(poolConfigInput)).(*poolConfigInput)

	poolAddr := derivePoolAddress(viewAddress, cfg)
	pool, found, err := p.poolKeeper.GetPool(ctx, poolAddr.Hex())
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack(poolOutput{})
	}

	out := poolOutput{
		PoolAddress: common.HexToAddress(pool.PoolAddress),
		ViewAddress: common.HexToAddress(pool.ViewAddress),
		Config:      poolConfigInput{WindowSize: pool.Config.WindowSize},
		IsActive:    p.poolKeeper.IsActive(ctx, poolAddr.Hex()),
		Price:       p.poolKeeper.GetPoolPrice(ctx, poolAddr.Hex()).BigInt(),
	}
	return method.Outputs.Pack(out)
}

func (p Precompile) GetPoolDetail(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	poolAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid pool address")
	}

	detail, found, err := p.poolKeeper.GetPoolDetail(ctx, poolAddress.Hex())
	if err != nil {
		return nil, err
	}
	if !found {
		return method.Outputs.Pack(poolDetailOutput{})
	}

	hosts := make([]hostEntryOutput, 0, len(detail.Hosts))
	for _, h := range detail.Hosts {
		hosts = append(hosts, hostEntryOutput{
			HostAddress: common.HexToAddress(h.HostAddress),
			JoinedAt:    h.Host.JoinedAt,
		})
	}

	demands := make([]demandEntryOutput, 0, len(detail.Demands))
	for _, d := range detail.Demands {
		demands = append(demands, demandEntryOutput{
			Registrant: common.HexToAddress(d.RegistrantAddress),
			Bond:       parseInt(d.Demand.Bond),
			PricePref:  parseInt(d.Demand.PricePref),
			Binding:    d.Demand.Binding,
			ExpiresAt:  d.Demand.ExpiresAt,
		})
	}

	out := poolDetailOutput{
		Pool: poolOutput{
			PoolAddress: common.HexToAddress(detail.Pool.PoolAddress),
			ViewAddress: common.HexToAddress(detail.Pool.ViewAddress),
			Config:      poolConfigInput{WindowSize: detail.Pool.Config.WindowSize},
			IsActive:    p.poolKeeper.IsActive(ctx, poolAddress.Hex()),
			Price:       p.poolKeeper.GetPoolPrice(ctx, poolAddress.Hex()).BigInt(),
		},
		Hosts:   hosts,
		Demands: demands,
	}
	return method.Outputs.Pack(out)
}

// JoinPool adds the caller-supplied host to the pool identified by the EVM caller.
// The pool address is taken from contract.Caller() (Pool.sol's address); calls
// from anything other than a registered pool are rejected.
func (p Precompile) JoinPool(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	host, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid host address")
	}

	poolAddrHex := contract.Caller().Hex()
	if !p.poolKeeper.PoolExists(ctx, poolAddrHex) {
		return nil, fmt.Errorf("caller is not a registered pool: %s", poolAddrHex)
	}

	if err := p.poolKeeper.AddHost(ctx, poolAddrHex, host.Hex()); err != nil {
		return nil, fmt.Errorf("add host: %w", err)
	}
	return nil, nil
}

// LeavePool removes the caller-supplied host from the pool identified by the
// EVM caller. Same authorisation as JoinPool.
func (p Precompile) LeavePool(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	host, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid host address")
	}

	poolAddrHex := contract.Caller().Hex()
	if !p.poolKeeper.PoolExists(ctx, poolAddrHex) {
		return nil, fmt.Errorf("caller is not a registered pool: %s", poolAddrHex)
	}

	if err := p.poolKeeper.RemoveHost(ctx, poolAddrHex, host.Hex()); err != nil {
		return nil, fmt.Errorf("remove host: %w", err)
	}
	return nil, nil
}

func emitPoolCreated(
	stateDB vm.StateDB,
	precompileAddr common.Address,
	poolAddr, viewAddr common.Address,
	cfg poolConfigInput,
) {
	topic0 := crypto.Keccak256Hash([]byte("PoolCreated(address,address,(uint64))"))
	// Non-indexed data: the PoolConfig tuple.
	configArgs := abi.Arguments{
		{Type: mustABIType("(uint64)")},
	}
	data, _ := configArgs.Pack(cfg)
	stateDB.AddLog(&gethtypes.Log{
		Address: precompileAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(poolAddr.Bytes()),
			common.BytesToHash(viewAddr.Bytes()),
		},
		Data: data,
	})
}

func emitDemandRegistered(
	stateDB vm.StateDB,
	precompileAddr common.Address,
	poolAddr, registrant common.Address,
	bond interface{},
) {
	topic0 := crypto.Keccak256Hash([]byte("DemandRegistered(address,address,uint256)"))
	uintArgs := abi.Arguments{
		{Type: mustABIType("uint256")},
	}
	data, _ := uintArgs.Pack(bond)
	stateDB.AddLog(&gethtypes.Log{
		Address: precompileAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(poolAddr.Bytes()),
			common.BytesToHash(registrant.Bytes()),
		},
		Data: data,
	})
}

func mustABIType(t string) abi.Type {
	at, err := abi.NewType(t, "", nil)
	if err != nil {
		panic(err)
	}
	return at
}
