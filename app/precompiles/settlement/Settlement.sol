// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.20;

/// @title Settlement precompile
/// @notice Lets operators read their pending SHINUSD settlement balance and
///         claim into their wallet. The precompile lives at a fixed address.
interface ISettlement {
    /// @notice Emitted when a claimer pulls amount from their pending balance.
    /// @param claimer the EVM address that initiated the claim
    /// @param amount the SHINUSD claimed (in ushinusd, 6 decimals)
    /// @param remaining the claimer's pending balance after this claim
    event Claimed(address indexed claimer, uint256 amount, uint256 remaining);

    /// @notice Claim SHINUSD from the caller's pending settlement balance.
    ///         The protocol mints SHINUSD and transfers it to the caller's
    ///         wallet, then decrements the pending balance by amount.
    /// @param amount how much ushinusd to claim (must be > 0 and <= balance)
    /// @return remaining the claimer's pending balance after this claim
    function claim(uint256 amount) external returns (uint256 remaining);

    /// @notice Read the pending settlement balance for an address.
    ///         This is NOT the holder's wallet SHINUSD balance — for that,
    ///         use the SHINUSD ERC-20 precompile.
    /// @param holder the address to inspect
    /// @return the pending ushinusd owed by the protocol
    function balanceOf(address holder) external view returns (uint256);
}
