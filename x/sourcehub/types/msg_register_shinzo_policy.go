package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Make sure your generated MsgRegisterShinzoPolicy implements sdk.Msg
var _ sdk.Msg = &MsgRegisterShinzoPolicy{}

// Route returns the module name
func (m *MsgRegisterShinzoPolicy) Route() string { return RouterKey }

// Type returns the action
func (m *MsgRegisterShinzoPolicy) Type() string { return "RegisterShinzoPolicy" }

// GetSigners defines whose signature is required
func (m *MsgRegisterShinzoPolicy) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err) // should never happen because ValidateBasic catches it
	}
	return []sdk.AccAddress{addr}
}

// ValidateBasic runs stateless checks
func (m *MsgRegisterShinzoPolicy) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}

	// TODO: validate admin

	return nil
}
