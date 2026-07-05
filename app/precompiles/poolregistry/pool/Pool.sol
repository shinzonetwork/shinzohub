// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "../PoolRegistry.sol";

/// @author Shinzo Team
/// @title Pool
/// @notice Per-instance contract spawned by PoolRegistry at the deterministic
///         address derived from (viewAddress, config). One Pool contract per
///         (view, config) tuple.
/// @dev    This contract is intentionally thin. It does not hold operational
///         state (hosts, indexers, asks, demands) — that lives in the Cosmos
///         x/pool module behind the PoolRegistry precompile. The contract
///         exists so that each pool has its own EVM-addressable identity,
///         making it easy for off-chain indexers, block explorers, and other
///         contracts to address a specific pool.
contract Pool {
    /// @notice The view this pool serves.
    address public immutable viewAddress;

    /// @notice Address of the PoolRegistry precompile this contract was spawned by.
    address public immutable registry;

    /// @param _viewAddress The view this pool serves.
    constructor(address _viewAddress) {
        viewAddress = _viewAddress;
        registry = msg.sender;
    }

    /// @notice Return the full live snapshot of this pool — metadata plus every
    ///         host and demand currently in it.
    /// @dev    Delegates to PoolRegistry.getPoolDetail to read the latest state
    ///         from the Cosmos keeper. Use `snapshot().hosts` or
    ///         `snapshot().demands` to pull individual slices.
    function snapshot() external view returns (PoolRegistryI.PoolDetail memory) {
        return PoolRegistryI(registry).getPoolDetail(address(this));
    }

    /// @notice Join this pool as a host.
    /// @dev    Forwards msg.sender to the registry, which records the new host
    ///         entry against this pool's address.
    ///
    ///         The registry is the PoolRegistry precompile. Precompiles report
    ///         zero EXTCODESIZE, so a high-level `PoolRegistryI(registry).joinPool`
    ///         call would revert on Solidity's contract-existence check before
    ///         ever reaching the precompile. We use a low-level call, which skips
    ///         that check, and forward any revert reason from the precompile.
    ///         `registry` is set once at construction to the deployer (the
    ///         precompile), so it is not attacker-controllable.
    function join() external {
        _callRegistry(abi.encodeWithSelector(PoolRegistryI.joinPool.selector, msg.sender));
    }

    /// @notice Exit this pool as a host.
    /// @dev    Forwards msg.sender to the registry, which removes the host
    ///         entry. Emits PoolDeactivated if this exit drops the pool
    ///         below the activation threshold. Uses a low-level call for the
    ///         same reason as {join}.
    function exit() external {
        _callRegistry(abi.encodeWithSelector(PoolRegistryI.leavePool.selector, msg.sender));
    }

    /// @dev Low-level call into the registry precompile, bubbling up its revert
    ///      reason on failure. Used instead of a typed interface call because
    ///      precompiles report zero EXTCODESIZE and would trip Solidity's
    ///      contract-existence guard.
    function _callRegistry(bytes memory callData) private {
        (bool success, bytes memory ret) = registry.call(callData);
        if (!success) {
            assembly {
                revert(add(ret, 0x20), mload(ret))
            }
        }
    }
}
