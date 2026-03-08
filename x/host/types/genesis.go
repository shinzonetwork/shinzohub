package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Hosts: []Host{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
