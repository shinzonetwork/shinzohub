package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgAdminIndexerAssertion{}

func (m *MsgAdminIndexerAssertion) Route() string { return RouterKey }

func (m *MsgAdminIndexerAssertion) Type() string { return "AdminIndexerAssertion" }

func (m *MsgAdminIndexerAssertion) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgAdminIndexerAssertion) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}

	if m.ConsensusPubKey == "" {
		return fmt.Errorf("consensus pub key cannot be empty")
	}

	if m.DelegateAddress == "" {
		return fmt.Errorf("delegate address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.DelegateAddress); err != nil {
		return fmt.Errorf("invalid delegate address: %w", err)
	}

	if m.SourceChain == "" {
		return fmt.Errorf("source chain cannot be empty")
	}

	if m.SourceChainId == 0 {
		return fmt.Errorf("source chain id must be non-zero")
	}

	if m.AssertionId == "" {
		return fmt.Errorf("assertion id cannot be empty")
	}

	return nil
}
