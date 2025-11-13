package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgRegisterShinzoObjects{}

func (m *MsgRegisterShinzoObjects) Route() string { return RouterKey }

func (m *MsgRegisterShinzoObjects) Type() string { return "RegisterShinzoObjects" }

func (m *MsgRegisterShinzoObjects) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err) // should never happen because ValidateBasic catches it
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgRegisterShinzoObjects) ValidateBasic() error {
	if len(m.Resources) == 0 {
		return fmt.Errorf("at least one resource is required")
	}

	seen := map[string]struct{}{}

	for i, r := range m.Resources {
		rr := strings.TrimSpace(strings.ToLower(r))
		if rr == "" {
			return fmt.Errorf("resource at index %d is empty", i)
		}
		if _, ok := seen[rr]; ok {
			return fmt.Errorf("duplicate resource: %s", rr)
		}
		m.Resources[i] = rr
		seen[rr] = struct{}{}
	}

	// TODO: validate admin

	return nil
}
