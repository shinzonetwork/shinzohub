// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The ViewRegistry contract's address.
address constant VIEW_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000210;

/// @dev The ViewRegistry contract's instance.
ViewRegistryI constant VIEW_REGISTRY_CONTRACT = ViewRegistryI(VIEW_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title ViewRegistry Precompiled Contract
/// @notice The interface through which solidity contracts can register and retrive views.
/// @custom:address 0x0000000000000000000000000000000000000210
interface ViewRegistryI {
    /// @notice Registers a value in the ViewRegistry.
    /// @dev The key is derived as keccak256(msg.sender, value).
    /// @param value The blob to store.
    function register(bytes memory value) external;

    /// @notice Retrieves a stored value using its key.
    /// @param key The key used to store the value (typically keccak256(sender, value)).
    /// @return result The stored blob.
    function get(bytes32 key) external view returns (bytes memory result);
}
