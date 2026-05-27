package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgDeleteStreamAccess{}

// Route returns the module name.
func (m *MsgDeleteStreamAccess) Route() string { return RouterKey }

// Type returns the action label used by the SDK's legacy router.
func (m *MsgDeleteStreamAccess) Type() string { return "DeleteStreamAccess" }

// GetSigners returns the address authorized to submit the message.
func (m *MsgDeleteStreamAccess) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err) // should never happen because ValidateBasic catches it
	}
	return []sdk.AccAddress{addr}
}

// ValidateBasic runs stateless checks.
func (m *MsgDeleteStreamAccess) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}

	if m.StreamId == "" {
		return fmt.Errorf("stream id cannot be empty")
	}

	if m.Did == "" {
		return fmt.Errorf("did cannot be empty")
	}

	return nil
}
