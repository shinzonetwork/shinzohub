package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Balances: []QueryBalance{},
	}
}

func (gs GenesisState) Validate() error {
	seen := make(map[string]struct{}, len(gs.Balances))
	for i, qb := range gs.Balances {
		if _, err := sdk.AccAddressFromBech32(qb.Address); err != nil {
			return fmt.Errorf("balance %d: invalid address %q: %w", i, qb.Address, err)
		}
		if _, ok := seen[qb.Address]; ok {
			return fmt.Errorf("balance %d: duplicate address %q", i, qb.Address)
		}
		seen[qb.Address] = struct{}{}

		amount, ok := math.NewIntFromString(qb.Amount)
		if !ok {
			return fmt.Errorf("balance %d (%s): invalid amount %q", i, qb.Address, qb.Amount)
		}
		if amount.IsNegative() {
			return fmt.Errorf("balance %d (%s): negative amount %s", i, qb.Address, amount)
		}
	}
	return nil
}
