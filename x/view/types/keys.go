package types

const (
	ModuleName = "view"
	StoreKey   = ModuleName

	// view/<address> → View
	ViewPrefix = "view/"
	// pending_view/<address> → View (awaiting sourcehub ack)
	PendingViewPrefix = "pending_view/"
	ViewCountKey      = "view_count"
)

const (
	EventTypeViewPending              = "view.view_pending"
	EventTypeViewRegistered           = "view.view_registered"
	EventTypeViewRegistrationFailed   = "view.view_registration_failed"
	EventTypeViewRegistrationTimedOut = "view.view_registration_timed_out"

	AttrKeyAddress = "address"
	AttrKeyCreator = "creator"
	AttrKeyName    = "name"
	AttrKeyData    = "data"
	AttrKeyError   = "error"
)
