package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// TestDeriveDID_Compressed pins the derived did:key for a known key and checks that
// the compressed and uncompressed encodings of that key derive the same value. The
// secp256k1 did:key multicodec (0xe7) is defined over the compressed key, so both
// must produce the compressed form (zQ3s...), not the uncompressed one (z7r8...).
func TestDeriveDID_Compressed(t *testing.T) {
	const (
		compressedHex = "02c2ca8868cd31f0c0ca8ffd3a58baa945623b1f0d6245611f27a43d954541a04c"
		wantDID       = "did:key:zQ3shaXAyH7cPt1SiemqWtwXTt47EUWvCucxXmg1asUPdNk6P"
	)
	compressed, err := hex.DecodeString(compressedHex)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := secp256k1.ParsePubKey(compressed)
	if err != nil {
		t.Fatal(err)
	}

	for name, in := range map[string][]byte{
		"compressed":   compressed,
		"uncompressed": pk.SerializeUncompressed(),
	} {
		got, err := DeriveDID(in)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != wantDID {
			t.Fatalf("%s input:\n want %s\n got  %s", name, wantDID, got)
		}
	}
}

func TestDeriveDID_InvalidPubkey(t *testing.T) {
	if _, err := DeriveDID([]byte{0x00, 0x01, 0x02}); err == nil {
		t.Fatal("want an error for an invalid pubkey")
	}
}
