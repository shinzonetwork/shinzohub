// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "./IView.sol";
import "./IViewPricing.sol";
import "../../hostregistry/HostRegistry.sol";

/// @author Shinzo Team
/// @title View
/// Default implementation of IView.
contract View is IView {

    /// Shinzo protocol cut in basis points (500 = 5%).
    uint256 private constant SHINZO_CUT_BPS = 500;

    /// Basis points denominator.
    uint256 private constant BPS = 10_000;

    string private _name;
    address private immutable _creator;

    struct HostReport {
        uint256 complexity;
        uint256 rate;
        bool exists;
    }

    address[] private _hosts;
    mapping(address => HostReport) private _hostReports;

    uint256 private _totalStake;
    mapping(address => uint256) private _stakes;

    /// funder address → didHash → amount deposited
    mapping(address => mapping(bytes32 => uint256)) private _funds;
    /// didHash → total funds across all payers
    mapping(bytes32 => uint256) private _balances;

    uint256 private _earnings;

    /// Optional custom pricing contract. address(0) = use built-in default.
    IViewPricing private immutable _pricingContract;

    /// @param initName    The SDL resource name of this view.
    /// @param initCreator The address that registered the view.
    /// @param initPricing Custom pricing contract, or address(0) for default.
    constructor(
        string memory initName,
        address initCreator,
        address initPricing
    ) {
        _name = initName;
        _creator = initCreator;
        _pricingContract = IViewPricing(initPricing);
    }

    function name() external view override returns (string memory result) {
        return _name;
    }

    function creator() external view override returns (address result) {
        return _creator;
    }

    function report(uint256 _complexity, uint256 _rate) external override {
        require(
            HOST_REGISTRY_CONTRACT.isRegistered(msg.sender),
            "caller is not a registered host"
        );
        require(_complexity >= 1 && _complexity <= 100, "complexity must be 1-100");

        if (!_hostReports[msg.sender].exists) {
            _hosts.push(msg.sender);
            _hostReports[msg.sender].exists = true;
        }

        _hostReports[msg.sender].complexity = _complexity;
        _hostReports[msg.sender].rate = _rate;

        emit HostReported(msg.sender, _complexity, _rate);
    }

    function hosts() external view override returns (address[] memory result) {
        return _hosts;
    }

    function stake() external payable override {
        require(msg.value > 0, "must send SHNZ");
        _stakes[msg.sender] += msg.value;
        _totalStake += msg.value;

        emit Staked(msg.sender, msg.value);
    }

    function unstake(uint256 amount) external override {
        require(_stakes[msg.sender] >= amount, "insufficient stake");
        _stakes[msg.sender] -= amount;
        _totalStake -= amount;

        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "SHNZ transfer failed");

        emit Unstaked(msg.sender, amount);
    }

    function totalStake() external view override returns (uint256 result) {
        return _totalStake;
    }

    function stakeOf(address staker) external view override returns (uint256 result) {
        return _stakes[staker];
    }

    function rate() public view override returns (uint256 result) {
        uint256 len = _hosts.length;
        if (len == 0) return 0;

        uint256 total;
        for (uint256 i = 0; i < len; i++) {
            total += _hostReports[_hosts[i]].rate;
        }
        return total / len;
    }

    function complexity() public view override returns (uint256 result) {
        uint256 len = _hosts.length;
        if (len == 0) return 0;

        uint256 total;
        for (uint256 i = 0; i < len; i++) {
            total += _hostReports[_hosts[i]].complexity;
        }
        return total / len;
    }

    function fund(bytes calldata did) external payable override {
        _fund(msg.sender, msg.sender, did);
    }

    function fundFor(address delegate, bytes calldata did) external payable override {
        _fund(delegate, msg.sender, did);
    }

    function _fund(address beneficiary, address payer, bytes calldata did) internal {
        require(msg.value > 0, "must send SHNZ");

        bytes32 didHash = keccak256(did);
        _funds[beneficiary][didHash] += msg.value;
        _balances[didHash] += msg.value;

        emit Funded(beneficiary, payer, msg.value, did);
    }

    function fundOf(bytes calldata did) external view override returns (uint256 result) {
        return _balances[keccak256(did)];
    }

    function fundBy(address funder, bytes calldata did) external view override returns (uint256 result) {
        return _funds[funder][keccak256(did)];
    }

    function defund(bytes calldata did, uint256 amount) external override {
        bytes32 didHash = keccak256(did);
        require(_funds[msg.sender][didHash] >= amount, "insufficient balance");
        _funds[msg.sender][didHash] -= amount;
        _balances[didHash] -= amount;

        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "SHNZ transfer failed");

        emit Defunded(msg.sender, amount, did);
    }

    function earnings() external view override returns (uint256 result) {
        return _earnings;
    }

    function claimEarnings() external override {
        require(msg.sender == _creator, "only creator");
        uint256 amount = _earnings;
        require(amount > 0, "nothing to claim");
        _earnings = 0;

        (bool ok, ) = _creator.call{value: amount}("");
        require(ok, "SHNZ transfer failed");

        emit EarningsClaimed(_creator, amount);
    }

    function consume(bytes calldata did) external override {
        require(
            HOST_REGISTRY_CONTRACT.isRegistered(msg.sender),
            "caller is not a registered host"
        );

        uint256 cost = this.price();

        bytes32 didHash = keccak256(did);
        require(_balances[didHash] >= cost, "insufficient DID balance");

        _balances[didHash] -= cost;
        _earnings += cost;

        emit Consumed(msg.sender, cost, did);
    }

    /// @dev Scaling constant for stake premium (1 SHNZ = 1e18 wei).
    uint256 private constant STAKE_SCALE = 1e18;

    /// @dev Maximum stake multiplier in basis points (30000 = 3x additional, total 4x).
    uint256 private constant MAX_PREMIUM_BPS = 30_000;

    function price() external view override returns (uint256 result) {
        uint256 raw;
        if (address(_pricingContract) != address(0)) {
            raw = _pricingContract.price();
        } else {
            uint256 avgRate = rate();
            uint256 avgComplexity = complexity();
            uint256 basePrice = avgRate * avgComplexity;

            uint256 premiumBps = 0;
            if (_totalStake > 0) {
                premiumBps = MAX_PREMIUM_BPS * _totalStake / (_totalStake + STAKE_SCALE);
            }
            raw = basePrice * (BPS + premiumBps) / BPS;
        }
        return raw * (BPS - SHINZO_CUT_BPS) / BPS;
    }

    function pricingContract() external view override returns (IViewPricing result) {
        return _pricingContract;
    }
}
