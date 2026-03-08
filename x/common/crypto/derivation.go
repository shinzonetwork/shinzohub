package crypto

import (
	"crypto/ed25519"
	"fmt"

	sdkcrypto "github.com/TBD54566975/ssi-sdk/crypto"
	didkey "github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DeriveDID derives a DID from a secp256k1 node identity public key.
func DeriveDID(nodeIdPubkey []byte) (string, error) {
	didDoc, err := didkey.CreateDIDKey(sdkcrypto.SECP256k1, nodeIdPubkey)
	if err != nil {
		return "", fmt.Errorf("error creating did")
	}

	return didDoc.String(), nil
}

// DerivePID derives a libp2p Peer ID from an Ed25519 peer key public key.
func DerivePID(peerKeyPubkey []byte) (string, error) {
	if len(peerKeyPubkey) != ed25519.PublicKeySize {
		return "", fmt.Errorf("invalid ed25519 pubkey length: got %d, want %d", len(peerKeyPubkey), ed25519.PublicKeySize)
	}

	lpPub, err := crypto.UnmarshalEd25519PublicKey(peerKeyPubkey)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal ed25519 pubkey: %w", err)
	}

	id, err := peer.IDFromPublicKey(lpPub)
	if err != nil {
		return "", fmt.Errorf("failed to derive peer id: %w", err)
	}

	return id.String(), nil
}
