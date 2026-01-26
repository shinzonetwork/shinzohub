// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The ViewRegistry contract's address.
address constant VIEW_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000210;

/// @dev The ViewRegistry contract's instance.
ViewRegistryI constant VIEW_REGISTRY_CONTRACT = ViewRegistryI(VIEW_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title ViewRegistry Precompiled Contract
/// @notice The interface through which solidity contracts can register and retrieve views.
/// @custom:address 0x0000000000000000000000000000000000000210
interface ViewRegistryI {
    /// @notice Registers a value in the ViewRegistry.
    /// @dev The key is derived as keccak256(msg.sender, value).
    /// @param value The blob to store.
    function register(bytes calldata value) external;

    /// @notice Retrieves a stored value using its key.
    /// @param key The key used to store the value (typically keccak256(sender, value)).
    /// @return result The stored blob.
    function get(bytes32 key) external view returns (bytes memory result);

    /// @notice Emitted when a value is registered.
    /// @param key The derived key of the stored value.
    /// @param sender The address that called register().
    /// @param value The raw bytes stored.
    event Registered(bytes32 indexed key, address indexed sender, bytes value);
}
