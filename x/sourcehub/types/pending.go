package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	PendingRequestPrefix   = "pending_ica/"
	PendingByRequestorPrefix = "pending_ica_by_requestor/"
)

func PendingRequestKey(portID, channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s%s/%s/%d", PendingRequestPrefix, portID, channelID, sequence))
}

func PendingByRequestorKey(requestor string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s%s/%020d", PendingByRequestorPrefix, requestor, sequence))
}

func PendingByRequestorPrefixKey(requestor string) []byte {
	return []byte(fmt.Sprintf("%s%s/", PendingByRequestorPrefix, requestor))
}

type PacketAckCallback interface {
	OnPacketAck(ctx sdk.Context, req PendingICARequest) error
}

const (
	EventTypeRequestPending      = "sourcehub.request_pending"
	EventTypeRequestAcknowledged = "sourcehub.request_acknowledged"
	EventTypeRequestFailed       = "sourcehub.request_failed"
	EventTypeRequestTimedOut     = "sourcehub.request_timed_out"

	EventTypeStreamAccessPending  = "sourcehub.stream_access_pending"
	EventTypeStreamAccessGranted  = "sourcehub.stream_access_granted"
	EventTypeStreamAccessDenied   = "sourcehub.stream_access_denied"
	EventTypeStreamAccessTimedOut = "sourcehub.stream_access_timed_out"

	EventTypeShinzoObjectsRegistered          = "sourcehub.shinzo_objects_registered"
	EventTypeShinzoObjectsRegistrationFailed  = "sourcehub.shinzo_objects_registration_failed"
	EventTypeShinzoObjectsRegistrationTimedOut = "sourcehub.shinzo_objects_registration_timed_out"

	EventTypePolicyCreated = "sourcehub.policy_created"
	AttrKeyPolicyID        = "policy_id"

	AttrKeySequence   = "sequence"
	AttrKeyPortID     = "port_id"
	AttrKeyChannelID  = "channel_id"
	AttrKeyRequestKind = "request_kind"
	AttrKeyRequestor  = "requestor"
	AttrKeyError      = "error"
	AttrKeyDid        = "did"
	AttrKeyStreamID   = "stream_id"
	AttrKeyResources  = "resources"
)
