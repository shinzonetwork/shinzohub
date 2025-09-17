// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console} from "forge-std/Test.sol";
import {Outpost} from "../src/Outpost.sol";

contract OutpostTest is Test {
    Outpost public outpost;

    function setUp() public {
        outpost = new Outpost();
    }

    function test_Payment() public {
        outpost.payment{value: 100 wei}("test", "test");
        outpost.payment{value: 100 wei}("test", "test");
        assertEq(outpost.getPaymentIndex(address(this)), 2);
    }

    function test_ExpirePayment() public {
        uint256 paymentIndex = outpost.payment{value: 100 wei}("test", "test");
        vm.warp(block.timestamp + 3600);
        outpost.expirePayment(address(this), paymentIndex);
        assertEq(outpost.getPaymentIndex(address(this)), 1);
    }
}
