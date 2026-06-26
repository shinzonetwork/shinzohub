package decorators

import (
	feegrant "cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// Event type and attribute keys for the participant index. The KV indexer stores
// the composite tag "<type>.<key>", so the query key is "shinzo.address". A single
// equality condition matches any occurrence, so one event per participant lets
// "shinzo.address='X'" match X as a sender or a recipient.
const (
	EventTypeTxParticipant = "shinzo"
	AttributeKeyAddress    = "address"
	AttributeKeyRole       = "role"
	RoleSender             = "sender"
	RoleRecipient          = "recipient"
)

// TxParticipantDecorator emits one indexed `shinzo` event per distinct
// participant address in a Cosmos tx, so a single `shinzo.address='X'` tx_search
// matches X as a sender or a recipient and pages natively.
//
// It emits from the ante path because ante events are indexed even when message
// execution later fails, which keeps failed txs queryable by participant; it
// skips simulation, which must stay side-effect free.
//
// The event is per participant because the KV indexer intersects ANDed conditions
// within a single event (by event sequence): each address must carry its own role
// for `shinzo.address='X' AND shinzo.role='sender'` to mean X was the sender.
// Collapsing all participants into one event would break that.
type TxParticipantDecorator struct{}

// NewTxParticipantDecorator returns the participant-emitting ante decorator.
func NewTxParticipantDecorator() TxParticipantDecorator {
	return TxParticipantDecorator{}
}

func (TxParticipantDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !simulate {
		senders, recipients := CollectParticipants(tx)
		EmitParticipants(ctx, senders, recipients)
	}
	return next(ctx, tx, simulate)
}

// CollectParticipants returns the distinct sender and recipient addresses of a
// tx. Senders are the tx signers (universal across every message type) unioned
// with the from-side of any value-transfer message, including authz-wrapped
// ones. Recipients are the to-side of those messages. Returned slices are
// deduplicated and in first-seen order.
func CollectParticipants(tx sdk.Tx) (senders, recipients []string) {
	s := newAddrSet()
	r := newAddrSet()

	// Universal senders: the tx signers, available for any message type via the
	// cosmos.msg.v1.signer annotation. Recipients have no such uniform interface,
	// so they are pulled per message type below.
	if sigTx, ok := tx.(authsigning.SigVerifiableTx); ok {
		if signers, err := sigTx.GetSigners(); err == nil {
			for _, signer := range signers {
				s.add(sdk.AccAddress(signer).String())
			}
		}
	}

	collectFromMsgs(tx.GetMsgs(), s, r)
	return s.list(), r.list()
}

// collectFromMsgs records the from/to addresses of the value-transfer message
// types. authz.MsgExec is unwrapped one level so inner participants are captured.
// Types with no recipient (votes, signer-only custom messages) appear only
// through their signer in CollectParticipants, so they are not listed here.
func collectFromMsgs(msgs []sdk.Msg, senders, recipients *addrSet) {
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *banktypes.MsgSend:
			senders.add(m.FromAddress)
			recipients.add(m.ToAddress)
		case *banktypes.MsgMultiSend:
			for _, in := range m.Inputs {
				senders.add(in.Address)
			}
			for _, out := range m.Outputs {
				recipients.add(out.Address)
			}
		case *ibctransfertypes.MsgTransfer:
			senders.add(m.Sender)
			recipients.add(m.Receiver)
		case *distrtypes.MsgSetWithdrawAddress:
			recipients.add(m.WithdrawAddress)
		case *authz.MsgGrant:
			recipients.add(m.Grantee)
		case *feegrant.MsgGrantAllowance:
			recipients.add(m.Grantee)
		case *authz.MsgExec:
			if inner, err := m.GetMessages(); err == nil {
				collectFromMsgs(inner, senders, recipients)
			}
		}
	}
}

// EmitParticipants writes one `shinzo` event per (address, role). An address that
// is both sender and recipient gets one event per role; tx_search dedups by hash,
// so the role-agnostic query still returns the tx once. Exported for the EVM ante
// path, which emits the same events.
func EmitParticipants(ctx sdk.Context, senders, recipients []string) {
	for _, addr := range senders {
		emitParticipant(ctx, addr, RoleSender)
	}
	for _, addr := range recipients {
		emitParticipant(ctx, addr, RoleRecipient)
	}
}

func emitParticipant(ctx sdk.Context, address, role string) {
	if address == "" {
		return
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeTxParticipant,
		sdk.NewAttribute(AttributeKeyAddress, address),
		sdk.NewAttribute(AttributeKeyRole, role),
	))
}

// addrSet is an insertion-ordered string set, so emitted events are
// deterministic for a given tx.
type addrSet struct {
	seen  map[string]struct{}
	order []string
}

func newAddrSet() *addrSet { return &addrSet{seen: make(map[string]struct{})} }

func (a *addrSet) add(addr string) {
	if addr == "" {
		return
	}
	if _, ok := a.seen[addr]; ok {
		return
	}
	a.seen[addr] = struct{}{}
	a.order = append(a.order, addr)
}

func (a *addrSet) list() []string { return a.order }
