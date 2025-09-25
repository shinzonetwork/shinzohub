package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Make sure your generated MsgRegisterSourcehubICA implements sdk.Msg
var _ sdk.Msg = &MsgRegisterSourcehubICA{}

// Route returns the module name
func (m *MsgRegisterSourcehubICA) Route() string { return RouterKey }

// Type returns the action
func (m *MsgRegisterSourcehubICA) Type() string { return "RegisterSourcehubICA" }

// GetSigners defines whose signature is required
func (m *MsgRegisterSourcehubICA) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err) // should never happen because ValidateBasic catches it
	}
	return []sdk.AccAddress{addr}
}

// ValidateBasic runs stateless checks
func (m *MsgRegisterSourcehubICA) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}

	if m.ControllerConnectionId == "" {
		return fmt.Errorf("controller connection ID cannot be empty")
	}

	if m.HostConnectionId == "" {
		return fmt.Errorf("host connection ID cannot be empty")
	}

	// TODO: validate admin

	return nil
}
