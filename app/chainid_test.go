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
