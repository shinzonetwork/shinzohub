package app_test

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	chainapp "github.com/shinzonetwork/shinzohub/app"
)

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
