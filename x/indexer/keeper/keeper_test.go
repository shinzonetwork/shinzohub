package keeper_test

import (
	"crypto/sha256"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cosmoslog "cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/shinzonetwork/shinzohub/x/indexer/keeper"
	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

type mockAdminKeeper struct {
	admins map[string]bool
}

func (m *mockAdminKeeper) IsAdmin(_ sdk.Context, address string) bool {
	return m.admins[address]
}

type mockSourcehubKeeper struct {
	setCalls       int
	setAndDelCalls int
	lastDid        string
	lastPrev       string
	lastGroup      string
	lastReq        string
	err            error
}

func (m *mockSourcehubKeeper) calls() int { return m.setCalls + m.setAndDelCalls }

func (m *mockSourcehubKeeper) SendICASetRelationship(
	_ sdk.Context,
	did string,
	group string,
	requestor string,
) (uint64, string, string, error) {
	m.setCalls++
	m.lastDid = did
	m.lastPrev = ""
	m.lastGroup = group
	m.lastReq = requestor
	return 0, "", "", m.err
}

func (m *mockSourcehubKeeper) SendICASetAndDeleteRelationship(
	_ sdk.Context,
	newDid string,
	prevDid string,
	group string,
	requestor string,
) (uint64, string, string, error) {
	m.setAndDelCalls++
	m.lastDid = newDid
	m.lastPrev = prevDid
	m.lastGroup = group
	m.lastReq = requestor
	return 0, "", "", m.err
}

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	keeper        keeper.Keeper
	mockAdmin     *mockAdminKeeper
	mockSourcehub *mockSourcehubKeeper
	codec         codec.Codec
}

func (s *KeeperTestSuite) cdc() codec.Codec { return s.codec }

func (s *KeeperTestSuite) SetupTest() {
	s.mockAdmin = &mockAdminKeeper{admins: map[string]bool{}}
	s.mockSourcehub = &mockSourcehubKeeper{}

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := storetypes2.NewCommitMultiStore(db, cosmoslog.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	storeService := runtime.NewKVStoreService(storeKey)

	s.codec = cdc
	s.keeper = keeper.NewKeeper(cdc, storeService, s.mockAdmin, s.mockSourcehub)
	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, cosmoslog.NewNopLogger())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// addr returns a bech32 address derived from 20 deterministic bytes.
func addr(b byte) string {
	bz := make([]byte, 20)
	for i := range bz {
		bz[i] = b
	}
	return sdk.AccAddress(bz).String()
}

// claimAndConfirm is a helper for tests that want a fully-registered indexer.
// In the new ack-confirmed flow, that just means calling ApplyRegistration
// directly (as the ack callback would on SUCCESS).
func (s *KeeperTestSuite) claimAndConfirm(op, did, conn string) {
	s.Require().NoError(s.keeper.ApplyRegistration(s.ctx, op, did, conn))
}

// nodeIdentityKey returns a fresh secp256k1 keypair and a DER signature over
// sha256(message) — the shape Keeper.RegisterIndexer expects.
func nodeIdentityKey(t *testing.T, message []byte) (pubkey, signature []byte) {
	priv, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	h := sha256.Sum256(message)
	return priv.PubKey().SerializeCompressed(), ecdsa.Sign(priv, h[:]).Serialize()
}

func validatorA() []byte { return []byte("validator-A") }
func validatorB() []byte { return []byte("validator-B") }

func baseAssertion(op, pay string) *types.MsgIndexerAssertion {
	return &types.MsgIndexerAssertion{
		Signer:             addr(0xAA),
		SourceChain:        "ethereum",
		SourceChainId:      1,
		ValidatorPubkey:    validatorA(),
		AssertionAuthority: []byte("withdrawal-W"),
		Nonce:              1,
		ChainSpecific:      []byte("audit-bytes"),
		OperatorAddress:    op,
		PayoutAddress:      pay,
	}
}

func (s *KeeperTestSuite) upsertRegisteredIndexer(
	op string,
	sourceChainID uint64,
	sourceChain string,
	validatorPubkey []byte,
	did string,
	connectionString string,
) {
	assertion := baseAssertion(op, op)
	assertion.SourceChainId = sourceChainID
	assertion.SourceChain = sourceChain
	assertion.ValidatorPubkey = validatorPubkey
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, assertion))
	s.claimAndConfirm(op, did, connectionString)
}

