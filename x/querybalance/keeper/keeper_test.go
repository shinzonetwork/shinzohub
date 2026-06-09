package keeper_test

import (
	"context"
	"crypto/sha256"
	"testing"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/stretchr/testify/require"

	commoncrypto "github.com/shinzonetwork/shinzohub/x/common/crypto"
	qbkeeper "github.com/shinzonetwork/shinzohub/x/querybalance/keeper"
	"github.com/shinzonetwork/shinzohub/x/querybalance/types"
)

const testDenom = "anzo"

type bankMove struct {
	kind  string
	from  string
	to    string
	coins sdk.Coins
}

type mockBankKeeper struct {
	moves      []bankMove
	failNextIn bool
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ sdk.Context, from sdk.AccAddress, mod string, amt sdk.Coins) error {
	if m.failNextIn {
		m.failNextIn = false
		return errMock
	}
	m.moves = append(m.moves, bankMove{kind: "in", from: from.String(), to: mod, coins: amt})
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ sdk.Context, mod string, to sdk.AccAddress, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "out", from: mod, to: to.String(), coins: amt})
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ sdk.Context, from, to string, amt sdk.Coins) error {
	m.moves = append(m.moves, bankMove{kind: "modmod", from: from, to: to, coins: amt})
	return nil
}

var errMock = &mockErr{msg: "mock failure"}

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }

type mockStakingKeeper struct{ denom string }

func (m *mockStakingKeeper) BondDenom(_ context.Context) (string, error) {
	return m.denom, nil
}

type fixture struct {
	t      *testing.T
	ctx    sdk.Context
	keeper qbkeeper.Keeper
	bank   *mockBankKeeper
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bank := &mockBankKeeper{}
	stk := &mockStakingKeeper{denom: testDenom}

	k := qbkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		bank,
		stk,
		"authority",
	)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, cosmoslog.NewNopLogger())

	return &fixture{t: t, ctx: ctx, keeper: k, bank: bank}
}

func nzo(amount int64) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(amount)))
}

func addr(b byte) sdk.AccAddress {
	out := make([]byte, 20)
	for i := range out {
		out[i] = b
	}
	return out
}

func signedKey(t *testing.T, message []byte) (pubkey, signature []byte) {
	t.Helper()
	priv, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	h := sha256.Sum256(message)
	return priv.PubKey().SerializeUncompressed(), ecdsa.Sign(priv, h[:]).Serialize()
}

func TestFund_CreditsAndMovesCoins(t *testing.T) {
	f := newFixture(t)
	funder := addr(1)

	require.NoError(t, f.keeper.Fund(f.ctx, funder, "did:test:1", nzo(100)))

	require.Equal(t, math.NewInt(100), f.keeper.GetBalance(f.ctx, "did:test:1"))
	require.Len(t, f.bank.moves, 1)
	require.Equal(t, "in", f.bank.moves[0].kind)
	require.Equal(t, funder.String(), f.bank.moves[0].from)
	require.Equal(t, types.ModuleName, f.bank.moves[0].to)
}

func TestFund_AccumulatesAcrossCalls(t *testing.T) {
	f := newFixture(t)

	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), "did:x", nzo(50)))
	require.NoError(t, f.keeper.Fund(f.ctx, addr(2), "did:x", nzo(30)))

	require.Equal(t, math.NewInt(80), f.keeper.GetBalance(f.ctx, "did:x"))
}

func TestFund_RejectsEmptyDID(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Fund(f.ctx, addr(1), "", nzo(100))
	require.ErrorContains(t, err, "did is required")
}

func TestFund_RejectsZeroAmount(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Fund(f.ctx, addr(1), "did:x", sdk.Coins{})
	require.ErrorContains(t, err, "positive coin")
}

