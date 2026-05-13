// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The IndexerRegistry precompile address.
address constant INDEXER_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000212;

/// @author Shinzo Team
/// @title IndexerRegistry Precompile
interface IndexerRegistryI {

    /// @param nodeIdentityKeyPubkey    Node identity key public key (used only to derive the DID).
    /// @param nodeIdentityKeySignature Signature by nodeIdentityKeyPubkey over `message`.
    /// @param message                  Bytes signed by the node identity key.
    /// @param connectionString         Connection string for the indexer.
    function register(
        bytes calldata nodeIdentityKeyPubkey,
        bytes calldata nodeIdentityKeySignature,
        bytes calldata message,
        string calldata connectionString
    ) external;

    /// @param addr The operator address to check.
    /// @return result True if the indexer has completed registration.
    function isRegistered(address addr) external view returns (bool result);

    /// @param addr The operator address to look up.
    /// @return did The DID bytes (empty if not yet registered).
    function getDid(address addr) external view returns (bytes memory did);

    /// @param addr The operator address to look up.
    /// @return connectionString The connection string (empty if not yet registered).
    function getConnectionString(address addr) external view returns (string memory connectionString);

    /// @param addr The operator address to look up.
    /// @return sourceChain The keccak256 hash of the source chain name (zero if not asserted).
    function getSourceChain(address addr) external view returns (bytes32 sourceChain);

    /// @param owner            Operator address that registered.
    /// @param did              The DID bytes.
    /// @param connectionString The connection string.
    /// @param sourceChain      The source chain name from the indexer's assertion.
    /// @param sourceChainId    The source chain id from the indexer's assertion.
    event Registered(
        address indexed owner,
        bytes did,
        string connectionString,
        string sourceChain,
        uint64 sourceChainId
    );
}