// ─── tests ────────────────────────────────────────────────────────────

func (s *KeeperTestSuite) TestUpsertAssertion_Fresh() {
	op := addr(0x01)
	pay := addr(0x02)

	err := s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay))
	s.Require().NoError(err)

	row, found, err := s.keeper.GetIndexerByValidator(s.ctx, 1, validatorA())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(op, row.OperatorAddress)
	s.Require().Equal(pay, row.PayoutAddress)
	s.Require().Equal(uint64(1), row.Nonce)
	s.Require().False(row.Registered)
	s.Require().Empty(row.Did)
	s.Require().Empty(row.ConnectionString)
	s.Require().Equal([]byte("audit-bytes"), row.ChainSpecific)

	byAddr, found, err := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(row, byAddr)

	s.Require().Equal(uint64(1), s.keeper.GetIndexerCount(s.ctx))
}

func (s *KeeperTestSuite) TestUpsertAssertion_NonceMustBeStrictlyGreater() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	stale := baseAssertion(op, pay)
	stale.Nonce = 1 // same nonce
	err := s.keeper.UpsertAssertion(s.ctx, stale)
	s.Require().ErrorContains(err, "nonce 1 not strictly greater")
}

func (s *KeeperTestSuite) TestUpsertAssertion_Rotation_ResetsOperatorSide() {
	op1 := addr(0x01)
	op2 := addr(0x02)
	pay := addr(0x03)

	// Initial assert + complete registration.
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op1, pay)))
	s.claimAndConfirm(op1, "did:op1", "https://op1/9090")

	// Rotate: same validator, new operator, higher nonce.
	rotate := baseAssertion(op2, pay)
	rotate.Nonce = 2
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, rotate))

	// Only one row, operator-side fields reset.
	row, found, err := s.keeper.GetIndexerByValidator(s.ctx, 1, validatorA())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(op2, row.OperatorAddress)
	s.Require().False(row.Registered)
	s.Require().Empty(row.Did)
	s.Require().Empty(row.ConnectionString)
	s.Require().Equal(uint64(2), row.Nonce)

	// addr_idx swapped.
	_, found, err = s.keeper.GetIndexerByAddress(s.ctx, op1)
	s.Require().NoError(err)
	s.Require().False(found)
	_, found, err = s.keeper.GetIndexerByAddress(s.ctx, op2)
	s.Require().NoError(err)
	s.Require().True(found)

	// Still one row total.
	s.Require().Equal(uint64(1), s.keeper.GetIndexerCount(s.ctx))
}

func (s *KeeperTestSuite) TestUpsertAssertion_SameOperator_PreservesRegistration() {
	op := addr(0x01)
	pay := addr(0x02)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))
	s.claimAndConfirm(op, "did:op", "https://op/9090")

	// Re-assert with same operator, higher nonce, fresh proof.
	refresh := baseAssertion(op, pay)
	refresh.Nonce = 2
	refresh.ChainSpecific = []byte("fresh-proof")
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, refresh))

	row, found, err := s.keeper.GetIndexerByValidator(s.ctx, 1, validatorA())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().True(row.Registered)
	s.Require().Equal("did:op", row.Did)
	s.Require().Equal("https://op/9090", row.ConnectionString)
	s.Require().Equal([]byte("fresh-proof"), row.ChainSpecific)
	s.Require().Equal(uint64(2), row.Nonce)
}

