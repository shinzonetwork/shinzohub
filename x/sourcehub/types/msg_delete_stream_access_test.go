package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// validBech32 is derived at runtime so the test does not depend on a
// specific Bech32 prefix being registered with the SDK config. The
// production binary sets a custom "shinzo" prefix during cmd init; this
// test package runs without that init, so the SDK default ("cosmos") is
// what AccAddressFromBech32 will accept.
var validBech32 = sdk.AccAddress(make([]byte, 20)).String()

const (
	validDID    = "did:key:zQ3sample"
	validStream = "0xc5d55f9a4e8788abaaf74d4772c2a4afe60a23a3"
)

// ValidateBasic must accept a well-formed delete request: bech32 signer,
// non-empty stream id, non-empty DID. The handler relies on these
// guarantees and would otherwise need its own checks.
func TestMsgDeleteStreamAccess_ValidateBasic_Ok(t *testing.T) {
	msg := &MsgDeleteStreamAccess{
		Signer:   validBech32,
		StreamId: validStream,
		Did:      validDID,
		Resource: Resource_RESOURCE_VIEW,
	}
	require.NoError(t, msg.ValidateBasic())
}

// A malformed signer address must be rejected before the handler runs;
// otherwise the keeper would panic in GetSigners.
func TestMsgDeleteStreamAccess_ValidateBasic_BadSigner(t *testing.T) {
	msg := &MsgDeleteStreamAccess{
		Signer:   "not-a-bech32-address",
		StreamId: validStream,
		Did:      validDID,
	}
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid signer address")
}

// Empty stream id has no meaningful tuple to revoke.
func TestMsgDeleteStreamAccess_ValidateBasic_EmptyStreamID(t *testing.T) {
	msg := &MsgDeleteStreamAccess{
		Signer:   validBech32,
		StreamId: "",
		Did:      validDID,
	}
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "stream id")
}

// Empty DID has no actor to remove the subscriber tuple from.
func TestMsgDeleteStreamAccess_ValidateBasic_EmptyDID(t *testing.T) {
	msg := &MsgDeleteStreamAccess{
		Signer:   validBech32,
		StreamId: validStream,
		Did:      "",
	}
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "did")
}

// GetSigners must return the signer derived from the Bech32 field; the
// SDK uses this for authentication of the carrying tx.
func TestMsgDeleteStreamAccess_GetSigners(t *testing.T) {
	msg := &MsgDeleteStreamAccess{
		Signer:   validBech32,
		StreamId: validStream,
		Did:      validDID,
	}
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, validBech32, signers[0].String())
}

// GetSigners panics on a malformed signer because the SDK invariant is
// that ValidateBasic has already accepted the message before any keeper
// path reaches GetSigners. The panic guards against a violation of that
// invariant elsewhere.
func TestMsgDeleteStreamAccess_GetSigners_PanicsOnBadAddress(t *testing.T) {
	msg := &MsgDeleteStreamAccess{Signer: "not-a-bech32"}
	require.Panics(t, func() { _ = msg.GetSigners() })
}
