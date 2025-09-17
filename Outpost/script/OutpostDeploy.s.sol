
// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, console} from "forge-std/Script.sol";
import {Outpost} from "../src/Outpost.sol";

contract OutpostScript is Script {
    Outpost public outpost;

    function setUp() public {}

    function run() public {
        vm.startBroadcast();

        outpost = new Outpost();

        vm.stopBroadcast();
    }
}
