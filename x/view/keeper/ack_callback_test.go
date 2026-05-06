package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
	"github.com/shinzonetwork/shinzohub/x/view/keeper"
	"github.com/shinzonetwork/shinzohub/x/view/types"
)

func (s *KeeperTestSuite) TestRegisterView_AckFailure_DropsPending() {
	err := s.keeper.RegisterView(s.ctx, "V1", "View1", "creator", "0xv1", []byte("d"))
	s.Require().NoError(err)

	_, pendingFound, _ := s.keeper.GetPendingView(s.ctx, "V1")
	s.Require().True(pendingFound)

	meta := &sourcehubtypes.RegisterObjectMeta{ResourceName: sourcehubtypes.ViewResourceName, ObjectId: "V1"}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	cb := keeper.NewAckCallback(s.keeper)
	s.Require().NoError(cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_REGISTER_OBJECT,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE,
		Error:  "ACP: policy not found",
	}))

	_, pendingFound, _ = s.keeper.GetPendingView(s.ctx, "V1")
	s.Require().False(pendingFound)
	_, canonicalFound, _ := s.keeper.GetView(s.ctx, "0xv1")
	s.Require().False(canonicalFound)

	s.Require().True(hasEvent(s.ctx.EventManager().Events(), types.EventTypeViewRegistrationFailed))
}

func (s *KeeperTestSuite) TestRegisterView_AckTimeout_DropsPending() {
	err := s.keeper.RegisterView(s.ctx, "V2", "View2", "creator", "0xv2", []byte("d"))
	s.Require().NoError(err)

	meta := &sourcehubtypes.RegisterObjectMeta{ResourceName: sourcehubtypes.ViewResourceName, ObjectId: "V2"}
	metaBz, err := s.cdc.Marshal(meta)
	s.Require().NoError(err)
	cb := keeper.NewAckCallback(s.keeper)
	s.Require().NoError(cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_REGISTER_OBJECT,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT,
		Error:  "packet timed out",
	}))

	_, pendingFound, _ := s.keeper.GetPendingView(s.ctx, "V2")
	s.Require().False(pendingFound)
	_, canonicalFound, _ := s.keeper.GetView(s.ctx, "0xv2")
	s.Require().False(canonicalFound)

	s.Require().True(hasEvent(s.ctx.EventManager().Events(), types.EventTypeViewRegistrationTimedOut))
}

func (s *KeeperTestSuite) TestAckCallback_IgnoresWrongKind() {
	err := s.keeper.RegisterView(s.ctx, "V3", "View3", "creator", "0xv3", []byte("d"))
	s.Require().NoError(err)

	cb := keeper.NewAckCallback(s.keeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)

	_, pendingFound, _ := s.keeper.GetPendingView(s.ctx, "V3")
	s.Require().True(pendingFound)
}

func (s *KeeperTestSuite) TestAckCallback_IgnoresPrimitiveResource() {
	err := s.keeper.RegisterView(s.ctx, "V4", "View4", "creator", "0xv4", []byte("d"))
	s.Require().NoError(err)

	meta := &sourcehubtypes.RegisterObjectMeta{ResourceName: "primitive", ObjectId: "eth_transfers"}
	metaBz, _ := s.cdc.Marshal(meta)
	cb := keeper.NewAckCallback(s.keeper)
	err = cb.OnPacketAck(s.ctx, sourcehubtypes.PendingICARequest{
		Kind:   sourcehubtypes.RequestKind_REQUEST_KIND_REGISTER_OBJECT,
		Meta:   metaBz,
		Status: sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS,
	})
	s.Require().NoError(err)

	_, pendingFound, _ := s.keeper.GetPendingView(s.ctx, "V4")
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
