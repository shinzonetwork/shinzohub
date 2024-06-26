package test

import (
	"reflect"
	"time"

	gogotypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/assert"
)

// MustDateTimeToProto parses a time.DateTime (YYYY-MM-DD HH:MM:SS) timestamp
// and converts into a proto Timestamp.
// Panics if input is invalid
func MustDateTimeToProto(timestamp string) *gogotypes.Timestamp {
	t, err := time.Parse(time.DateTime, timestamp)
	if err != nil {
		panic(err)
	}

	ts, err := gogotypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}

	return ts
}

func TimeToProto(ts time.Time) *gogotypes.Timestamp {
	return &gogotypes.Timestamp{
		Seconds: ts.Unix(),
		Nanos:   0,
	}
}

func AssertResults(ctx *TestCtx, got, want any, gotErr, wantErr error) {
	if wantErr != nil {
		assert.ErrorIs(ctx.T, gotErr, wantErr)
	} else {
		assert.NoError(ctx.T, gotErr)
	}
	if !isNil(want) {
		assert.Equal(ctx.T, want, got)
	}
}

func isNil(object interface{}) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	isNilableKind := containsKind(
		[]reflect.Kind{
			reflect.Chan, reflect.Func,
			reflect.Interface, reflect.Map,
			reflect.Ptr, reflect.Slice, reflect.UnsafePointer},
		kind)

	if isNilableKind && value.IsNil() {
		return true
	}

	return false
}

// containsKind checks if a specified kind in the slice of kinds.
func containsKind(kinds []reflect.Kind, kind reflect.Kind) bool {
	for i := 0; i < len(kinds); i++ {
		if kind == kinds[i] {
			return true
		}
	}

	return false
}
