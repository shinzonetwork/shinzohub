package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shinzonetwork/shinzohub/x/host/keeper"
	"github.com/shinzonetwork/shinzohub/x/host/types"
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (s *KeeperTestSuite) TestRegisterHost_AckFailure_DropsPending() {
	message := []byte("host-ack-failure")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d}
	_, err := s.keeper.RegisterHost(s.ctx, nodePub, nodeSig, message, "1.2.3.4:80", callerAddr)
	s.Require().NoError(err)

	did, found := s.keeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	s.Require().True(found)

	meta := &sourcehubtypes.SetRelationshipMeta{Did: string(did), Group: "host"}
	metaBz, _ := s.cdc.Marshal(meta)
	cb := keeper.NewAckCallback(s.keeper)
	s.Require().NoError(cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE,
		Error:  "policy rejected",
	}))

	s.Require().False(s.keeper.IsRegisteredHost(s.ctx, callerAddr))
	_, stillPending := s.keeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	s.Require().False(stillPending)

	bech32 := sdk.AccAddress(callerAddr).String()
	_, canonicalFound, _ := s.keeper.GetHost(s.ctx, bech32)
	s.Require().False(canonicalFound)

	s.Require().True(hasEvent(s.ctx.EventManager().Events(), types.EventTypeHostRegistrationFailed))
}

func (s *KeeperTestSuite) TestAckCallback_IgnoresIndexerGroup() {
	message := []byte("host-ignores-indexer")
	nodePub, nodeSig := generateNodeIdentityKey(s.T(), message)
	callerAddr := []byte{0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d}
	_, err := s.keeper.RegisterHost(s.ctx, nodePub, nodeSig, message, "1.2.3.4:80", callerAddr)
	s.Require().NoError(err)

	meta := &sourcehubtypes.SetRelationshipMeta{Did: "did:some:other", Group: "indexer"}
	metaBz, _ := s.cdc.Marshal(meta)
	cb := keeper.NewAckCallback(s.keeper)
	s.Require().NoError(cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	}))

	_, stillPending := s.keeper.GetDIDForPendingAddress(s.ctx, callerAddr)
	s.Require().True(stillPending)
}

func hasEvent(events sdk.Events, eventType string) bool {
	for _, e := range events {
		if e.Type == eventType {
			return true
		}
	}
	return false
}
