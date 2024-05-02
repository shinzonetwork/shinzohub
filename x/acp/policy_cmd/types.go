package policy_cmd

import "context"

// LogicalClock models a Provider for logical timestamps.
// Timestamps must be monotomically increasing and are used as reference
// for the total ordering of events in the system.
//
// This abstraction is general purpose but for the current context of SourceHub
// this primarily means the current system block height.
type LogicalClock interface {

	// GetTimestamp returns an integer for the current timestamp in the system.
	GetTimestampNow(ctx context.Context) (uint64, error)
}
