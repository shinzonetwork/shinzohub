package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/indexer/keeper"
	"github.com/shinzonetwork/shinzohub/x/indexer/types"
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (s *KeeperTestSuite) TestRegisterIndexer_AckFailure_DropsPending() {
	message := []byte("ack-failure-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d}
	_, err := s.keeper.RegisterIndexer(s.ctx, nodePub, nodeSig, message, "1.2.3.4:80", callerAddr, "ethereum", 1)
	s.Require().NoError(err)

	did, found := s.keeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	s.Require().True(found)

	meta := &sourcehubtypes.SetRelationshipMeta{Did: string(did), Group: "indexer"}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	cb := keeper.NewAckCallback(s.keeper)
	s.Require().NoError(cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE,
		Error:  "ACP rejected",
	}))

	bech32 := sdk.AccAddress(callerAddr).String()
	_, canonicalFound, _ := s.keeper.GetIndexer(s.ctx, bech32)
	s.Require().False(canonicalFound)
	_, pendingFound, _ := s.keeper.GetPendingIndexer(s.ctx, bech32)
	s.Require().False(pendingFound)

	_, stillPending := s.keeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	s.Require().False(stillPending)

	s.Require().True(hasEvent(s.ctx.EventManager().Events(), types.EventTypeIndexerRegistrationFailed))
}

func (s *KeeperTestSuite) TestRegisterIndexer_AckTimeout_DropsPending() {
	message := []byte("ack-timeout-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d}
	_, err := s.keeper.RegisterIndexer(s.ctx, nodePub, nodeSig, message, "1.2.3.4:80", callerAddr, "ethereum", 1)
	s.Require().NoError(err)

	did, _ := s.keeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	meta := &sourcehubtypes.SetRelationshipMeta{Did: string(did), Group: "indexer"}
	metaBz, _ := s.cdc.Marshal(meta)
	cb := keeper.NewAckCallback(s.keeper)
	s.Require().NoError(cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT,
	}))

	bech32 := sdk.AccAddress(callerAddr).String()
	_, canonicalFound, _ := s.keeper.GetIndexer(s.ctx, bech32)
	s.Require().False(canonicalFound)
	_, pendingFound, _ := s.keeper.GetPendingIndexer(s.ctx, bech32)
	s.Require().False(pendingFound)

	s.Require().True(hasEvent(s.ctx.EventManager().Events(), types.EventTypeIndexerRegistrationTimedOut))
}

func (s *KeeperTestSuite) TestAckCallback_IgnoresHostGroup() {
	message := []byte("ignores-host-nonce")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d}
	_, err := s.keeper.RegisterIndexer(s.ctx, nodePub, nodeSig, message, "1.2.3.4:80", callerAddr, "ethereum", 1)
	s.Require().NoError(err)

	meta := &sourcehubtypes.SetRelationshipMeta{Did: "did:some:other", Group: "host"}
	metaBz, _ := s.cdc.Marshal(meta)
	cb := keeper.NewAckCallback(s.keeper)
	s.Require().NoError(cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	}))

	bech32 := sdk.AccAddress(callerAddr).String()
	_, pendingFound, _ := s.keeper.GetPendingIndexer(s.ctx, bech32)
	s.Require().True(pendingFound)
}

func hasEvent(events sdk.Events, eventType string) bool {
	for _, e := range events {
		if e.Type == eventType {
			return true
		}
	}
	return false
}
