package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Balances: []QueryBalance{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
