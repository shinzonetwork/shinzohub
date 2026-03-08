// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The ViewRegistry precompile address.
address constant VIEW_REGISTRY_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000210;

/// @dev Pre-instantiated ViewRegistry precompile contract.
ViewRegistryI constant VIEW_REGISTRY_CONTRACT = ViewRegistryI(VIEW_REGISTRY_PRECOMPILE_ADDRESS);

/// @author Shinzo Team
/// @title ViewRegistry Precompile
/// @notice Precompile that registers views and deploys a View contract per registration.
///         Each view is its own contract with injectable pricing logic.
/// @custom:address 0x0000000000000000000000000000000000000210
interface ViewRegistryI {

    /// @param data The raw view bundle data.
    /// @return viewAddress The address of the deployed View contract.
    function register(bytes calldata data)
        external
        returns (address viewAddress);

    /// @param data    The raw view bundle data.
    /// @param pricing The address of a custom IViewPricing contract.
    /// @return viewAddress The address of the deployed View contract.
    function registerWithPricing(
        bytes calldata data,
        address pricing
    ) external returns (address viewAddress);

    /// @param viewAddress The view contract address to look up.
    /// @return creator The bech32 creator address (as string).
    function getView(address viewAddress) external view returns (string memory creator);

    /// @param viewAddress The address of the deployed View contract.
    /// @param creator     The address that registered the view.
    /// @param name        The SDL resource name of the view.
    event ViewCreated(
        address indexed viewAddress,
        address indexed creator,
        string name
    );
}
