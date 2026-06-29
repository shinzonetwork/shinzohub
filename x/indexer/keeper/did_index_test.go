package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

func (s *KeeperTestSuite) TestGetIndexerByDID_FoundAfterRegistration() {
	op := addr(0x01)
	pay := addr(0x02)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))
	s.claimAndConfirm(op, "did:key:indexer-1", "https://op/9090")

	row, found, err := s.keeper.GetIndexerByDID(s.ctx, "did:key:indexer-1")
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(op, row.OperatorAddress)
	s.Require().Equal(pay, row.PayoutAddress)
	s.Require().Equal("did:key:indexer-1", row.Did)
	s.Require().True(row.Registered)
}

func (s *KeeperTestSuite) TestGetIndexerByDID_FalseForUnknown() {
	_, found, err := s.keeper.GetIndexerByDID(s.ctx, "did:key:nonexistent")
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestGetIndexerByDID_FalseBeforeRegistrationApplies() {
	op := addr(0x01)
	pay := addr(0x02)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	_, found, err := s.keeper.GetIndexerByDID(s.ctx, "did:key:not-yet")
	s.Require().NoError(err)
	s.Require().False(found, "DID index should not contain entries until ApplyRegistration runs")
}

func (s *KeeperTestSuite) TestGetAddressForDID_ReturnsPayoutAddress() {
	op := addr(0x01)
	pay := addr(0x02)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))
	s.claimAndConfirm(op, "did:key:indexer-2", "https://op/9090")

	got, found := s.keeper.GetAddressForDID(s.ctx, "did:key:indexer-2")
	s.Require().True(found)
	expected, _ := sdk.AccAddressFromBech32(pay)
	s.Require().Equal(expected, got, "GetAddressForDID must return the payout address, not the operator")
}

func (s *KeeperTestSuite) TestGetAddressForDID_FalseForUnknown() {
	_, found := s.keeper.GetAddressForDID(s.ctx, "did:key:nonexistent")
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestRevokeIndexer_ClearsDIDIndex() {
	op := addr(0x01)
	pay := addr(0x02)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))
	s.claimAndConfirm(op, "did:key:to-be-revoked", "https://op/9090")

	_, found, _ := s.keeper.GetIndexerByDID(s.ctx, "did:key:to-be-revoked")
	s.Require().True(found, "precondition: DID index populated")

	revoke := &types.MsgRevokeIndexer{
		Signer:          addr(0xAA),
		SourceChainId:   1,
		ValidatorPubkey: validatorA(),
		Nonce:           2,
	}
	s.Require().NoError(s.keeper.RevokeIndexer(s.ctx, revoke))

	_, found, err := s.keeper.GetIndexerByDID(s.ctx, "did:key:to-be-revoked")
	s.Require().NoError(err)
	s.Require().False(found, "DID index must be cleared on revoke")
}

func (s *KeeperTestSuite) TestApplyRegistration_NewDIDReplacesOldInIndex() {
	op := addr(0x01)
	pay := addr(0x02)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))
	s.claimAndConfirm(op, "did:key:old", "https://op/9090")

	_, oldFound, _ := s.keeper.GetIndexerByDID(s.ctx, "did:key:old")
	s.Require().True(oldFound)

	s.claimAndConfirm(op, "did:key:new", "https://op/9091")

	_, oldStill, _ := s.keeper.GetIndexerByDID(s.ctx, "did:key:old")
	s.Require().False(oldStill, "old DID must be cleared from the index when a new DID is applied")

	row, found, err := s.keeper.GetIndexerByDID(s.ctx, "did:key:new")
	s.Require().NoError(err)
	s.Require().True(found, "new DID must be in the index")
	s.Require().Equal("did:key:new", row.Did)
	s.Require().Equal("https://op/9091", row.ConnectionString)
}

func (s *KeeperTestSuite) TestApplyRegistration_SameDIDIsIdempotent() {
	op := addr(0x01)
	pay := addr(0x02)

	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))
	s.claimAndConfirm(op, "did:key:same", "https://op/9090")
	s.claimAndConfirm(op, "did:key:same", "https://op/9090")

	row, found, err := s.keeper.GetIndexerByDID(s.ctx, "did:key:same")
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(op, row.OperatorAddress)
}

func (s *KeeperTestSuite) TestInitGenesis_PopulatesDIDIndex() {
	pay1 := addr(0x10)
	pay2 := addr(0x20)
	pay3 := addr(0x30)

	gs := types.GenesisState{
		Indexers: []types.Indexer{
			{
				SourceChain:     "ethereum",
				SourceChainId:   1,
				ValidatorPubkey: []byte("v1"),
				OperatorAddress: addr(0x11),
				PayoutAddress:   pay1,
				Did:             "did:key:g1",
				Registered:      true,
			},
			{
				SourceChain:     "ethereum",
				SourceChainId:   1,
				ValidatorPubkey: []byte("v2"),
				OperatorAddress: addr(0x21),
				PayoutAddress:   pay2,
				Did:             "did:key:g2",
				Registered:      true,
			},
			{
				SourceChain:     "ethereum",
				SourceChainId:   1,
				ValidatorPubkey: []byte("v3"),
				OperatorAddress: addr(0x31),
				PayoutAddress:   pay3,
				Did:             "",
			},
		},
	}

	s.keeper.InitGenesis(s.ctx, gs)

	row1, found, _ := s.keeper.GetIndexerByDID(s.ctx, "did:key:g1")
	s.Require().True(found)
	s.Require().Equal(pay1, row1.PayoutAddress)

	row2, found, _ := s.keeper.GetIndexerByDID(s.ctx, "did:key:g2")
	s.Require().True(found)
	s.Require().Equal(pay2, row2.PayoutAddress)

	_, found, _ = s.keeper.GetIndexerByDID(s.ctx, "")
	s.Require().False(found, "InitGenesis must skip rows whose DID is empty")
}
