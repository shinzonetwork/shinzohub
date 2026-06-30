package viewregistry

import (
	"fmt"
	"math/big"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shinzonetwork/viewbundle-go"

	"github.com/shinzonetwork/shinzohub/x/view/types"
)

const (
	MethodRegister              = "register"
	MethodGetView               = "getView"
	MethodListViews             = "listViews"
	MethodViewCount             = "viewCount"
	MethodRegisterDemandForView = "registerDemandForView"
)

// Matches a GraphQL type declaration anchored to the start of a line so that a
// `type <name>` mentioned inside a leading `#` comment is not picked up as the
// view name.
var sdlTypeRe = regexp.MustCompile(`(?m)^[ \t]*type[ \t]+([A-Za-z0-9_]+)\b`)

var viewCreatedTopic0 = crypto.Keccak256Hash([]byte("ViewCreated(address,address,string)"))

var viewCreatedDataArgs = func() abi.Arguments {
	stringType, _ := abi.NewType("string", "", nil)
	return abi.Arguments{{Name: "name", Type: stringType}}
}()

const (
	statusNone       uint8 = 0
	statusPending    uint8 = 1
	statusRegistered uint8 = 2
)

type viewTuple struct {
	ViewAddress common.Address `abi:"viewAddress"`
	Name        string         `abi:"name"`
	Creator     string         `abi:"creator"`
	Height      uint64         `abi:"height"`
	Status      uint8          `abi:"status"`
}

func toViewTuple(v types.View, status uint8) viewTuple {
	return viewTuple{
		ViewAddress: common.HexToAddress(v.Address),
		Name:        v.Name,
		Creator:     v.Creator,
		Height:      v.Height,
		Status:      status,
	}
}

func (p Precompile) Register(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	data, ok := args[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid data")
	}

	decoded, err := viewbundle.DecodeHeader(data)
	if err != nil {
		return nil, fmt.Errorf("decode viewbundle: %w", err)
	}
	matches := sdlTypeRe.FindStringSubmatch(decoded.Header.Sdl)
	if len(matches) < 2 {
		return nil, fmt.Errorf("SDL missing type name")
	}
	name := matches[1]

	caller := contract.Caller()
	id := crypto.Keccak256Hash([]byte("shinzo.view.v1"), caller.Bytes(), data)
	viewAddr := common.BytesToAddress(id.Bytes())

	if _, err := p.viewKeeper.RegisterView(ctx, name, caller.Hex(), viewAddr.Hex(), data); err != nil {
		return nil, fmt.Errorf("register view: %w", err)
	}

	nameData, err := viewCreatedDataArgs.Pack(name)
	if err != nil {
		return nil, fmt.Errorf("pack event data: %w", err)
	}

	stateDB.AddLog(&gethtypes.Log{
		Address: p.Address(),
		Topics: []common.Hash{
			viewCreatedTopic0,
			common.BytesToHash(viewAddr.Bytes()),
			common.BytesToHash(caller.Bytes()),
		},
		Data: nameData,
	})

	return method.Outputs.Pack(viewAddr, name)
}

func (p Precompile) GetView(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	viewAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid viewAddress")
	}

	addr := viewAddress.Hex()

	if view, found, err := p.viewKeeper.GetView(ctx, addr); err != nil {
		return nil, err
	} else if found {
		return method.Outputs.Pack(toViewTuple(view, statusRegistered))
	}

	if view, found, err := p.viewKeeper.GetPendingView(ctx, addr); err != nil {
		return nil, err
	} else if found {
		return method.Outputs.Pack(toViewTuple(view, statusPending))
	}

	return method.Outputs.Pack(viewTuple{Status: statusNone})
}

func (p Precompile) ListViews(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	offset, ok := args[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid offset")
	}
	limit, ok := args[1].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid limit")
	}

	// query.Paginate treats Limit==0 as "unset" and falls back to a default page
	// size (100). The ABI contract is an explicit page size, so a zero limit must
	// return an empty page rather than silently fetching the default.
	if limit.Sign() == 0 {
		return method.Outputs.Pack([]viewTuple{})
	}

	views, _, err := p.viewKeeper.GetAllViews(ctx, &query.PageRequest{
		Offset: offset.Uint64(),
		Limit:  limit.Uint64(),
	})
	if err != nil {
		return nil, err
	}

	out := make([]viewTuple, len(views))
	for i, v := range views {
		out[i] = toViewTuple(v, statusRegistered)
	}
	return method.Outputs.Pack(out)
}

func (p Precompile) ViewCount(ctx sdk.Context, method *abi.Method) ([]byte, error) {
	return method.Outputs.Pack(new(big.Int).SetUint64(p.viewKeeper.GetViewCount(ctx)))
}

type PoolConfig struct {
	WindowSize uint64
}

func (p Precompile) RegisterDemandForView(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	// get the view address from the args
	viewAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid view address")
	}

	poolConfig := abi.ConvertType(args[1], new(PoolConfig)).(*PoolConfig)
	if poolConfig == nil {
		return nil, fmt.Errorf("invalid pool config")
	}

	// validate the view exists
	if _, found, err := p.viewKeeper.GetView(ctx, viewAddress.Hex()); err != nil {
		return nil, err
	} else if !found {
		return nil, fmt.Errorf("invalid view address")
	}

	fmt.Println(viewAddress, poolConfig.WindowSize)

	// check if the demand/pool with this demand already exists
	// we need a way to get and set pools in the view keeper
	// i want it in a way that i can get pools for a view and all pools that kinda structure
	// pools would have paericipants(hosts) we should store that in the keeper too

	/**
	we need
	Pool {
		viewAddress
		config
		hosts(array of hosts address)
	}

		the contract address of the pool should also be deterministic, so that we don't have duplicate pools for the same view and config
	*/

	// check and debit a CONSTANT from the callers wallet

	// create a contract on this call the contract is a POOL

	// return the pool contract address and view address

	return nil, nil
}
