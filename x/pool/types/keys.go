package types

const (
	ModuleName = "pool"
	StoreKey   = ModuleName

	// pool/<pool_address> → Pool (metadata only)
	PoolPrefix = "pool/"

	// pool_by_view/<view_address>/<pool_address> → empty (presence-only secondary index)
	PoolByViewPrefix = "pool_by_view/"

	// pool_host/<pool_address>/<host_address> → PoolHost
	PoolHostPrefix = "pool_host/"

	// pool_demand/<pool_address>/<registrant_address> → PoolDemand
	PoolDemandPrefix = "pool_demand/"

	PoolCountKey = "pool_count"
)

const (
	EventTypePoolCreated      = "pool.pool_created"
	EventTypeHostJoined       = "pool.host_joined"
	EventTypeHostLeft         = "pool.host_left"
	EventTypeAskSubmitted     = "pool.ask_submitted"
	EventTypeDemandRegistered = "pool.demand_registered"

	AttrKeyPoolAddress       = "pool_address"
	AttrKeyViewAddress       = "view_address"
	AttrKeyHostAddress       = "host_address"
	AttrKeyRegistrantAddress = "registrant_address"
	AttrKeyAsk               = "ask"
	AttrKeyBond              = "bond"
	AttrKeyError             = "error"
)
