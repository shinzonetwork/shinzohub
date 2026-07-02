// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

address constant QUERY_BALANCE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000214;

QueryBalanceI constant QUERY_BALANCE_CONTRACT = QueryBalanceI(QUERY_BALANCE_PRECOMPILE_ADDRESS);

interface QueryBalanceI {
    function fund() external payable;

    function fundFor(address recipient) external payable;

    function balanceOf(address holder) external view returns (uint256);

    event Funded(address indexed funder, address indexed recipient, uint256 amount);
}
