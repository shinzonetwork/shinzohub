package viewbundle

import "errors"

var (
	ErrBadMagic   = errors.New("viewbundle: bad magic")
	ErrBadVersion = errors.New("viewbundle: unsupported version")
	ErrTooLarge   = errors.New("viewbundle: field too large")
	ErrCorrupt    = errors.New("viewbundle: corrupt encoding")
	ErrCodec      = errors.New("viewbundle: unknown codec")
	ErrMismatch   = errors.New("viewbundle: lens count mismatch")
)
