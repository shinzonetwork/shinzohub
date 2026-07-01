package app

import erc20types "github.com/cosmos/evm/x/erc20/types"

// WTokenContractMainnet is the WrappedToken contract address for mainnet
const WTokenContractMainnet = "0xD4949664cD82660AaE99bEdc034a0deA8A0bd517"

// TokenPairs registers cosmos-bank denoms that are accessible via ERC-20.
// The entry uses OWNER_MODULE so cosmos bank remains the source of truth and
// the EVM surface is a synced view.
var TokenPairs = []erc20types.TokenPair{
	{
		Erc20Address:  WTokenContractMainnet,
		Denom:         BaseDenom,
		Enabled:       true,
		ContractOwner: erc20types.OWNER_MODULE,
	},
}
