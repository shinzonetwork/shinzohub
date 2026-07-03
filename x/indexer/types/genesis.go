package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Indexers: []Indexer{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
