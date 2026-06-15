// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

address constant QUERY_BALANCE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000214;

QueryBalanceI constant QUERY_BALANCE_CONTRACT = QueryBalanceI(QUERY_BALANCE_PRECOMPILE_ADDRESS);

interface QueryBalanceI {
    function fund(string calldata did) external payable;

    function balanceOf(string calldata did) external view returns (uint256);

    event Funded(string did, address indexed funder, uint256 amount);
}
