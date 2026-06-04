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
}
