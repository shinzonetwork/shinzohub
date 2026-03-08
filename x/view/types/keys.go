package types

const (
	ModuleName = "view"
	StoreKey   = ModuleName

	// ViewPrefix is the KVStore prefix for view records.
	// Key format: view/<contract_address> → View proto bytes
	ViewPrefix = "view/"

	// ViewCountKey stores the total number of registered views.
	ViewCountKey = "view_count"
)
