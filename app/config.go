package app

import (
	evmconfig "github.com/cosmos/evm/config"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// EVMOptionsFn defines a function type for setting app options specifically for
// the app. The function should receive the chainID and return an error if
// any.
type EVMOptionsFn func(uint64) error

// NoOpEVMOptions is a no-op function that can be used when the app does not
// need any specific configuration.
func NoOpEVMOptions(_ uint64) error {
	return nil
}

// ChainsCoinInfo is a map of the chain id and its corresponding EvmCoinInfo
// that allows initializing the app with different coin info based on the
// chain id
var ChainsCoinInfo = map[uint64]evmtypes.EvmCoinInfo{
	ChainID18Decimals: {
		Denom:         BaseDenom,
		ExtendedDenom: BaseDenom,
		DisplayDenom:  DisplayDenom,
		Decimals:      evmtypes.EighteenDecimals,
	},
}

// EVMAppOptions allows to setup the global configuration
// for the chain.
func EVMAppOptions(chainID uint64) error {
	return evmconfig.EvmAppOptionsWithConfig(chainID, ChainsCoinInfo, cosmosEVMActivators)
}
