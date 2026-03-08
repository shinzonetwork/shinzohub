// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @author Shinzo Team
/// @title IViewPricing
/// @notice Interface for custom view pricing logic.
///         Implementations read view state and return a final price.
interface IViewPricing {

    /// @notice Returns the final price for accessing the view.
    /// @return result The calculated price.
    function price() external view returns (uint256 result);
}
