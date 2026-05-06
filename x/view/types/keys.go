package types

const (
	ModuleName = "view"
	StoreKey   = ModuleName

	// ViewPrefix is the KVStore prefix for view records.
	// Key format: view/<contract_address> → View proto bytes
	ViewPrefix = "view/"

	// ViewCountKey stores the total number of registered views.
	ViewCountKey = "view_count"

	PendingViewPrefix = "pending_view/"
)

const (
	EventTypeViewPending              = "view.view_pending"
	EventTypeViewRegistered           = "view.view_registered"
	EventTypeViewRegistrationFailed   = "view.view_registration_failed"
	EventTypeViewRegistrationTimedOut = "view.view_registration_timed_out"

	AttrKeyViewID          = "view_id"
	AttrKeyContractAddress = "contract_address"
	AttrKeyCreator         = "creator"
	AttrKeyError           = "error"
)
