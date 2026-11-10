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

// TestAckCallback_RevokeWhileClaimInFlight_ClaimCleanedUp covers the race where
// an operator is revoked between firing the registration ICA and its ack landing.
// RevokeIndexer deletes the row + addr index and proactively drops the in-flight
// pending claim, so no orphan lingers in state. When the (now stale) ack arrives
// it finds no claim and no-ops — the indexer is NOT resurrected.
func (s *KeeperTestSuite) TestAckCallback_RevokeWhileClaimInFlight_ClaimCleanedUp() {
	op := addr(0x01)
	pay := addr(0x02)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, pay)))

	// A registration in flight, claim recorded and ack outstanding.
	// RegisterIndexer no longer writes this state, so stage it directly.
	const inFlightDID = "did:key:zInFlightIndexer"
	s.Require().NoError(s.keeper.SetPendingClaim(s.ctx, inFlightDID, indexertypes.PendingClaim{
		OperatorAddress:  op,
		ConnectionString: "https://op/9090",
	}))

	// An unrelated operator also has a pending claim in flight — it must survive
	// the revoke (the scan only drops claims for the revoked operator).
	const otherDID = "did:other-op"
	otherOp := addr(0x07)
	s.Require().NoError(s.keeper.SetPendingClaim(s.ctx, otherDID, indexertypes.PendingClaim{
		OperatorAddress:  otherOp,
		ConnectionString: "https://other/9090",
	}))

	// Admin revokes before the ack lands.
	s.Require().NoError(s.keeper.RevokeIndexer(s.ctx, &indexertypes.MsgRevokeIndexer{
		Signer:          addr(0xAA),
		SourceChainId:   1,
		ValidatorPubkey: validatorA(),
		Nonce:           2,
	}))
	s.Require().Equal(uint64(0), s.keeper.GetIndexerCount(s.ctx))

	// Revoke proactively drops the in-flight pending claim — no orphan in state.
	_, found, err := s.keeper.GetPendingClaim(s.ctx, inFlightDID)
	s.Require().NoError(err)
	s.Require().False(found, "revoke cleans up the in-flight pending claim")

	// The unrelated operator's claim is untouched.
	_, found, err = s.keeper.GetPendingClaim(s.ctx, otherDID)
	s.Require().NoError(err)
	s.Require().True(found, "revoke must not drop other operators' claims")

	// The stale SUCCESS ack arrives for the revoked operator.
	meta := &sourcehubtypes.SetRelationshipMeta{Did: inFlightDID, Group: "indexer"}
	metaBz, err := s.cdc().Marshal(meta)
	s.Require().NoError(err)

	cb := keeper.NewAckCallback(s.keeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)

	// No indexer resurrected — row and addr index stay gone, count unchanged.
	_, found, err = s.keeper.GetIndexerByValidator(s.ctx, 1, validatorA())
	s.Require().NoError(err)
	s.Require().False(found)
	_, found, err = s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().NoError(err)
	s.Require().False(found)
	s.Require().Equal(uint64(0), s.keeper.GetIndexerCount(s.ctx))

	// Claim is still gone (revoke already cleaned it up; the stale ack was a no-op).
	_, found, err = s.keeper.GetPendingClaim(s.ctx, inFlightDID)
	s.Require().NoError(err)
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

// ackSuccess delivers a SUCCESS ack for a set-relationship on did.
func (s *KeeperTestSuite) ackSuccess(did string) {
	meta := &sourcehubtypes.SetRelationshipMeta{Did: did, Group: "indexer"}
	metaBz, err := s.cdc().Marshal(meta)
	s.Require().NoError(err)
	err = keeper.NewAckCallback(s.keeper).OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)
}

// A claim left by the old ICA flow under an earlier DID must not survive a
// registration under a new DID. Its late SUCCESS ack would otherwise roll the
// row back to the earlier DID and drop the index of the current one.
func (s *KeeperTestSuite) TestAckCallback_StaleClaimUnderOldDID_DoesNotRollBack() {
	op := addr(0x01)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, addr(0x02))))

	// Registered on did A, then a re-registration on A was stranded mid ICA.
	const oldDID = "did:op-A"
	s.claimAndConfirm(op, oldDID, "https://op/A")
	s.Require().NoError(s.keeper.SetPendingClaim(s.ctx, oldDID, indexertypes.PendingClaim{
		OperatorAddress:  op,
		ConnectionString: "https://op/A-stale",
	}))

	// Another operator's claim must be left alone.
	const otherDID = "did:other-op"
	s.Require().NoError(s.keeper.SetPendingClaim(s.ctx, otherDID, indexertypes.PendingClaim{
		OperatorAddress:  addr(0x07),
		ConnectionString: "https://other/9090",
	}))

	// The operator rotates their node key and registers on a new DID.
	msg := []byte("op-claim")
	pub, sig := nodeIdentityKey(s.T(), msg)
	res, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/B")
	s.Require().NoError(err)
	newDID := res.Did
	s.Require().NotEqual(oldDID, newDID)

	_, found, err := s.keeper.GetPendingClaim(s.ctx, oldDID)
	s.Require().NoError(err)
	s.Require().False(found, "registration clears the operator's claim under the old DID")
	_, found, err = s.keeper.GetPendingClaim(s.ctx, otherDID)
	s.Require().NoError(err)
	s.Require().True(found, "other operators' claims survive")

	// The stranded ICA finally acks.
	s.ackSuccess(oldDID)

	row, _, err := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().NoError(err)
	s.Require().True(row.Registered)
	s.Require().Equal(newDID, row.Did, "row stays on the new DID")
	s.Require().Equal("https://op/B", row.ConnectionString)

	byDID, found, err := s.keeper.GetIndexerByDID(s.ctx, newDID)
	s.Require().NoError(err)
	s.Require().True(found, "the new DID index survives")
	s.Require().Equal(op, byDID.OperatorAddress)
	_, found, err = s.keeper.GetIndexerByDID(s.ctx, oldDID)
	s.Require().NoError(err)
	s.Require().False(found, "the old DID index stays gone")
}

// The unchanged path must clear stale claims too: re-registering the row's
// current values after the upgrade still disarms a stranded ICA.
func (s *KeeperTestSuite) TestAckCallback_StaleClaim_ClearedOnUnchangedRegister() {
	op := addr(0x01)
	s.Require().NoError(s.keeper.UpsertAssertion(s.ctx, baseAssertion(op, addr(0x02))))

	msg := []byte("op-claim")
	pub, sig := nodeIdentityKey(s.T(), msg)
	first, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/B")
	s.Require().NoError(err)

	// A claim under an earlier DID, as the old flow could have left behind.
	const oldDID = "did:op-A"
	s.Require().NoError(s.keeper.SetPendingClaim(s.ctx, oldDID, indexertypes.PendingClaim{
		OperatorAddress:  op,
		ConnectionString: "https://op/A-stale",
	}))

	again, err := s.keeper.RegisterIndexer(s.ctx, op, pub, sig, msg, "https://op/B")
	s.Require().NoError(err)
	s.Require().Equal(keeper.RegisterStatusUnchanged, again.Status)

	_, found, err := s.keeper.GetPendingClaim(s.ctx, oldDID)
	s.Require().NoError(err)
	s.Require().False(found)

	s.ackSuccess(oldDID)
	row, _, err := s.keeper.GetIndexerByAddress(s.ctx, op)
	s.Require().NoError(err)
	s.Require().Equal(first.Did, row.Did)
	s.Require().Equal("https://op/B", row.ConnectionString)
}
