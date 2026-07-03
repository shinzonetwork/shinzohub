package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgIndexerAssertion{}
	_ sdk.Msg = &MsgSetPayout{}
	_ sdk.Msg = &MsgRevokeIndexer{}
)

func (m *MsgIndexerAssertion) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if m.SourceChain == "" {
		return fmt.Errorf("source_chain cannot be empty")
	}
	if m.SourceChainId == 0 {
		return fmt.Errorf("source_chain_id must be non-zero")
	}
	if len(m.ValidatorPubkey) == 0 {
		return fmt.Errorf("validator_pubkey cannot be empty")
	}
	if len(m.ValidatorPubkey) > MaxValidatorPubkeyLen {
		return fmt.Errorf("validator_pubkey too large: %d > %d", len(m.ValidatorPubkey), MaxValidatorPubkeyLen)
	}
	if len(m.ChainSpecific) > MaxChainSpecificLen {
		return fmt.Errorf("chain_specific too large: %d > %d", len(m.ChainSpecific), MaxChainSpecificLen)
	}
	if len(m.AssertionAuthority) == 0 {
		return fmt.Errorf("assertion_authority cannot be empty")
	}
	if m.Nonce == 0 {
		return fmt.Errorf("nonce must be non-zero")
	}
	if _, err := sdk.AccAddressFromBech32(m.OperatorAddress); err != nil {
		return fmt.Errorf("invalid operator_address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.PayoutAddress); err != nil {
		return fmt.Errorf("invalid payout_address: %w", err)
	}
	return nil
}

func (m *MsgSetPayout) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if m.SourceChainId == 0 {
		return fmt.Errorf("source_chain_id must be non-zero")
	}
	if len(m.ValidatorPubkey) == 0 {
		return fmt.Errorf("validator_pubkey cannot be empty")
	}
	if len(m.ValidatorPubkey) > MaxValidatorPubkeyLen {
		return fmt.Errorf("validator_pubkey too large: %d > %d", len(m.ValidatorPubkey), MaxValidatorPubkeyLen)
	}
	if _, err := sdk.AccAddressFromBech32(m.PayoutAddress); err != nil {
		return fmt.Errorf("invalid payout_address: %w", err)
	}
	if m.Nonce == 0 {
		return fmt.Errorf("nonce must be non-zero")
	}
	return nil
}

func (m *MsgRevokeIndexer) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if m.SourceChainId == 0 {
		return fmt.Errorf("source_chain_id must be non-zero")
	}
	if len(m.ValidatorPubkey) == 0 {
		return fmt.Errorf("validator_pubkey cannot be empty")
	}
	if len(m.ValidatorPubkey) > MaxValidatorPubkeyLen {
		return fmt.Errorf("validator_pubkey too large: %d > %d", len(m.ValidatorPubkey), MaxValidatorPubkeyLen)
	}
	if m.Nonce == 0 {
		return fmt.Errorf("nonce must be non-zero")
	}
	return nil
}
