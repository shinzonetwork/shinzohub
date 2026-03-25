// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "./IViewPricing.sol";

/// @author Shinzo Team
/// @title IView
interface IView {
    function name() external view returns (string memory result);

    function creator() external view returns (address result);

    /// A host reports its pricing parameters for this view.
    /// @dev Only registered hosts.
    /// @param complexity The host's complexity score (1-100).
    /// @param rate The host's rate per query served.
    function report(uint256 complexity, uint256 rate) external;

    /// Returns every host that has reported pricing for this view.
    function hosts() external view returns (address[] memory result);

    /// A host removes itself from this view.
    /// @dev Only hosts that have previously reported.
    function unhost() external;

    event HostReported(
        address indexed host,
        uint256 complexity,
        uint256 rate
    );

    event Unhosted(address indexed host);

    /// Stake SHNZ on this view. Send SHNZ with the call.
    function stake() external payable;

    /// Unstake SHNZ from this view. SHNZ is returned to msg.sender.
    function unstake(uint256 amount) external;

    /// Returns the total SHNZ staked on this view.
    function totalStake() external view returns (uint256 result);

    /// Returns the staked amount for a given address.
    function stakeOf(address staker) external view returns (uint256 result);

    event Staked(address indexed staker, uint256 amount);
    event Unstaked(address indexed staker, uint256 amount);

    /// Returns the average rate across all hosts.
    function rate() external view returns (uint256 result);

    /// Returns the average complexity across all hosts.
    function complexity() external view returns (uint256 result);

    /// Fund a DID's access to this view. Send SHNZ with the call.
    /// @param did The DID to fund.
    function fund(bytes calldata did) external payable;

    /// Fund a DID's access on behalf of a delegate.
    /// @param delegate The address that will control the funds.
    /// @param did The DID to fund.
    function fundFor(address delegate, bytes calldata did) external payable;

    /// Returns the total funds deposited for a DID across all payers.
    function fundOf(bytes calldata did) external view returns (uint256 result);

    /// Returns how much `funder` has deposited for a given DID.
    function fundBy(address funder, bytes calldata did) external view returns (uint256 result);

    /// Withdraw (defund) unused prepaid balance for a DID back to msg.sender.
    /// @param did The DID to defund.
    /// @param amount The SHNZ amount to withdraw.
    function defund(bytes calldata did, uint256 amount) external;

    event Funded(address indexed beneficiary, address indexed payer, uint256 amount, bytes did);
    event Defunded(address indexed controller, uint256 amount, bytes did);

    /// Returns the creator's accumulated earnings.
    function earnings() external view returns (uint256 result);

    /// Creator claims their accumulated earnings.
    /// Only callable by the creator.
    function claimEarnings() external;

    event EarningsClaimed(address indexed creator, uint256 amount);

    /// A registered host consumes access for a DID, debiting the current price from the DID's balance.
    /// Only registered hosts.
    /// @param did The DID being served.
    function consume(bytes calldata did) external;

    event Consumed(address indexed host, uint256 amount, bytes did);

    /// Returns the price for accessing this view (after Shinzo cut).
    /// Shinzo cut is always applied regardless of custom pricing.
    function price() external view returns (uint256 result);

    /// Returns the custom pricing contract, or address(0) if using default.
    function pricingContract() external view returns (IViewPricing result);
}
