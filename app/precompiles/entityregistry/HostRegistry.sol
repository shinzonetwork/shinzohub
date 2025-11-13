// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev EntityRegistry precompile deployed at a fixed address.
address constant ENTITY_REGISTRY_PRECOMPILE_ADDRESS =
    0x0000000000000000000000000000000000000211;

/// @dev Convenience instance of the EntityRegistry precompile.
EntityRegistryI constant ENTITY_REGISTRY_CONTRACT =
    EntityRegistryI(ENTITY_REGISTRY_PRECOMPILE_ADDRESS);

/// @title EntityRegistry Precompiled Contract
/// @notice Registers an entity DID/PID for the caller using dual key proofs.
/// @dev The underlying keeper is responsible for:
///      - Verifying `peerKeyPubkey` / `peerKeySignature`.
///      - Verifying `nodeIdentityKeyPubkey` / `nodeIdentityKeySignature`.
///      - Both signatures are over the same domain-separated payload:
///        (chain, precompile, msg.sender, entity, message, etc.).
///      - Enforcing one-address-one-DID and any entity-specific rules.
interface EntityRegistryI {
    /// @notice Register an entity for msg.sender using two key proofs.
    /// @dev
    ///  - `peerKeyPubkey` / `peerKeySignature`:
    ///        DefraDB peer-key.
    ///  - `nodeIdentityKeyPubkey` / `nodeIdentityKeySignature`:
    ///        DefraDB node-identity-key.
    ///  - `message`:
    ///        A 32-byte value used for domain separation and replay protection.
    ///        The keeper defines and validates the exact payload that is signed.
    /// @param peerKeyPubkey            Peer key public key bytes.
    /// @param peerKeySignature         Signature by `peerKeyPubkey` over the keeper-defined payload.
    /// @param nodeIdentityKeyPubkey    Node identity key public key bytes.
    /// @param nodeIdentityKeySignature Signature by `nodeIdentityKeyPubkey` over the keeper-defined payload.
    /// @param message                  message for replay protection / domain separation.
    /// @param entity                   Entity tag (e.g. 0 = indexer, 1 = host).
    function register(
        bytes calldata peerKeyPubkey,
        bytes calldata peerKeySignature,
        bytes calldata nodeIdentityKeyPubkey,
        bytes calldata nodeIdentityKeySignature,
        bytes calldata message,
        uint8 entity
    ) external;

    /// @notice Emitted when a DID/PID is registered for an owner.
    /// @dev
    ///  - `key` = keccak256(abi.encodePacked(owner, did))
    ///  - `owner` = msg.sender that performed the registration
    ///  - `did` / `pid` = values returned from the keeper after successful verification
    /// @param key     keccak256(abi.encodePacked(owner, did)).
    /// @param owner   Address that registered the entity.
    /// @param did     The DID bytes.
    /// @param pid     The Peer ID bytes.
    event EntityRegistered(
        bytes32 indexed key,
        address indexed owner,
        bytes did,
        bytes pid
    );
}
