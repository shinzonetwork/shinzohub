package access_ticket

import (
	"errors"
	"fmt"
)

var (
	// ErrExternal represents an error in an external system,
	// which means the validation procedure can be retried
	ErrExternal error = errors.New("external error")

	ErrInvalidInput     = errors.New("invalid input")
	ErrDecisionNotFound = fmt.Errorf("decision not found: %w", ErrInvalidInput)
)
