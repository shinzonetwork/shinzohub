// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "forge-std/Test.sol";
import "./View.sol";
import "./IView.sol";

contract MockHostRegistry {
    mapping(address => bool) public registered;

    function setRegistered(address addr, bool val) external {
        registered[addr] = val;
    }

    function isRegistered(address addr) external view returns (bool) {
        return registered[addr];
    }
}

contract MockPricing {
    uint256 public price;

    constructor(uint256 _price) {
        price = _price;
    }
}

contract Receiver {
    receive() external payable {}
}

contract ViewTest is Test {
    View public v;
    MockHostRegistry public mockHost;
    address public creatorAddr;
    address public host1;
    address public host2;
    address public funder1;

    bytes constant DID1 = "did:key:z6Mk1";
    bytes constant DID2 = "did:key:z6Mk2";

    function setUp() public {
        creatorAddr = address(new Receiver());
        host1 = address(0xA001);
        host2 = address(0xA002);
        funder1 = address(0xB001);

        mockHost = new MockHostRegistry();
        vm.etch(
            0x0000000000000000000000000000000000000211,
            address(mockHost).code
        );

        MockHostRegistry(0x0000000000000000000000000000000000000211).setRegistered(host1, true);
        MockHostRegistry(0x0000000000000000000000000000000000000211).setRegistered(host2, true);

        v = new View("TestView", creatorAddr, address(0));

        vm.deal(host1, 100 ether);
        vm.deal(host2, 100 ether);
        vm.deal(funder1, 100 ether);
        vm.deal(creatorAddr, 1 ether);
    }

    function test_Constructor() public view {
        assertEq(v.name(), "TestView");
        assertEq(v.creator(), creatorAddr);
        assertEq(address(v.pricingContract()), address(0));
    }

    function test_Stake() public {
        vm.prank(host1);
        v.stake{value: 1 ether}();

        assertEq(v.stakeOf(host1), 1 ether);
        assertEq(v.totalStake(), 1 ether);
    }

    function test_Stake_MultipleStakers() public {
        vm.prank(host1);
        v.stake{value: 2 ether}();
        vm.prank(host2);
        v.stake{value: 3 ether}();

        assertEq(v.stakeOf(host1), 2 ether);
        assertEq(v.stakeOf(host2), 3 ether);
        assertEq(v.totalStake(), 5 ether);
    }

    function test_Stake_ZeroReverts() public {
        vm.prank(host1);
        vm.expectRevert("must send SHNZ");
        v.stake{value: 0}();
    }

    function test_Unstake() public {
        vm.startPrank(host1);
        v.stake{value: 5 ether}();
        uint256 balBefore = host1.balance;
        v.unstake(2 ether);
        vm.stopPrank();

        assertEq(v.stakeOf(host1), 3 ether);
        assertEq(v.totalStake(), 3 ether);
        assertEq(host1.balance, balBefore + 2 ether);
    }

    function test_Unstake_InsufficientReverts() public {
        vm.prank(host1);
        vm.expectRevert("insufficient stake");
        v.unstake(1 ether);
    }

    function test_Report() public {
        vm.prank(host1);
        v.report(50, 1000);

        address[] memory h = v.hosts();
        assertEq(h.length, 1);
        assertEq(h[0], host1);
        assertEq(v.complexity(), 50);
        assertEq(v.rate(), 1000);
    }

    function test_Report_MultipleHosts() public {
        vm.prank(host1);
        v.report(40, 800);
        vm.prank(host2);
        v.report(60, 1200);

        assertEq(v.complexity(), 50);
        assertEq(v.rate(), 1000);
    }

    function test_Report_UpdateExisting() public {
        vm.prank(host1);
        v.report(50, 1000);
        vm.prank(host1);
        v.report(80, 2000);

        address[] memory h = v.hosts();
        assertEq(h.length, 1);
        assertEq(v.complexity(), 80);
        assertEq(v.rate(), 2000);
    }

    function test_Report_NotRegisteredHost() public {
        address notHost = address(0xDEAD);
        vm.prank(notHost);
        vm.expectRevert("caller is not a registered host");
        v.report(50, 1000);
    }

    function test_Report_ComplexityOutOfRange() public {
        vm.prank(host1);
        vm.expectRevert("complexity must be 1-100");
        v.report(0, 1000);

        vm.prank(host1);
        vm.expectRevert("complexity must be 1-100");
        v.report(101, 1000);
    }

    function test_Fund() public {
        vm.prank(funder1);
        v.fund{value: 1 ether}(DID1);

        assertEq(v.fundOf(DID1), 1 ether);
        assertEq(v.fundBy(funder1, DID1), 1 ether);
    }

    function test_Fund_ZeroReverts() public {
        vm.prank(funder1);
        vm.expectRevert("must send SHNZ");
        v.fund{value: 0}(DID1);
    }

    function test_FundFor() public {
        address delegate = address(0xC001);
        vm.prank(funder1);
        v.fundFor{value: 2 ether}(delegate, DID1);

        assertEq(v.fundOf(DID1), 2 ether);
        assertEq(v.fundBy(delegate, DID1), 2 ether);
        assertEq(v.fundBy(funder1, DID1), 0);
    }

    function test_Fund_MultipleFunders() public {
        address funder2 = address(0xB002);
        vm.deal(funder2, 10 ether);

        vm.prank(funder1);
        v.fund{value: 3 ether}(DID1);
        vm.prank(funder2);
        v.fund{value: 2 ether}(DID1);

        assertEq(v.fundOf(DID1), 5 ether);
        assertEq(v.fundBy(funder1, DID1), 3 ether);
        assertEq(v.fundBy(funder2, DID1), 2 ether);
    }

    function test_Fund_DifferentDIDs() public {
        vm.prank(funder1);
        v.fund{value: 1 ether}(DID1);
        vm.prank(funder1);
        v.fund{value: 2 ether}(DID2);

        assertEq(v.fundOf(DID1), 1 ether);
        assertEq(v.fundOf(DID2), 2 ether);
    }

    function test_Defund() public {
        vm.startPrank(funder1);
        v.fund{value: 5 ether}(DID1);
        uint256 balBefore = funder1.balance;
        v.defund(DID1, 2 ether);
        vm.stopPrank();

        assertEq(v.fundOf(DID1), 3 ether);
        assertEq(v.fundBy(funder1, DID1), 3 ether);
        assertEq(funder1.balance, balBefore + 2 ether);
    }

    function test_Defund_InsufficientReverts() public {
        vm.prank(funder1);
        vm.expectRevert("insufficient balance");
        v.defund(DID1, 1 ether);
    }

    function test_Price_NoHosts() public view {
        assertEq(v.price(), 0);
    }

    function test_Price_NoStake() public {
        vm.prank(host1);
        v.report(10, 100);

        uint256 basePrice = 10 * 100;
        uint256 expected = basePrice * 9500 / 10000;
        assertEq(v.price(), expected);
    }

    function test_Price_WithStake() public {
        vm.prank(host1);
        v.report(10, 100);

        vm.prank(host1);
        v.stake{value: 1 ether}();

        uint256 basePrice = 10 * 100;
        uint256 premiumBps = 30000 * 1 ether / (1 ether + 1 ether);
        uint256 raw = basePrice * (10000 + premiumBps) / 10000;
        uint256 expected = raw * 9500 / 10000;
        assertEq(v.price(), expected);
    }

    function test_Consume() public {
        vm.prank(host1);
        v.report(10, 100);

        vm.prank(funder1);
        v.fund{value: 10 ether}(DID1);

        uint256 cost = v.price();
        uint256 balBefore = v.fundOf(DID1);

        vm.prank(host1);
        v.consume(DID1);

        assertEq(v.fundOf(DID1), balBefore - cost);
        assertEq(v.earnings(), cost);
    }

    function test_Consume_NotRegisteredHost() public {
        address notHost = address(0xDEAD);
        vm.prank(notHost);
        vm.expectRevert("caller is not a registered host");
        v.consume(DID1);
    }

    function test_Consume_InsufficientBalance() public {
        vm.prank(host1);
        v.report(10, 100);

        vm.prank(host1);
        vm.expectRevert("insufficient DID balance");
        v.consume(DID1);
    }

    function test_ClaimEarnings() public {
        vm.prank(host1);
        v.report(10, 100);

        vm.prank(funder1);
        v.fund{value: 10 ether}(DID1);

        vm.prank(host1);
        v.consume(DID1);

        uint256 earned = v.earnings();
        assertTrue(earned > 0);

        uint256 creatorBal = creatorAddr.balance;
        vm.prank(creatorAddr);
        v.claimEarnings();

        assertEq(v.earnings(), 0);
        assertEq(creatorAddr.balance, creatorBal + earned);
    }

    function test_ClaimEarnings_NotCreator() public {
        vm.prank(funder1);
        vm.expectRevert("only creator");
        v.claimEarnings();
    }

    function test_ClaimEarnings_NothingToClaim() public {
        vm.prank(creatorAddr);
        vm.expectRevert("nothing to claim");
        v.claimEarnings();
    }

    function test_Hosts_Empty() public view {
        address[] memory h = v.hosts();
        assertEq(h.length, 0);
    }

    function test_Rate_NoHosts() public view {
        assertEq(v.rate(), 0);
    }

    function test_Complexity_NoHosts() public view {
        assertEq(v.complexity(), 0);
    }

    function test_CustomPricing() public {
        MockPricing mp = new MockPricing(5000);
        View vCustom = new View("Custom", creatorAddr, address(mp));

        uint256 expected = 5000 * 9500 / 10000;
        assertEq(vCustom.price(), expected);
    }

    function test_Consume_MultipleConsumptions() public {
        vm.prank(host1);
        v.report(10, 100);

        vm.prank(funder1);
        v.fund{value: 10 ether}(DID1);

        uint256 cost = v.price();

        vm.prank(host1);
        v.consume(DID1);
        vm.prank(host1);
        v.consume(DID1);

        assertEq(v.earnings(), cost * 2);
    }

    function test_StakeOf_NotStaked() public view {
        assertEq(v.stakeOf(address(0xFFFF)), 0);
    }

    function test_FundOf_NotFunded() public view {
        assertEq(v.fundOf(DID1), 0);
    }

    function test_FundBy_NotFunded() public view {
        assertEq(v.fundBy(funder1, DID1), 0);
    }
}
