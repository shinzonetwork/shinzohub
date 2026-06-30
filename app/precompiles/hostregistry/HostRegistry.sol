// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The HostRegistry precompile address.
address constant HOST_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000211;

/// @dev Pre-instantiated HostRegistry precompile contract.
HostRegistryI constant HOST_REGISTRY_CONTRACT = HostRegistryI(HOST_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title HostRegistry Precompile
interface HostRegistryI {

    /// @param nodeIdentityKeyPubkey    Node identity key public key bytes.
    /// @param nodeIdentityKeySignature Signature by nodeIdentityKeyPubkey.
    /// @param message                  Payload.
    /// @param connectionString         Connection string for the host.
    /// @param endpointAddress         	Address of GraphQL endpoint exposed by the host.
    function register(
        bytes calldata nodeIdentityKeyPubkey,
        bytes calldata nodeIdentityKeySignature,
        bytes calldata message,
        string calldata connectionString,
        string calldata endpointAddress
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

    /// @param addr The address to look up.
    /// @return endpointAddress Address of GraphQL endpoint exposed by host (empty if not registered).
    function getEndpointAddress(address addr) external view returns (string memory endpointAddress);

    /// @param owner            Address that registered.
    /// @param did              The DID bytes.
    /// @param connectionString The connection string.
    /// @param endpointAddress   Address of GraphQL endpoint exposed by host.
    event Registered(
        address indexed owner,
        bytes did,
        string connectionString,
        string endpointAddress
    );
}
