package types

const (
	ModuleName = "view"
	StoreKey   = ModuleName

	// view/<address> → View
	ViewPrefix   = "view/"
	ViewCountKey = "view_count"
)

const (
	EventTypeViewRegistered = "view.view_registered"

	AttrKeyAddress = "address"
	AttrKeyCreator = "creator"
	AttrKeyName    = "name"
	AttrKeyData    = "data"
)
