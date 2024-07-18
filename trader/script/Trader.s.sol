// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, console2} from "forge-std/Script.sol";
import {Trader} from "../src/Trader.sol";

// forge script --chain sepolia script/Trader.s.sol:TraderScript --rpc-url $RPC_TESTNET --broadcast -vvvv
// forge script --chain mainnet script/Trader.s.sol:TraderScript --rpc-url $RPC_MAINNET --gas-price 3600000000 --broadcast -vvvv
contract TraderScript is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY2");
        vm.startBroadcast(deployerPrivateKey);

        Trader trader = new Trader();

        vm.stopBroadcast();

        console2.log(address(trader));
    }
}
