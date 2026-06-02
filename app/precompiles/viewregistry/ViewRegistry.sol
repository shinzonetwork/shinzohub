// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The ViewRegistry precompile address.
address constant VIEW_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000210;

/// @dev Pre-instantiated ViewRegistry precompile contract.
ViewRegistryI constant VIEW_REGISTRY_CONTRACT = ViewRegistryI(VIEW_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title ViewRegistry Precompile
/// @notice Registers Shinzo views by a deterministic 20-byte address derived
///         from `(caller, bundle)`. No contract bytecode is ever deployed at
///         the view address — it is a pure identifier the registry owns. All
///         view metadata lives in the Cosmos x/view module and is exposed
///         back to the EVM through this precompile.
/// @custom:address 0x0000000000000000000000000000000000000210
interface ViewRegistryI {
    /// @notice Snapshot of a registered view.
    /// @param viewAddress Deterministic 20-byte view address.
    /// @param name        Resource name parsed from the SDL `type` declaration.
    /// @param creator     EVM hex address of the EOA that called `register()`.
    /// @param height      Cosmos block height at which `register()` ran.
    struct View {
        address viewAddress;
        string name;
        string creator;
        uint64 height;
    }

    /// @notice Register a new view from a viewbundle.
    /// @param data The opaque viewbundle bytes (see `viewbundle-go`).
    /// @return viewAddress Deterministic 20-byte address now associated with this view.
    /// @return name        Resource name extracted from the SDL.
    function register(bytes calldata data) external returns (address viewAddress, string memory name);

    /// @notice Fetch a single view by its address.
    function getView(address viewAddress) external view returns (View memory);

    /// @notice Paginate registered views.
    function listViews(uint256 offset, uint256 limit) external view returns (View[] memory);

    /// @notice Total number of registered views.
    function viewCount() external view returns (uint256);

    /// @notice Emitted when a view is registered.
    event ViewCreated(address indexed viewAddress, address indexed creator, string name);
}
