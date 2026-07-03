package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Balances: []SettlementBalance{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
