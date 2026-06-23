package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Pools:   []Pool{},
		Hosts:   []PoolHostEntry{},
		Demands: []PoolDemandEntry{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
