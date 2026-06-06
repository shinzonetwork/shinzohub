// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The PoolRegistry precompile address.
address constant POOL_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000213;

/// @dev Pre-instantiated PoolRegistry precompile contract.
PoolRegistryI constant POOL_REGISTRY_CONTRACT = PoolRegistryI(POOL_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title PoolRegistry Precompile
/// @notice Creates and tracks pools. A pool is the (view, config) tuple that
///         binds supply (hosts, indexers) to a developer's demand. Pool state
///         lives in the Cosmos x/pool module; this precompile is the entry
///         point and the view → pools / pool → view index.
/// @custom:address 0x0000000000000000000000000000000000000213
interface PoolRegistryI {
    /// @notice Configuration that uniquely identifies a pool for a given view.
    /// @dev    Changing a field (e.g. adding a new one) yields a different pool
    ///         address. The whole struct is hashed into the CREATE2 salt.
    struct PoolConfig {
        uint64 windowSize;
    }

    /// @notice Snapshot of a pool.
    /// @param poolAddress  Deterministic pool address derived from (view, config).
    /// @param viewAddress  The view this pool serves.
    /// @param config       The PoolConfig that produced `poolAddress`.
    /// @param isActive     Whether the pool currently meets its liveness threshold.
    /// @param price        Current effective price per unit of data, in NZO wei.
    ///                     Sourced from this pool's host asks if available; falls
    ///                     back to the network market median, then to the default.
    struct Pool {
        address poolAddress;
        address viewAddress;
        PoolConfig config;
        bool isActive;
        uint256 price;
    }

    /// @notice One host's membership entry in a pool.
    struct PoolHostEntry {
        address hostAddress;
        uint256 ask;       // sdk.Int as uint256; 0 means no ask submitted yet
        int64 joinedAt;
    }

    /// @notice One demand registration on a pool.
    struct PoolDemandEntry {
        address registrant;
        uint256 bond;
        uint256 pricePref;
        bool binding;
        int64 expiresAt;
    }

    /// @notice Denormalised pool snapshot: metadata plus every host and demand
    ///         currently in the pool.
    struct PoolDetail {
        Pool pool;
        PoolHostEntry[] hosts;
        PoolDemandEntry[] demands;
    }

    /// @notice Register a demand for a view, materialising the pool if it
    ///         doesn't exist yet. The caller's `msg.value` is the demand bond.
    /// @dev    Computes the pool address from `(viewAddress, config)`. If no
    ///         pool exists at that address, persists one. Then records a
    ///         demand entry for `msg.sender` with `msg.value` as the bond.
    ///         Reverts if the view isn't registered or `msg.value == 0`.
    /// @param viewAddress The view to register demand against.
    /// @param config      The pool configuration.
    /// @return pool       The pool snapshot after the call.
    function registerDemandForView(address viewAddress, PoolConfig calldata config)
        external
        payable
        returns (Pool memory pool);

    /// @notice List every pool that has been created for a given view.
    /// @param viewAddress The view to list pools for.
    /// @return Array of pool addresses, possibly empty.
    function poolsOf(address viewAddress) external view returns (address[] memory);

    /// @notice Reverse lookup: which view does this pool serve.
    /// @param poolAddress A pool address previously returned by {registerDemandForView}.
    /// @return The view address this pool serves, or zero if unknown.
    function viewOfPool(address poolAddress) external view returns (address);

    /// @notice Fetch a single pool by its address.
    /// @param poolAddress The pool to look up.
    /// @return The pool snapshot. Returns a zero-filled Pool if unknown.
    function getPool(address poolAddress) external view returns (Pool memory);

    /// @notice Look up a pool by its (view, config) tuple.
    /// @dev    Useful for checking whether a pool already exists before calling
    ///         {registerDemandForView}. Internally derives the pool address from
    ///         (viewAddress, config) and looks it up. Returns a zero-filled Pool
    ///         (poolAddress == address(0)) if no pool has been materialised yet.
    /// @param viewAddress The view the pool would serve.
    /// @param config      The pool configuration.
    /// @return The pool snapshot, or a zero-filled Pool if it doesn't exist.
    function getPoolFor(address viewAddress, PoolConfig calldata config)
        external
        view
        returns (Pool memory);

    /// @notice Fetch a pool's full snapshot — metadata plus every host and
    ///         demand currently in the pool.
    /// @dev    Primary use case is Pool.sol's own `detail()` method delegating
    ///         to this. Callers with an instance prefer `pool.detail()`; this
    ///         entry point exists for off-chain callers that only have an address.
    /// @param poolAddress The pool to look up.
    /// @return The denormalised pool snapshot. Returns a zero-valued PoolDetail
    ///         (with empty arrays) if the pool doesn't exist.
    function getPoolDetail(address poolAddress) external view returns (PoolDetail memory);

    /// @notice Record a new host membership against the calling pool.
    /// @dev    The pool the host joins is `msg.sender`. The precompile rejects
    ///         calls whose caller isn't a registered pool, so this can only be
    ///         invoked indirectly through a deployed Pool.sol's `join()`.
    /// @param host The address being added as a host of the calling pool.
    function joinPool(address host) external;

    /// @notice Remove a host membership from the calling pool.
    /// @dev    Same authorisation model as {joinPool}.
    /// @param host The host being removed.
    function leavePool(address host) external;

    /// @notice Emitted by {registerDemandForView} when the pool is first created.
    /// @param poolAddress The newly materialised pool address.
    /// @param viewAddress The view this pool serves.
    /// @param config      The PoolConfig the pool was created with.
    event PoolCreated(address indexed poolAddress, address indexed viewAddress, PoolConfig config);

    /// @notice Emitted by {registerDemandForView} after the demand bond is recorded.
    /// @param poolAddress The pool the demand was registered against.
    /// @param registrant  The address that sent the bond (msg.sender).
    /// @param bond        The bond amount (msg.value).
    event DemandRegistered(address indexed poolAddress, address indexed registrant, uint256 bond);
}
