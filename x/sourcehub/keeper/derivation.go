package keeper

import (
	"crypto/ed25519"
	"fmt"

	sdkcrypto "github.com/TBD54566975/ssi-sdk/crypto"
	didkey "github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func deriveDIDFromNodeIdentityPublicKey(pubkey []byte) (string, error) {
	didDoc, err := didkey.CreateDIDKey(sdkcrypto.SECP256k1, pubkey)
	if err != nil {
		return "", fmt.Errorf("error creating did")
	}

	return didDoc.String(), nil
}

func derivePIDFromPeerKeyPublicKey(pubkey []byte) (string, error) {
	if len(pubkey) != ed25519.PublicKeySize {
		return "", fmt.Errorf("invalid ed25519 pubkey length: got %d, want %d", len(pubkey), ed25519.PublicKeySize)
	}

	lpPub, err := crypto.UnmarshalEd25519PublicKey(pubkey)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal ed25519 pubkey: %w", err)
	}

	id, err := peer.IDFromPublicKey(lpPub)
	if err != nil {
		return "", fmt.Errorf("failed to derive peer id: %w", err)
	}

	return id.String(), nil
}
