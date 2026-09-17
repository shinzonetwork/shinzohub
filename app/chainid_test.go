package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseChainID(t *testing.T) {
	cases := []struct {
		name    string
		chainID string
		want    uint64
		wantErr bool
	}{
		{name: "devnet", chainID: "91273002", want: 91273002},
		{name: "testnet", chainID: "91273001", want: 91273001},
		{name: "empty", chainID: "", wantErr: true},
		{name: "zero", chainID: "0", wantErr: true},
		{name: "words", chainID: "testing", wantErr: true},
		{name: "named form", chainID: "shinzo_91273002-1", wantErr: true},
		{name: "negative", chainID: "-1", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseChainID(tc.chainID)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// Coin info must exist for whichever id the network runs, or the app only
// starts on one chain.
func TestChainsCoinInfo(t *testing.T) {
	for _, chainID := range []uint64{91273000, 91273001, 91273002, 91273003} {
		info := ChainsCoinInfo(chainID)
		require.Len(t, info, 1)
		coin, ok := info[chainID]
		require.True(t, ok, "no coin info for chain id %d", chainID)
		require.Equal(t, BaseDenom, coin.Denom)
		require.Equal(t, DisplayDenom, coin.DisplayDenom)
	}
}
