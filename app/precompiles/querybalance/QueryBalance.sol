// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

address constant QUERY_BALANCE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000214;

QueryBalanceI constant QUERY_BALANCE_CONTRACT = QueryBalanceI(QUERY_BALANCE_PRECOMPILE_ADDRESS);

/// @title QueryBalance precompile
/// @notice Holds SHINUSD that the gateway debits on a per-query basis. Users
///         top up their own query balance (or someone else's) by funding the
///         module account with SHINUSD they already hold — claimed from
///         settlement, bridged, or transferred.
interface QueryBalanceI {
    /// @notice Move `amount` SHINUSD (ushinusd) from msg.sender's wallet into
    ///         msg.sender's own query balance. msg.sender must hold at least
    ///         `amount` ushinusd.
    function fund(uint256 amount) external;

    /// @notice Like fund, but credits `recipient`'s query balance instead of
    ///         msg.sender's. SHINUSD still comes from msg.sender's wallet.
    function fundFor(address recipient, uint256 amount) external;

    function balanceOf(address holder) external view returns (uint256);

    event Funded(address indexed funder, address indexed recipient, uint256 amount);
}
