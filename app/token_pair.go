package app

import erc20types "github.com/cosmos/evm/x/erc20/types"

// WTokenContractMainnet is the WrappedToken contract address for mainnet
const WTokenContractMainnet = "0xD4949664cD82660AaE99bEdc034a0deA8A0bd517"

// ShinusdBaseDenom is the base unit (micro-SHINUSD) of the SHINUSD stablecoin.
// 1 SHINUSD = 1,000,000 ushinusd (6 decimals, USDC-style).
const ShinusdBaseDenom = "ushinusd"

// ShinusdDisplayDenom is the human-facing display denomination.
const ShinusdDisplayDenom = "shinusd"

// ShinusdSymbol is the SHINUSD ticker symbol.
const ShinusdSymbol = "SHINUSD"

// ShinusdNativePrecompile is the fixed address where the SHINUSD ERC-20 native
// precompile lives. EVM callers interact with SHINUSD through this address; the
const ShinusdNativePrecompile = "0x0000000000000000000000000000000000000215"

// TokenPairs registers cosmos-bank denoms that are accessible via ERC-20.
// Both entries use OWNER_MODULE so cosmos bank remains the source of truth and
// the EVM surface is a synced view.
var TokenPairs = []erc20types.TokenPair{
	{
		Erc20Address:  WTokenContractMainnet,
		Denom:         BaseDenom,
		Enabled:       true,
		ContractOwner: erc20types.OWNER_MODULE,
	},
	{
		Erc20Address:  ShinusdNativePrecompile,
		Denom:         ShinusdBaseDenom,
		Enabled:       true,
		ContractOwner: erc20types.OWNER_MODULE,
	},
}