func (s *KeeperTestSuite) TestUpsertAssertion_OperatorCollisionAcrossValidators() {
	op := addr(0x01)
	pay := addr(0x02)

	// Assert validator A with operator op.
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	// Try to assert validator B with the same operator address.
	collide := baseAssertion(op, pay)
	collide.ValidatorPubkey = validatorB()
	err := s.keeper.UpsertAssertion(s.ctx, collide)
	s.Require().ErrorContains(err, "already in use by another validator")

	// Validator A's row is untouched.
	rowA, found, err := s.keeper.GetIndexerByValidator(s.ctx, 1, validatorA())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(op, rowA.OperatorAddress)

	// Validator B's row never came into existence.
	_, found, err = s.keeper.GetIndexerByValidator(s.ctx, 1, validatorB())
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestSetPayout_OperatorUntouched() {
	op := addr(0x01)
	pay := addr(0x02)
	pay2 := addr(0x03)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))
	s.claimAndConfirm(op, "did:op", "https://op/9090")

	err := s.keeper.SetPayout(s.ctx, &types.MsgSetPayout{
		Signer:          addr(0xAA),
		SourceChainId:   1,
		ValidatorPubkey: validatorA(),
		PayoutAddress:   pay2,
		Nonce:           2,
	})
	s.Require().NoError(err)

	row, _, _ := s.keeper.GetIndexerByValidator(s.ctx, 1, validatorA())
	s.Require().Equal(pay2, row.PayoutAddress)
	s.Require().Equal(op, row.OperatorAddress)
	s.Require().True(row.Registered)
	s.Require().Equal("did:op", row.Did)
	s.Require().Equal("https://op/9090", row.ConnectionString)
	s.Require().Equal(uint64(2), row.Nonce)
}

func (s *KeeperTestSuite) TestRevokeIndexer_DropsRowAndIndex() {
	op := addr(0x01)
	pay := addr(0x02)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))
	s.Require().Equal(uint64(1), s.keeper.GetIndexerCount(s.ctx))

	err := s.keeper.RevokeIndexer(s.ctx, &types.MsgRevokeIndexer{
		Signer:          addr(0xAA),
		SourceChainId:   1,
		ValidatorPubkey: validatorA(),
		Nonce:           2,
	})
	s.Require().NoError(err)

	_, found, err := s.keeper.GetIndexerByValidator(s.ctx, 1, validatorA())
	s.Require().NoError(err)
	s.Require().False(found)

	_, found, err = s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().NoError(err)
	s.Require().False(found)

	s.Require().Equal(uint64(0), s.keeper.GetIndexerCount(s.ctx))
}

func (s *KeeperTestSuite) TestQueryServer_IndexersFilters() {
	qs := keeper.NewQueryServerImpl(s.keeper)
	op0 := addr(0x10)
	op1 := addr(0x11)
	op2 := addr(0x12)
	op3 := addr(0x13)
	s.upsertRegisteredIndexer(op0, 1, "ethereum", []byte("validator-filter-0"), "did:key:z0", "10.0.0.1:8080")
	s.upsertRegisteredIndexer(op1, 1, "ethereum", []byte("validator-filter-1"), "did:key:z1", "10.0.0.2:8080")
	s.upsertRegisteredIndexer(op2, 501, "solana", []byte("validator-filter-2"), "did:key:z2", "10.0.0.3:8080")
	s.upsertRegisteredIndexer(op3, 1, "ethereum", []byte("validator-filter-3"), "did:key:z3", "wss://example.com/indexer")

	resp, err := qs.Indexers(s.ctx, &types.QueryIndexersRequest{Did: "did:key:z1"})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 1)
	s.Require().Equal(op1, resp.Indexers[0].OperatorAddress)

	resp, err = qs.Indexers(s.ctx, &types.QueryIndexersRequest{
		SourceChainId:    1,
		ConnectionString: "10.0.0.",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 2)
	s.Require().Equal(op0, resp.Indexers[0].OperatorAddress)
	s.Require().Equal(op1, resp.Indexers[1].OperatorAddress)

	resp, err = qs.Indexers(s.ctx, &types.QueryIndexersRequest{
		Did:              "did:key:z1",
		ConnectionString: "example.com",
	})
	s.Require().NoError(err)
	s.Require().Empty(resp.Indexers)

	resp, err = qs.Indexers(s.ctx, &types.QueryIndexersRequest{
		SourceChainId: 1,
		Did:           "did:key:z2",
	})
	s.Require().NoError(err)
	s.Require().Empty(resp.Indexers)
}

