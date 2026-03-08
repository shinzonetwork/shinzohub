package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Views: []View{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
