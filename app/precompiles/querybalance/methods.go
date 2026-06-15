package querybalance

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	MethodFund      = "fund"
	MethodBalanceOf = "balanceOf"
)

func (p Precompile) Fund(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	did, ok := args[0].(string)
	if !ok || did == "" {
		return nil, fmt.Errorf("invalid did")
	}

	value := contract.Value()
	if value == nil || value.Sign() == 0 {
		return nil, fmt.Errorf("must send a non-zero amount")
	}

	bondDenom, err := p.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, fmt.Errorf("look up bond denom: %w", err)
	}

	amount := math.NewIntFromBigInt(value.ToBig())
	coins := sdk.NewCoins(sdk.NewCoin(bondDenom, amount))

	funder := sdk.AccAddress(contract.Caller().Bytes())
	if err := p.qbKeeper.Fund(ctx, funder, did, coins); err != nil {
		return nil, fmt.Errorf("fund: %w", err)
	}

	emitFunded(stateDB, contract.Address(), did, contract.Caller(), value.ToBig())

	return method.Outputs.Pack()
}

func (p Precompile) BalanceOf(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	did, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid did")
	}

	balance := p.qbKeeper.GetBalance(ctx, did)
	return method.Outputs.Pack(balance.BigInt())
}

func emitFunded(
	stateDB vm.StateDB,
	precompileAddr common.Address,
	did string,
	funder common.Address,
	amount *big.Int,
) {
	topic0 := crypto.Keccak256Hash([]byte("Funded(string,address,uint256)"))
	dataArgs := abi.Arguments{
		{Type: mustABIType("string")},
		{Type: mustABIType("uint256")},
	}
	data, _ := dataArgs.Pack(did, amount)
	stateDB.AddLog(&gethtypes.Log{
		Address: precompileAddr,
		Topics: []common.Hash{
			topic0,
			common.BytesToHash(funder.Bytes()),
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
