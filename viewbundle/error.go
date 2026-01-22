package viewbundle

import (
	"errors"
	"fmt"
)

var (
	ErrBadMagic   = errors.New("viewbundle: bad magic")
	ErrBadVersion = errors.New("viewbundle: unsupported version")
	ErrTooLarge   = errors.New("viewbundle: field too large")
	ErrCorrupt    = errors.New("viewbundle: corrupt encoding")
	ErrCodec      = errors.New("viewbundle: unknown codec")
	ErrMismatch   = errors.New("viewbundle: lens count mismatch")
)

type TooLargeError struct {
	Field string
	Size  int
	Limit int
}

func (e *TooLargeError) Error() string {
	return fmt.Sprintf("%v: %s size=%d limit=%d", ErrTooLarge, e.Field, e.Size, e.Limit)
}

func (e *TooLargeError) Unwrap() error { return ErrTooLarge }

func tooLarge(field string, size, limit int) error {
	return &TooLargeError{Field: field, Size: size, Limit: limit}
}
