package types_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

func acc(b byte) string {
	return sdk.AccAddress(bytes.Repeat([]byte{b}, 20)).String()
}

func validAssertion() *types.MsgIndexerAssertion {
	return &types.MsgIndexerAssertion{
		Signer:             acc(0xAA),
		SourceChain:        "ethereum",
		SourceChainId:      1,
		ValidatorPubkey:    bytes.Repeat([]byte{0x01}, 33),
		AssertionAuthority: []byte("withdrawal-W"),
		Nonce:              1,
		ChainSpecific:      []byte("audit-bytes"),
		OperatorAddress:    acc(0x01),
		PayoutAddress:      acc(0x02),
	}
}

func TestMsgIndexerAssertion_SizeBounds(t *testing.T) {
	require.NoError(t, validAssertion().ValidateBasic())

	t.Run("pubkey at cap is allowed", func(t *testing.T) {
		m := validAssertion()
		m.ValidatorPubkey = bytes.Repeat([]byte{0x01}, types.MaxValidatorPubkeyLen)
		require.NoError(t, m.ValidateBasic())
	})

	t.Run("pubkey over cap is rejected", func(t *testing.T) {
		m := validAssertion()
		m.ValidatorPubkey = bytes.Repeat([]byte{0x01}, types.MaxValidatorPubkeyLen+1)
		require.ErrorContains(t, m.ValidateBasic(), "validator_pubkey too large")
	})

	t.Run("chain_specific at cap is allowed", func(t *testing.T) {
		m := validAssertion()
		m.ChainSpecific = bytes.Repeat([]byte{0x01}, types.MaxChainSpecificLen)
		require.NoError(t, m.ValidateBasic())
	})

	t.Run("chain_specific over cap is rejected", func(t *testing.T) {
		m := validAssertion()
		m.ChainSpecific = bytes.Repeat([]byte{0x01}, types.MaxChainSpecificLen+1)
		require.ErrorContains(t, m.ValidateBasic(), "chain_specific too large")
	})
}

func TestMsgSetPayout_PubkeySizeBound(t *testing.T) {
	base := &types.MsgSetPayout{
		Signer:          acc(0xAA),
		SourceChainId:   1,
		ValidatorPubkey: bytes.Repeat([]byte{0x01}, 33),
		PayoutAddress:   acc(0x02),
		Nonce:           1,
	}
	require.NoError(t, base.ValidateBasic())

	base.ValidatorPubkey = bytes.Repeat([]byte{0x01}, types.MaxValidatorPubkeyLen+1)
	require.ErrorContains(t, base.ValidateBasic(), "validator_pubkey too large")
}

func TestMsgRevokeIndexer_PubkeySizeBound(t *testing.T) {
	base := &types.MsgRevokeIndexer{
		Signer:          acc(0xAA),
		SourceChainId:   1,
		ValidatorPubkey: bytes.Repeat([]byte{0x01}, 33),
		Nonce:           1,
	}
	require.NoError(t, base.ValidateBasic())

	base.ValidatorPubkey = bytes.Repeat([]byte{0x01}, types.MaxValidatorPubkeyLen+1)
	require.ErrorContains(t, base.ValidateBasic(), "validator_pubkey too large")
}
