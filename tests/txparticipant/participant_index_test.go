// Package txparticipant validates the participant-index design end to end against
// the real CometBFT v0.38.19 KV transaction indexer. It indexes synthetic tx
// results carrying the `shinzo` participant events the ante decorators emit, then
// runs tx_search queries to confirm the query behavior the index must support:
//
//   - shinzo.address='X' matches a tx whether X was a sender or a recipient, and
//     a self-transfer returns once (OR semantics + dedup).
//   - shinzo.address='X' AND shinzo.role='sender' returns only txs where X was
//     the sender, because address and role are bound within a single event.
//   - the rejected tx-level schema (all addresses + all roles on one event)
//     produces a false positive on the role query, which is why the decorators
//     emit one event per participant instead.
package txparticipant

import (
	"context"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/state/txindex/kv"
	"github.com/stretchr/testify/require"
)

func p(address, role string) [2]string { return [2]string{address, role} }

// participantTx builds a tx result in the shape the ante decorators produce: one
// `shinzo` event per participant, each carrying that participant's address and role.
func participantTx(height int64, index uint32, tx string, participants ...[2]string) *abci.TxResult {
	events := make([]abci.Event, 0, len(participants))
	for _, participant := range participants {
		events = append(events, abci.Event{
			Type: "shinzo",
			Attributes: []abci.EventAttribute{
				{Key: "address", Value: participant[0], Index: true},
				{Key: "role", Value: participant[1], Index: true},
			},
		})
	}
	return &abci.TxResult{
		Height: height,
		Index:  index,
		Tx:     []byte(tx),
		Result: abci.ExecTxResult{Code: abci.CodeTypeOK, Events: events},
	}
}

// txLevelTx builds the rejected schema: a single tx-level event listing every
// address and every role as separate attributes.
func txLevelTx(height int64, index uint32, tx string, addresses, roles []string) *abci.TxResult {
	attrs := make([]abci.EventAttribute, 0, len(addresses)+len(roles))
	for _, address := range addresses {
		attrs = append(attrs, abci.EventAttribute{Key: "address", Value: address, Index: true})
	}
	for _, role := range roles {
		attrs = append(attrs, abci.EventAttribute{Key: "role", Value: role, Index: true})
	}
	return &abci.TxResult{
		Height: height,
		Index:  index,
		Tx:     []byte(tx),
		Result: abci.ExecTxResult{
			Code:   abci.CodeTypeOK,
			Events: []abci.Event{{Type: "shinzo", Attributes: attrs}},
		},
	}
}

func search(t *testing.T, idx *kv.TxIndex, q string) []string {
	t.Helper()
	parsed, err := query.New(q)
	require.NoError(t, err)
	results, err := idx.Search(context.Background(), parsed)
	require.NoError(t, err)

	txs := make([]string, 0, len(results))
	for _, result := range results {
		txs = append(txs, string(result.Tx))
	}
	return txs
}

func TestParticipantIndex(t *testing.T) {
	idx := kv.NewTxIndex(dbm.NewMemDB())
	require.NoError(t, idx.Index(participantTx(1, 0, "tx1", p("alice", "sender"), p("bob", "recipient"))))
	require.NoError(t, idx.Index(participantTx(2, 0, "tx2", p("carol", "sender"), p("alice", "recipient"))))
	require.NoError(t, idx.Index(participantTx(3, 0, "tx3", p("alice", "sender"), p("alice", "recipient"))))
	require.NoError(t, idx.Index(participantTx(4, 0, "tx4", p("dave", "sender"), p("erin", "recipient"))))

	// Single equality OR-matches across roles, and the self-transfer (tx3) is
	// returned once.
	require.ElementsMatch(t, []string{"tx1", "tx2", "tx3"},
		search(t, idx, "shinzo.address='alice'"))

	// Role binds to the address within one event: alice as sender is tx1 and tx3,
	// not tx2 (where alice was the recipient).
	require.ElementsMatch(t, []string{"tx1", "tx3"},
		search(t, idx, "shinzo.address='alice' AND shinzo.role='sender'"))
	require.ElementsMatch(t, []string{"tx2", "tx3"},
		search(t, idx, "shinzo.address='alice' AND shinzo.role='recipient'"))

	// An address that never appears returns nothing.
	require.Empty(t, search(t, idx, "shinzo.address='zoe'"))
}

// TestTxLevelSchemaFalsePositive demonstrates the failure mode of the rejected
// schema and is the empirical reason the decorators emit one event per participant.
func TestTxLevelSchemaFalsePositive(t *testing.T) {
	idx := kv.NewTxIndex(dbm.NewMemDB())
	// carol -> alice, so alice is the recipient. Emitted as one tx-level event.
	require.NoError(t, idx.Index(txLevelTx(1, 0, "tx5",
		[]string{"carol", "alice"},
		[]string{"sender", "recipient"})))

	// The role query wrongly matches: address='alice' and role='sender' both have
	// a matching value in the same event, even though alice was not the sender.
	require.ElementsMatch(t, []string{"tx5"},
		search(t, idx, "shinzo.address='alice' AND shinzo.role='sender'"))

	// The same logical tx under the per-participant schema is correctly excluded.
	perParticipant := kv.NewTxIndex(dbm.NewMemDB())
	require.NoError(t, perParticipant.Index(participantTx(1, 0, "tx5",
		p("carol", "sender"), p("alice", "recipient"))))
	require.Empty(t,
		search(t, perParticipant, "shinzo.address='alice' AND shinzo.role='sender'"))
}
