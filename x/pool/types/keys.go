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

	// pool_stats/<pool_address> → PoolStats. Stats are written by the
	// settlement module's AccountSettle handler whenever it processes a
	// pools[] entry. Missing key → all-zero stats (genesis pool, no
	// settlement messages applied yet).
	PoolStatsPrefix = "pool_stats/"

	PoolCountKey = "pool_count"

	// GlobalPriceKey holds the network-wide price-per-unit-of-data shared
	// by every pool. Lazily initialised to DefaultStartingPrice on first
	// read if unset.
	GlobalPriceKey = "global_price"

	// MinHostsForActive is the minimum number of hosts a pool needs before
	// it's considered active and eligible to serve traffic.
	MinHostsForActive = 3

	// MinPoolsForMarketPrice is the minimum number of other active pools with
	// derivable prices required before a pool's price falls back to the
	// network-wide market median instead of the default starting price.
	MinPoolsForMarketPrice = 5

	// NZO uses 18 decimals, like ETH. Useful sub-unit reference points:
	//
	//   1 NZO         = 10^18 base units
	//   1 milli-NZO   = 10^15 base units  (0.001 NZO)
	//   1 micro-NZO   = 10^12 base units  (0.000001 NZO)
	//   1 nano-NZO    = 10^9  base units  (0.000000001 NZO)
	//
	// Pool prices are stored as base units (sdk.Int as base-10 string).

	// DefaultStartingPrice is the seed price-per-unit-of-data used when there
	// aren't enough active pools to derive a market average from.
	// 100 micro-NZO = 1e14 base units = 0.0001 NZO.
	DefaultStartingPrice = "100000000000000"
)

const (
	EventTypePoolCreated      = "pool.pool_created"
	EventTypePoolActivated    = "pool.pool_activated"
	EventTypePoolDeactivated  = "pool.pool_deactivated"
	EventTypeHostJoined       = "pool.host_joined"
	EventTypeHostLeft         = "pool.host_left"
	EventTypeDemandRegistered = "pool.demand_registered"
	EventTypePoolStatsUpdated = "pool.stats_updated"

	AttrKeyPoolAddress       = "pool_address"
	AttrKeyViewAddress       = "view_address"
	AttrKeyHostAddress       = "host_address"
	AttrKeyRegistrantAddress = "registrant_address"
	AttrKeyBond              = "bond"
	AttrKeyError             = "error"
	AttrKeyPrice             = "price"
	AttrKeyUtilization       = "utilization"
	AttrKeyTotalQueries      = "total_queries"
	AttrKeyTotalRewards      = "total_rewards"
	AttrKeyEpoch             = "epoch"
)
