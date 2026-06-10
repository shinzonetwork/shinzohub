package types

import "errors"

var (
	ErrInvalidEndpointAddress        = errors.New("invalid endpointAddress")
	ErrAddressRegisteredDifferentDID = errors.New("address already registered as host with a different DID")
	ErrDIDRegisteredDifferentAddress = errors.New("DID already registered as host with a different address")
)
