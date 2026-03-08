// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The HostRegistry precompile address.
address constant HOST_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000211;

/// @dev Pre-instantiated HostRegistry precompile contract.
HostRegistryI constant HOST_REGISTRY_CONTRACT = HostRegistryI(HOST_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title HostRegistry Precompile
interface HostRegistryI {

    /// @param peerKeyPubkey            Peer key public key bytes.
    /// @param peerKeySignature         Signature by peerKeyPubkey.
    /// @param nodeIdentityKeyPubkey    Node identity key public key bytes.
    /// @param nodeIdentityKeySignature Signature by nodeIdentityKeyPubkey.
    /// @param message                  Payload.
    function register(
        bytes calldata peerKeyPubkey,
        bytes calldata peerKeySignature,
        bytes calldata nodeIdentityKeyPubkey,
        bytes calldata nodeIdentityKeySignature,
        bytes calldata message
    ) external;

    /// @param addr The address to check.
    /// @return result True if the address is a registered host.
    function isRegistered(address addr) external view returns (bool result);

    /// @param addr The address to look up.
    /// @return did The DID bytes (empty if not registered).
    function getDid(address addr) external view returns (bytes memory did);

    /// @param addr The address to look up.
    /// @return pid The PID bytes (empty if not registered).
    function getPid(address addr) external view returns (bytes memory pid);

    /// @param owner Address that registered.
    /// @param did   The DID bytes.
    /// @param pid   The Peer ID bytes.
    event Registered(
        address indexed owner,
        bytes did,
        bytes pid
    );
}