func TestFund_RejectsMultipleDenoms(t *testing.T) {
	f := newFixture(t)
	mixed := sdk.NewCoins(
		sdk.NewCoin(testDenom, math.NewInt(10)),
		sdk.NewCoin("other", math.NewInt(10)),
	)

	err := f.keeper.Fund(f.ctx, addr(1), "did:x", mixed)
	require.ErrorContains(t, err, "single coin denomination")
}

func TestFund_BankFailureBubbles(t *testing.T) {
	f := newFixture(t)
	f.bank.failNextIn = true

	err := f.keeper.Fund(f.ctx, addr(1), "did:x", nzo(100))
	require.ErrorContains(t, err, "transfer to module account")
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, "did:x"))
}

func TestClaim_SetsOwner(t *testing.T) {
	f := newFixture(t)
	claimer := addr(9)
	message := []byte("claim-nonce-1")
	pub, sig := signedKey(t, message)

	did, err := f.keeper.Claim(f.ctx, claimer, pub, sig, message)
	require.NoError(t, err)
	require.NotEmpty(t, did)

	entry, found := f.keeper.GetEntry(f.ctx, did)
	require.True(t, found)
	require.Equal(t, claimer.String(), entry.Owner)
	require.Equal(t, int64(1), entry.ClaimedAt)
}

func TestClaim_PreservesExistingBalance(t *testing.T) {
	f := newFixture(t)
	message := []byte("claim-nonce-2")
	pub, sig := signedKey(t, message)
	did := mustDeriveDID(t, pub)

	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), did, nzo(500)))

	gotDid, err := f.keeper.Claim(f.ctx, addr(9), pub, sig, message)
	require.NoError(t, err)
	require.Equal(t, did, gotDid)

	require.Equal(t, math.NewInt(500), f.keeper.GetBalance(f.ctx, did))
	entry, _ := f.keeper.GetEntry(f.ctx, did)
	require.Equal(t, addr(9).String(), entry.Owner)
}

func TestClaim_IdempotentForSameOwner(t *testing.T) {
	f := newFixture(t)
	message := []byte("claim-nonce-3")
	pub, sig := signedKey(t, message)

	owner := addr(9)
	did1, err := f.keeper.Claim(f.ctx, owner, pub, sig, message)
	require.NoError(t, err)

	did2, err := f.keeper.Claim(f.ctx, owner, pub, sig, message)
	require.NoError(t, err)
	require.Equal(t, did1, did2)
}

func TestClaim_RejectsDifferentOwner(t *testing.T) {
	f := newFixture(t)
	message := []byte("claim-nonce-4")
	pub, sig := signedKey(t, message)

	_, err := f.keeper.Claim(f.ctx, addr(9), pub, sig, message)
	require.NoError(t, err)

	_, err = f.keeper.Claim(f.ctx, addr(10), pub, sig, message)
	require.ErrorContains(t, err, "already claimed")
}

func TestClaim_RejectsBadSignature(t *testing.T) {
	f := newFixture(t)
	message := []byte("genuine-message")
	pub, _ := signedKey(t, message)

	_, otherSig := signedKey(t, message)

	_, err := f.keeper.Claim(f.ctx, addr(9), pub, otherSig, message)
	require.ErrorContains(t, err, "signature verification")
}

func TestWithdraw_HappyPath(t *testing.T) {
	f := newFixture(t)
	message := []byte("withdraw-nonce")
	pub, sig := signedKey(t, message)
	did := mustDeriveDID(t, pub)

	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), did, nzo(500)))
	_, err := f.keeper.Claim(f.ctx, addr(9), pub, sig, message)
	require.NoError(t, err)

	require.NoError(t, f.keeper.Withdraw(f.ctx, addr(9), did, math.NewInt(200)))

	require.Equal(t, math.NewInt(300), f.keeper.GetBalance(f.ctx, did))

	outMove := f.bank.moves[len(f.bank.moves)-1]
	require.Equal(t, "out", outMove.kind)
	require.Equal(t, addr(9).String(), outMove.to)
	require.Equal(t, math.NewInt(200), outMove.coins[0].Amount)
}