func (s *KeeperTestSuite) TestQueryServer_IndexersFilterBeforePagination() {
	qs := keeper.NewQueryServerImpl(s.keeper)
	opA := addr(0x20)
	opB := addr(0x21)
	opC := addr(0x22)
	s.upsertRegisteredIndexer(opA, 1, "ethereum", []byte("validator-page-a"), "did:key:za", "alpha")
	s.upsertRegisteredIndexer(opB, 1, "ethereum", []byte("validator-page-b"), "did:key:zb", "needle-1")
	s.upsertRegisteredIndexer(opC, 1, "ethereum", []byte("validator-page-c"), "did:key:zc", "needle-2")

	resp, err := qs.Indexers(s.ctx, &types.QueryIndexersRequest{
		Pagination:       &query.PageRequest{Limit: 1},
		ConnectionString: "needle",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 1)
	s.Require().Equal(opB, resp.Indexers[0].OperatorAddress)
	s.Require().NotEmpty(resp.Pagination.NextKey)

	resp, err = qs.Indexers(s.ctx, &types.QueryIndexersRequest{
		Pagination:       &query.PageRequest{Key: resp.Pagination.NextKey, Limit: 1},
		ConnectionString: "needle",
	})
	s.Require().NoError(err)
	s.Require().Len(resp.Indexers, 1)
	s.Require().Equal(opC, resp.Indexers[0].OperatorAddress)
	s.Require().Empty(resp.Pagination.NextKey)
}

func (s *KeeperTestSuite) TestQueryServer_NilRequests() {
	qs := keeper.NewQueryServerImpl(s.keeper)

	_, err := qs.Indexers(s.ctx, nil)
	s.Require().Error(err)

	_, err = qs.Indexer(s.ctx, nil)
	s.Require().Error(err)

	_, err = qs.IndexerCount(s.ctx, nil)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRegisterIndexer_FirstTime_FiresICA_NoRowMutation() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	msg := []byte("op-claim-1")
	pub, sig := nodeIdentityKey(s.T(), msg)

	result, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/9090")
	s.Require().NoError(err)
	s.Require().True(result.Pending)
	s.Require().NotEmpty(result.Did)
	s.Require().Equal("ethereum", result.SourceChain)
	s.Require().Equal(uint64(1), result.SourceChainID)

	s.Require().Equal(1, s.mockSourcehub.calls())
	s.Require().Equal(result.Did, s.mockSourcehub.lastDid)
	s.Require().Equal("indexer", s.mockSourcehub.lastGroup)
	s.Require().Equal(op, s.mockSourcehub.lastReq)

	// Pending claim recorded so the ack callback can find the operator +
	// connection string when the SetRelationship ack lands.
	claim, found, err := s.keeper.GetPendingClaim(s.ctx, result.Did)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(op, claim.OperatorAddress)
	s.Require().Equal("https://op/9090", claim.ConnectionString)

	// Row is NOT mutated yet — operator-side fields stay empty until ack.
	row, _, _ := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().False(row.Registered)
	s.Require().Empty(row.Did)
	s.Require().Empty(row.ConnectionString)
}

func (s *KeeperTestSuite) TestRegisterIndexer_AppliedOnAck_FlipsRow() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	msg := []byte("op-claim")
	pub, sig := nodeIdentityKey(s.T(), msg)
	result, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/9090")
	s.Require().NoError(err)

	// Simulate ack landing successfully — the keeper's ack-side helper writes
	// the target state into the row.
	s.Require().NoError(s.keeper.ApplyRegistration(s.ctx, op, result.Did, "https://op/9090"))

	row, _, _ := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().True(row.Registered)
	s.Require().Equal(result.Did, row.Did)
	s.Require().Equal("https://op/9090", row.ConnectionString)
}

