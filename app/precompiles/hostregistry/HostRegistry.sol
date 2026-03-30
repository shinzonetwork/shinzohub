// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The HostRegistry precompile address.
address constant HOST_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000211;

/// @dev Pre-instantiated HostRegistry precompile contract.
HostRegistryI constant HOST_REGISTRY_CONTRACT = HostRegistryI(HOST_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title HostRegistry Precompile
interface HostRegistryI {

    /// @param connectionString  Connection string for the host.
    function register(
        string calldata connectionString
    ) external;

    /// @param addr The address to check.
    /// @return result True if the address is a registered host.
    function isRegistered(address addr) external view returns (bool result);

    /// @param addr The address to look up.
    /// @return did The DID bytes (empty if not registered).
    function getDid(address addr) external view returns (bytes memory did);

    /// @param addr The address to look up.
    /// @return connectionString The connection string (empty if not registered).
    function getConnectionString(address addr) external view returns (string memory connectionString);

    /// @param owner            Address that registered.
    /// @param did              The DID bytes.
    /// @param connectionString The connection string.
    event Registered(
        address indexed owner,
        bytes did,
        string connectionString
    );
}
