// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The ViewRegistry precompile address.
address constant VIEW_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000210;

/// @dev Pre-instantiated ViewRegistry precompile contract.
ViewRegistryI constant VIEW_REGISTRY_CONTRACT = ViewRegistryI(VIEW_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title ViewRegistry Precompile
/// @notice Registers Shinzo views and tracks them by a deterministic 20-byte
///         "view address" derived from `(caller, bundle)`. No contract bytecode
///         is ever deployed at the view address — it is a pure identifier the
///         registry owns. All view metadata lives in the Cosmos x/view module
///         and is exposed back to the EVM through this precompile.
/// @dev    Registration is asynchronous: `register()` stages the view as
///         PENDING and submits an ACP "register object" request to sourcehub
///         over IBC ICA. Once the ack returns SUCCESS the view becomes
///         REGISTERED and is visible to `listViews` / `viewCount`. Failure or
///         timeout drops the pending entry. EVM callers poll `getView()` to
///         observe the transition; the precompile never blocks on the ack.
/// @custom:address 0x0000000000000000000000000000000000000210
interface ViewRegistryI {
    /// @notice Lifecycle status of a view, returned by `getView` and `listViews`.
    /// @dev    NONE indicates the address is not known to the registry; this
    ///         lets callers distinguish "doesn't exist" from "pending".
    enum Status {
        NONE,       // 0 — no view at this address
        PENDING,    // 1 — register() submitted, awaiting sourcehub ACP ack
        REGISTERED  // 2 — ack succeeded, view is finalised and queryable
    }

    /// @notice Snapshot of a registered or pending view.
    /// @param viewAddress Deterministic 20-byte view address.
    /// @param name        Resource name parsed from the SDL `type` declaration.
    /// @param creator     EVM hex address (mixed-case EIP-55) of the EOA that
    ///                    called `register()`.
    /// @param height      Cosmos block height at which `register()` ran.
    /// @param status      Current lifecycle status — see {Status}.
    struct View {
        address viewAddress;
        string name;
        string creator;
        uint64 height;
        uint8 status;
    }

    /// @notice Register a new view from a viewbundle.
    /// @dev    Computes `viewAddress = keccak256("shinzo.view.v1" || msg.sender || data)[12:]`,
    ///         parses the SDL `type` declaration for `name`, persists the
    ///         bundle in the Cosmos store, and fires an ACP register-object
    ///         request to sourcehub over ICA. Reverts if the bundle is
    ///         malformed, if the SDL has no `type` declaration, or if the ICA
    ///         channel is not ready. On success the view is PENDING until the
    ///         ack lands.
    /// @param data The opaque viewbundle bytes (see `viewbundle-go`).
    /// @return viewAddress Deterministic 20-byte address now associated with
    ///         this view. Use it for {getView} lookups.
    /// @return name        Resource name extracted from the SDL.
    function register(bytes calldata data) external returns (address viewAddress, string memory name);

    /// @notice Fetch a single view by its address.
    /// @dev    Checks the finalised store first, then the pending store.
    ///         Returns an empty View with `status == NONE` if neither has it,
    ///         so callers can branch on status without separate exists checks.
    /// @param viewAddress The view address returned by `register()`.
    /// @return The view snapshot, including its lifecycle `status`.
    function getView(address viewAddress) external view returns (View memory);

    /// @notice Paginate registered views. Pending views are NOT included —
    ///         use {getView} on a specific address to observe pending state.
    /// @param offset Number of entries to skip from the start of the registry.
    /// @param limit  Maximum number of entries to return in this page.
    /// @return Array of registered view snapshots. May be shorter than `limit`
    ///         when fewer registered views remain.
    function listViews(uint256 offset, uint256 limit) external view returns (View[] memory);

    /// @notice Total number of finalised (REGISTERED) views in the registry.
    /// @dev    Pending views do not count.
    /// @return The cumulative count of registered views since genesis.
    function viewCount() external view returns (uint256);

    /// @notice Emitted by `register()` as soon as the view is staged as PENDING.
    /// @dev    Indicates the register call succeeded and an ACP request was
    ///         submitted. Observers should still poll `getView()` to detect
    ///         the eventual transition to REGISTERED, since the ack-side
    ///         transition does not produce an EVM log.
    /// @param viewAddress The deterministic view address.
    /// @param creator     The EVM caller that registered the view.
    /// @param name        The SDL resource name.
    event ViewCreated(address indexed viewAddress, address indexed creator, string name);

    // config for a pool, for now i only know of windowSize for now, mabe more in the future
    struct PoolConfig {
        uint64 windowSize;
    }

    // struct to represent pool
    struct Pool {
        address poolAddress,
        address viewAddress,
        PoolConfig config,
        bool isActive
    }

    // we want to create a pool demand contract here, this is what creates a pool
    function registerDemandForView(address viewAddress, PoolConfig config) payable external returns (Pool pool);

    // event for pool creation, anytime an event is created we get this event
    event PoolCreated(
        address poolAddress,
        address viewAddress,
        PoolConfig config
    )
