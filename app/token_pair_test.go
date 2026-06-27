package app_test

import (
	"strings"
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	chainapp "github.com/shinzonetwork/shinzohub/app"
)

func TestTokenPairs_ShinusdEntryWellFormed(t *testing.T) {
	var shinusd *erc20types.TokenPair
	for i := range chainapp.TokenPairs {
		if chainapp.TokenPairs[i].Denom == chainapp.ShinusdBaseDenom {
			shinusd = &chainapp.TokenPairs[i]
			break
		}
	}

	require.NotNil(t, shinusd, "TokenPairs missing the SHINUSD entry")
	require.Equal(t, chainapp.ShinusdNativePrecompile, shinusd.Erc20Address)
	require.Equal(t, chainapp.ShinusdBaseDenom, shinusd.Denom)
	require.True(t, shinusd.Enabled)
	require.Equal(t, erc20types.OWNER_MODULE, shinusd.ContractOwner)
}

func TestTokenPairs_NoDuplicates(t *testing.T) {
	seenDenoms := map[string]struct{}{}
	seenAddrs := map[string]struct{}{}

	for _, pair := range chainapp.TokenPairs {
		_, dupDenom := seenDenoms[pair.Denom]
		require.False(t, dupDenom, "duplicate denom in TokenPairs: %s", pair.Denom)
		seenDenoms[pair.Denom] = struct{}{}

		addr := strings.ToLower(pair.Erc20Address)
		_, dupAddr := seenAddrs[addr]
		require.False(t, dupAddr, "duplicate erc20 address in TokenPairs: %s", addr)
		seenAddrs[addr] = struct{}{}
	}
}

func TestTokenPairs_AddressesAreValidHex(t *testing.T) {
	for _, pair := range chainapp.TokenPairs {
		require.True(t,
			common.IsHexAddress(pair.Erc20Address),
			"erc20 address is not valid hex for denom %s: %q", pair.Denom, pair.Erc20Address,
		)
	}
}

func TestDefaultGenesis_RegistersShinusdTokenPair(t *testing.T) {
	gapp := chainapp.Setup(t)
	genesis := gapp.DefaultGenesis()

	rawErc20, ok := genesis[erc20types.ModuleName]
	require.True(t, ok, "default genesis missing the erc20 module state")

	var erc20State erc20types.GenesisState
	require.NoError(t, gapp.AppCodec().UnmarshalJSON(rawErc20, &erc20State))

	var found bool
	for _, pair := range erc20State.TokenPairs {
		if pair.Denom == chainapp.ShinusdBaseDenom {
			found = true
			require.Equal(t, chainapp.ShinusdNativePrecompile, pair.Erc20Address)
			require.True(t, pair.Enabled)
			break
		}
	}
	require.True(t, found, "SHINUSD token pair not in default genesis")

	require.Contains(t, erc20State.NativePrecompiles, chainapp.ShinusdNativePrecompile,
		"SHINUSD precompile address not registered in NativePrecompiles")
}

func TestDefaultGenesis_RegistersShinusdDenomMetadata(t *testing.T) {
	gapp := chainapp.Setup(t)
	genesis := gapp.DefaultGenesis()

	rawBank, ok := genesis[banktypes.ModuleName]
	require.True(t, ok, "default genesis missing the bank module state")

	var bankState banktypes.GenesisState
	require.NoError(t, gapp.AppCodec().UnmarshalJSON(rawBank, &bankState))

	var found bool
	for _, m := range bankState.DenomMetadata {
		if m.Base == chainapp.ShinusdBaseDenom {
			found = true
			require.Equal(t, chainapp.ShinusdDisplayDenom, m.Display)
			require.Equal(t, chainapp.ShinusdSymbol, m.Symbol)
			require.Equal(t, "Shinzo USD", m.Name)
			break
		}
	}
	require.True(t, found, "SHINUSD denom metadata not in default bank genesis")
}
