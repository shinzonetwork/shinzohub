package decorators_test

import (
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"

	"github.com/shinzonetwork/shinzohub/app/decorators"
)

// signerMockTx adds GetSigners to MockTx so the universal sender path
// (authsigning.SigVerifiableTx) can be exercised without a full tx builder.
type signerMockTx struct {
	decorators.MockTx
	signers [][]byte
}

func (tx signerMockTx) GetSigners() ([][]byte, error)                  { return tx.signers, nil }
func (signerMockTx) GetPubKeys() ([]cryptotypes.PubKey, error)         { return nil, nil }
func (signerMockTx) GetSignaturesV2() ([]txsigning.SignatureV2, error) { return nil, nil }

func TestCollectParticipantsMsgSend(t *testing.T) {
	msg := &banktypes.MsgSend{FromAddress: "shinzo1alice", ToAddress: "shinzo1bob"}

	senders, recipients := decorators.CollectParticipants(decorators.NewMockTx(msg))

	require.Equal(t, []string{"shinzo1alice"}, senders)
	require.Equal(t, []string{"shinzo1bob"}, recipients)
}

func TestCollectParticipantsMultiSend(t *testing.T) {
	msg := &banktypes.MsgMultiSend{
		Inputs:  []banktypes.Input{{Address: "shinzo1alice"}},
		Outputs: []banktypes.Output{{Address: "shinzo1bob"}, {Address: "shinzo1carol"}},
	}

	senders, recipients := decorators.CollectParticipants(decorators.NewMockTx(msg))

	require.Equal(t, []string{"shinzo1alice"}, senders)
	require.ElementsMatch(t, []string{"shinzo1bob", "shinzo1carol"}, recipients)
}

// A self-send still produces the address once per role; the role-agnostic query
// dedups by tx hash at search time, so this is the intended shape.
func TestCollectParticipantsSelfSend(t *testing.T) {
	msg := &banktypes.MsgSend{FromAddress: "shinzo1alice", ToAddress: "shinzo1alice"}

	senders, recipients := decorators.CollectParticipants(decorators.NewMockTx(msg))

	require.Equal(t, []string{"shinzo1alice"}, senders)
	require.Equal(t, []string{"shinzo1alice"}, recipients)
}

// A message with no recipient field (here a reward withdrawal, which is not in
// the value-transfer switch) contributes its signer as the only participant.
func TestCollectParticipantsSignerOnly(t *testing.T) {
	signer := []byte("0123456789abcdef0123") // 20 bytes
	tx := signerMockTx{
		MockTx:  decorators.NewMockTx(&distrtypes.MsgWithdrawDelegatorReward{}),
		signers: [][]byte{signer},
	}

	senders, recipients := decorators.CollectParticipants(tx)

	require.Equal(t, []string{sdk.AccAddress(signer).String()}, senders)
	require.Empty(t, recipients)
}

func TestAnteHandleEmitsParticipants(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	msg := &banktypes.MsgSend{FromAddress: "shinzo1alice", ToAddress: "shinzo1bob"}

	_, err := decorators.NewTxParticipantDecorator().
		AnteHandle(ctx, decorators.NewMockTx(msg), false, decorators.EmptyAnte)
	require.NoError(t, err)

	require.Equal(t, []participant{
		{address: "shinzo1alice", role: decorators.RoleSender},
		{address: "shinzo1bob", role: decorators.RoleRecipient},
	}, participantsFromEvents(t, ctx))
}

// Simulation must stay side-effect free; no participant events are emitted.
func TestAnteHandleSkipsSimulate(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	msg := &banktypes.MsgSend{FromAddress: "shinzo1alice", ToAddress: "shinzo1bob"}

	_, err := decorators.NewTxParticipantDecorator().
		AnteHandle(ctx, decorators.NewMockTx(msg), true, decorators.EmptyAnte)
	require.NoError(t, err)

	require.Empty(t, ctx.EventManager().Events())
}

type participant struct {
	address string
	role    string
}

func participantsFromEvents(t *testing.T, ctx sdk.Context) []participant {
	t.Helper()
	var out []participant
	for _, event := range ctx.EventManager().Events() {
		if event.Type != decorators.EventTypeTxParticipant {
			continue
		}
		var p participant
		for _, attr := range event.Attributes {
			switch attr.Key {
			case decorators.AttributeKeyAddress:
				p.address = attr.Value
			case decorators.AttributeKeyRole:
				p.role = attr.Value
			}
		}
		out = append(out, p)
	}
	return out
}
