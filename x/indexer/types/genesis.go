package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Indexers:     []Indexer{},
		Assertions: []IndexerAssertion{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
