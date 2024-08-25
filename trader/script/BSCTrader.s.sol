// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, console2} from "forge-std/Script.sol";
import {BSCTrader} from "../src/BSCTrader.sol";

// forge script --chain bsc trader/script/BSCTrader.s.sol:TraderScript --rpc-url $RPC_BSC --broadcast -vvvv
contract TraderScript is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY2");
        vm.startBroadcast(deployerPrivateKey);

        BSCTrader trader = new BSCTrader();

        vm.stopBroadcast();

        console2.log(address(trader));
    }
}
