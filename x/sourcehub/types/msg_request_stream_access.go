package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Make sure your generated MsgRegisterSourcehubICA implements sdk.Msg
var _ sdk.Msg = &MsgRequestStreamAccess{}

// Route returns the module name
func (m *MsgRequestStreamAccess) Route() string { return RouterKey }

// Type returns the action
func (m *MsgRequestStreamAccess) Type() string { return "RequestStreamAccess" }

// GetSigners defines whose signature is required
func (m *MsgRequestStreamAccess) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err) // should never happen because ValidateBasic catches it
	}
	return []sdk.AccAddress{addr}
}

// ValidateBasic runs stateless checks
func (m *MsgRequestStreamAccess) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}

	if m.StreamId == "" {
		return fmt.Errorf("Stream ID cannot be empty")
	}

	if m.Did == "" {
		return fmt.Errorf("DID cannot be empty")
	}

	// TODO: validate admin

	return nil
}
