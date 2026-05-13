package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	calls int
	err   error
}

func (m *mockSourcehubKeeper) SendICASetRelationship(_ sdk.Context, _, _, _ string) (uint64, string, string, error) {
	m.calls++
	return 0, "", "", m.err
}

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	keeper        keeper.Keeper
	mockAdmin     *mockAdminKeeper
	mockSourcehub *mockSourcehubKeeper
}

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
	_, firstTime, err := s.keeper.CompleteRegistration(s.ctx, op1, "did:op1", "https://op1/9090")
	s.Require().NoError(err)
	s.Require().True(firstTime)

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
	_, _, err := s.keeper.CompleteRegistration(s.ctx, op, "did:op", "https://op/9090")
	s.Require().NoError(err)

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
	_, _, err := s.keeper.CompleteRegistration(s.ctx, op, "did:op", "https://op/9090")
	s.Require().NoError(err)

	err = s.keeper.SetPayout(s.ctx, &types.MsgSetPayout{
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

func (s *KeeperTestSuite) TestCompleteRegistration_FiresOnce() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	_, firstTime, err := s.keeper.CompleteRegistration(s.ctx, op, "did:op", "https://op/9090")
	s.Require().NoError(err)
	s.Require().True(firstTime)

	// Idempotent re-register: row stays registered, firstTime is false.
	_, firstTime, err = s.keeper.CompleteRegistration(s.ctx, op, "did:op", "https://op/new")
	s.Require().NoError(err)
	s.Require().False(firstTime)

	row, _, _ := s.keeper.GetIndexerByValidator(s.ctx, 1, validatorA())
	s.Require().Equal("https://op/new", row.ConnectionString)
}

func (s *KeeperTestSuite) TestCompleteRegistration_UnknownOperatorErrors() {
	_, _, err := s.keeper.CompleteRegistration(s.ctx, addr(0x99), "did:any", "https://x")
	s.Require().ErrorContains(err, "not asserted")
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
