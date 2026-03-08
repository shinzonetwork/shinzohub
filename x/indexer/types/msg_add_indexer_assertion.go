package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgIndexerAssertion{}

func (m *MsgIndexerAssertion) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
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

	if len(m.DelegateDigest) != 32 {
		return fmt.Errorf("delegate_digest must be exactly 32 bytes (got %d)", len(m.DelegateDigest))
	}

	if len(m.DelegateSignature) != 65 {
		return fmt.Errorf("delegate_signature must be exactly 65 bytes (got %d)", len(m.DelegateSignature))
	}

	return nil
}