func (s *KeeperTestSuite) TestRegisterIndexer_NewDIDRecordsFreshPendingClaim() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	// Confirmed registration with key A.
	msgA := []byte("op-key-A")
	pubA, sigA := nodeIdentityKey(s.T(), msgA)
	resultA, err := s.keeper.RegisterIndexer(s.ctx, op, pubA, sigA, msgA, "https://op/A")
	s.Require().NoError(err)
	s.Require().NoError(s.keeper.ApplyRegistration(s.ctx, op, resultA.Did, "https://op/A"))
	s.keeper.DeletePendingClaim(s.ctx, resultA.Did)

	// New registration with key B — fires a fresh SetRelationship ICA and
	// records a new pending-claim entry keyed by the new DID.
	msgB := []byte("op-key-B")
	pubB, sigB := nodeIdentityKey(s.T(), msgB)
	priorSetCalls := s.mockSourcehub.setCalls
	resultB, err := s.keeper.RegisterIndexer(s.ctx, op, pubB, sigB, msgB, "https://op/B")
	s.Require().NoError(err)
	s.Require().True(resultB.Pending)
	s.Require().NotEqual(resultA.Did, resultB.Did)
	s.Require().Equal(resultB.Did, s.mockSourcehub.lastDid)

	// Rotation goes through the atomic Set+Delete path; the single-Set path
	// is not used again.
	s.Require().Equal(priorSetCalls, s.mockSourcehub.setCalls, "single-Set ICA should not fire on rotation")
	s.Require().Equal(1, s.mockSourcehub.setAndDelCalls, "Set+Delete ICA should fire on rotation")
	s.Require().Equal(resultA.Did, s.mockSourcehub.lastPrev, "rotation must forward the prev DID to sourcehub for the atomic Delete")

	claim, found, err := s.keeper.GetPendingClaim(s.ctx, resultB.Did)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(op, claim.OperatorAddress)
	s.Require().Equal("https://op/B", claim.ConnectionString)

	// Row still reflects the previously-confirmed state — no speculative write.
	row, _, _ := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().True(row.Registered)
	s.Require().Equal(resultA.Did, row.Did)
	s.Require().Equal("https://op/A", row.ConnectionString)
}

func (s *KeeperTestSuite) TestRegisterIndexer_IdempotentNoICA() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	msg := []byte("op-claim")
	pub, sig := nodeIdentityKey(s.T(), msg)
	first, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/9090")
	s.Require().NoError(err)
	s.Require().NoError(s.keeper.ApplyRegistration(s.ctx, op, first.Did, "https://op/9090"))
	s.mockSourcehub.setCalls = 0
	s.mockSourcehub.setAndDelCalls = 0

	// Same node identity key + same connection string while row is Registered:
	// keeper returns Pending=false and skips the ICA.
	second, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/9090")
	s.Require().NoError(err)
	s.Require().False(second.Pending)
	s.Require().Equal(first.Did, second.Did)
	s.Require().Equal(0, s.mockSourcehub.calls())
}

