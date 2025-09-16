package types

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrInvalidGenesis = sdkerrors.Register(ModuleName, 1, "invalid genesis")
)
