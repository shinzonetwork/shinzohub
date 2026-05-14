package keeper_test

import (
	"github.com/shinzonetwork/shinzohub/x/indexer/keeper"
	indexertypes "github.com/shinzonetwork/shinzohub/x/indexer/types"
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (s *KeeperTestSuite) TestAckCallback_SuccessAppliesPendingClaim() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	// Pre-stage the pending claim that RegisterIndexer would have written.
	const did = "did:op-A"
	claim := &indexertypes.PendingClaim{
		OperatorAddress:  op,
		ConnectionString: "https://op/9090",
	}
	s.Require().NoError(s.keeper.SetPendingClaim(s.ctx, did, *claim))

	meta := &sourcehubtypes.SetRelationshipMeta{Did: did, Group: "indexer"}
	metaBz, err := s.cdc().Marshal(meta)
	s.Require().NoError(err)

	cb := keeper.NewAckCallback(s.keeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)

	row, _, _ := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().True(row.Registered)
	s.Require().Equal(did, row.Did)
	s.Require().Equal("https://op/9090", row.ConnectionString)

	// Pending claim consumed.
	_, found, _ := s.keeper.GetPendingClaim(s.ctx, did)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestAckCallback_FailureDropsPendingClaim_RowUntouched() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	// Row currently has a confirmed registration on did A.
	s.claimAndConfirm(op, "did:op-A", "https://op/A")

	// Operator submits a re-registration on did B → pending claim stored.
	const didB = "did:op-B"
	s.Require().NoError(s.keeper.SetPendingClaim(s.ctx, didB, indexertypes.PendingClaim{
		OperatorAddress:  op,
		ConnectionString: "https://op/B",
	}))

	meta := &sourcehubtypes.SetRelationshipMeta{Did: didB, Group: "indexer"}
	metaBz, err := s.cdc().Marshal(meta)
	s.Require().NoError(err)

	cb := keeper.NewAckCallback(s.keeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE,
		Error:  "actor is not a manager",
	})
	s.Require().NoError(err)

	// Row stays at did A — nothing was speculatively written.
	row, _, _ := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().True(row.Registered)
	s.Require().Equal("did:op-A", row.Did)
	s.Require().Equal("https://op/A", row.ConnectionString)

	// Pending claim gone.
	_, found, _ := s.keeper.GetPendingClaim(s.ctx, didB)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestAckCallback_NonIndexerGroupIsIgnored() {
	meta := &sourcehubtypes.SetRelationshipMeta{Did: "did:host:something", Group: "host"}
	metaBz, err := s.cdc().Marshal(meta)
	s.Require().NoError(err)

	cb := keeper.NewAckCallback(s.keeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err) // silently ignored, no error
}