func (s *KeeperTestSuite) TestRegisterIndexer_SameDID_ConnStringChange_UpdatesInPlaceNoICA() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	// Confirmed registration with a node identity key.
	msg := []byte("op-claim")
	pub, sig := nodeIdentityKey(s.T(), msg)
	first, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/9090")
	s.Require().NoError(err)
	s.Require().NoError(s.keeper.ApplyRegistration(s.ctx, op, first.Did, "https://op/9090"))
	s.keeper.DeletePendingClaim(s.ctx, first.Did)
	s.mockSourcehub.setCalls = 0
	s.mockSourcehub.setAndDelCalls = 0

	// Same node identity key (same DID), NEW connection string. The group→DID
	// relationship is unchanged, so this must update the row in place with no ICA
	// round-trip — and must NOT hit the Set+Delete path (which rejects
	// prevDid == newDid and previously forced a DID rotation just to move endpoints).
	second, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/7070")
	s.Require().NoError(err)
	s.Require().False(second.Pending)
	s.Require().Equal(first.Did, second.Did)
	s.Require().Equal(0, s.mockSourcehub.calls(), "connection-string-only change must not fire any ICA")

	// Row reflects the new connection string immediately, DID unchanged.
	row, _, _ := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().True(row.Registered)
	s.Require().Equal(first.Did, row.Did)
	s.Require().Equal("https://op/7070", row.ConnectionString)

	// No lingering pending claim from the in-place update.
	_, found, err := s.keeper.GetPendingClaim(s.ctx, first.Did)
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestRegisterIndexer_UnknownOperatorErrors() {
	msg := []byte("op-claim")
	pub, sig := nodeIdentityKey(s.T(), msg)
	_, err := s.keeper.RegisterIndexer(s.ctx, addr(0x99), pub, sig, msg, "https://x")
	s.Require().ErrorContains(err, "not asserted")
}

func (s *KeeperTestSuite) TestRegisterIndexer_BadSignatureErrors() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	msg := []byte("op-claim")
	pub, _ := nodeIdentityKey(s.T(), msg)
	_, otherSig := nodeIdentityKey(s.T(), msg) // signature from a DIFFERENT key
	_, err := s.keeper.RegisterIndexer(s.ctx, op, pub, otherSig, msg, "https://x")
	s.Require().Error(err)
	s.Require().Equal(0, s.mockSourcehub.calls())
}

func (s *KeeperTestSuite) TestIterateIndexers_FiltersBySourceChain() {
	// Seed three rows across two chains.
	mkAssert := func(op string, chainID uint64, chain string, pubkey []byte) *types.MsgIndexerAssertion {
		return &types.MsgIndexerAssertion{
			Signer:             addr(0xAA),
			SourceChain:        chain,
			SourceChainId:      chainID,
			ValidatorPubkey:    pubkey,
			AssertionAuthority: []byte("auth"),
			Nonce:              1,
			OperatorAddress:    op,
			PayoutAddress:      op,
		}
	}
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, mkAssert(addr(0x01), 1, "ethereum", validatorA())))
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, mkAssert(addr(0x02), 1, "ethereum", validatorB())))
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, mkAssert(addr(0x03), 501, "solana", []byte("validator-S"))))

	all, _, err := s.keeper.IterateIndexers(s.ctx, 0, nil)
	s.Require().NoError(err)
	s.Require().Len(all, 3, "no filter returns every row")

	ethOnly, _, err := s.keeper.IterateIndexers(s.ctx, 1, nil)
	s.Require().NoError(err)
	s.Require().Len(ethOnly, 2)
	for _, ix := range ethOnly {
		s.Require().Equal(uint64(1), ix.SourceChainId)
		s.Require().Equal("ethereum", ix.SourceChain)
	}

	solOnly, _, err := s.keeper.IterateIndexers(s.ctx, 501, nil)
	s.Require().NoError(err)
	s.Require().Len(solOnly, 1)
	s.Require().Equal(uint64(501), solOnly[0].SourceChainId)

	none, _, err := s.keeper.IterateIndexers(s.ctx, 999, nil)
	s.Require().NoError(err)
	s.Require().Empty(none, "unknown chain returns empty list")
}

func (s *KeeperTestSuite) TestSetPayout_NonceMustAdvance() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	err := s.keeper.SetPayout(s.ctx, &types.MsgSetPayout{
		Signer:          addr(0xAA),
		SourceChainId:   1,
		ValidatorPubkey: validatorA(),
		PayoutAddress:   addr(0x03),
		Nonce:           1, // same as initial assertion
	})
	s.Require().ErrorContains(err, "not strictly greater")
}
