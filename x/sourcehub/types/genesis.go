package types

func DefaultGenesis() *GenesisState {
	// you can override in genesis.json
	return &GenesisState{
		ControllerConnectionId: "connection-0",
		HostConnectionId:       "connection-0",
		Version:                "ics27-1",
		Encoding:               "proto3",
		TxType:                 "sdk_multi_msg",
		PolicyId:               "",
	}
}

func (gs *GenesisState) Validate() error {
	return nil // TODO: validate genesis
}