func TestWithdraw_RejectsNonOwner(t *testing.T) {
	f := newFixture(t)
	message := []byte("withdraw-nonowner")
	pub, sig := signedKey(t, message)
	did := mustDeriveDID(t, pub)

	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), did, nzo(500)))
	_, err := f.keeper.Claim(f.ctx, addr(9), pub, sig, message)
	require.NoError(t, err)

	err = f.keeper.Withdraw(f.ctx, addr(10), did, math.NewInt(100))
	require.ErrorContains(t, err, "not the owner")
}

func TestWithdraw_RejectsUnclaimed(t *testing.T) {
	f := newFixture(t)
	did := "did:test:unclaimed"
	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), did, nzo(500)))

	err := f.keeper.Withdraw(f.ctx, addr(9), did, math.NewInt(100))
	require.ErrorContains(t, err, "not claimed")
}

func TestWithdraw_RejectsInsufficientBalance(t *testing.T) {
	f := newFixture(t)
	message := []byte("withdraw-broke")
	pub, sig := signedKey(t, message)
	did := mustDeriveDID(t, pub)

	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), did, nzo(50)))
	_, err := f.keeper.Claim(f.ctx, addr(9), pub, sig, message)
	require.NoError(t, err)

	err = f.keeper.Withdraw(f.ctx, addr(9), did, math.NewInt(100))
	require.ErrorContains(t, err, "insufficient balance")
}

func TestDebit_DeductsAndPreservesOwner(t *testing.T) {
	f := newFixture(t)
	message := []byte("debit-nonce")
	pub, sig := signedKey(t, message)
	did := mustDeriveDID(t, pub)

	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), did, nzo(500)))
	_, err := f.keeper.Claim(f.ctx, addr(9), pub, sig, message)
	require.NoError(t, err)

	require.NoError(t, f.keeper.Debit(f.ctx, did, math.NewInt(200)))

	require.Equal(t, math.NewInt(300), f.keeper.GetBalance(f.ctx, did))
	entry, _ := f.keeper.GetEntry(f.ctx, did)
	require.Equal(t, addr(9).String(), entry.Owner)
}

func TestDebit_RejectsInsufficient(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, f.keeper.Fund(f.ctx, addr(1), "did:x", nzo(50)))

	err := f.keeper.Debit(f.ctx, "did:x", math.NewInt(100))
	require.ErrorContains(t, err, "insufficient balance")
}

func TestDebit_RejectsUnknownDID(t *testing.T) {
	f := newFixture(t)

	err := f.keeper.Debit(f.ctx, "did:never-seen", math.NewInt(1))
	require.ErrorContains(t, err, "no balance")
}

func TestGetBalance_ZeroForUnknown(t *testing.T) {
	f := newFixture(t)
	require.Equal(t, math.ZeroInt(), f.keeper.GetBalance(f.ctx, "did:unknown"))
}

func TestGenesis_RoundTrip(t *testing.T) {
	src := newFixture(t)
	require.NoError(t, src.keeper.Fund(src.ctx, addr(1), "did:a", nzo(100)))
	require.NoError(t, src.keeper.Fund(src.ctx, addr(2), "did:b", nzo(250)))

	exported := src.keeper.ExportGenesis(src.ctx)

	dst := newFixture(t)
	dst.keeper.InitGenesis(dst.ctx, *exported)

	require.Equal(t, math.NewInt(100), dst.keeper.GetBalance(dst.ctx, "did:a"))
	require.Equal(t, math.NewInt(250), dst.keeper.GetBalance(dst.ctx, "did:b"))
}

func mustDeriveDID(t *testing.T, pubkey []byte) string {
	t.Helper()
	did, err := commoncrypto.DeriveDID(pubkey)
	require.NoError(t, err)
	return did
}
