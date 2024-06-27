package did

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/require"
)

func TestSECP256K1ToDIDThenResolve(t *testing.T) {
	t.Skip("secp256k1 resolver is currently not supported by the ssi-sdk lib")
	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()

	did, err := DIDFromPubKey(pub)
	require.NoError(t, err)

	resolver := KeyResolver{}
	_, err = resolver.Resolve(context.Background(), did)
	require.NoError(t, err)
}

func TestED25519ToDIDThenResolve(t *testing.T) {
	priv := ed25519.GenPrivKey()
	pub := priv.PubKey()

	did, err := DIDFromPubKey(pub)
	require.NoError(t, err)

	resolver := KeyResolver{}
	_, err = resolver.Resolve(context.Background(), did)
	require.NoError(t, err)
}
