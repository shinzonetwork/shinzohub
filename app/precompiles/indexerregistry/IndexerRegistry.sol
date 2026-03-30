// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The IndexerRegistry precompile address.
address constant INDEXER_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000212;

/// @author Shinzo Team
/// @title IndexerRegistry Precompile
interface IndexerRegistryI {

    /// @param connectionString  Connection string for the indexer.
    /// @param sourceChain       Source chain name.
    /// @param sourceChainId     Source chain id.
    function register(
        string calldata connectionString,
        string calldata sourceChain,
        uint64 sourceChainId
    ) external;

    /// @param addr The address to check.
    /// @return result True if the address is a registered indexer.
    function isRegistered(address addr) external view returns (bool result);

    /// @param addr The address to look up.
    /// @return did The DID bytes (empty if not registered).
    function getDid(address addr) external view returns (bytes memory did);

    /// @param addr The address to look up.
    /// @return connectionString The connection string (empty if not registered).
    function getConnectionString(address addr) external view returns (string memory connectionString);

    /// @param addr The address to look up.
    /// @return sourceChain The source chain name hash.
    function getSourceChain(address addr) external view returns (bytes32 sourceChain);

    /// @param owner            Address that registered.
    /// @param did              The DID bytes.
    /// @param connectionString The connection string.
    event Registered(
        address indexed owner,
        bytes did,
        string connectionString,
        string sourceChain,
        uint64 sourceChainId
    );
}
