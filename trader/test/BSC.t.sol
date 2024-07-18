// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2} from "forge-std/Test.sol";
import {Trader} from "../src/Trader.sol";
import {IERC20} from "forge-std/interfaces/IERC20.sol";

struct SwapParamsData {
    address buyPool;
    address sellPool;
    uint256 amount;
    uint256 deadline;
    uint256 types; //池类型：1是UniswapV3,2是UniswapV2
}

contract TraderTest is Test {
    string MAINNET_RPC_URL = vm.rpcUrl("bsc");

    uint256 mainnetFork;
    uint256 mainnetFork2;
    uint256 mainnetFork3;

    address testAddress = address(0x36F18e8B735592dE9A32A417e482e106eAa0C77A);
    address wethAddress = address(0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c);
    address richAddress = address(0x628ff693426583D9a7FB391E54366292F509D457);

    function setUp() public {
        mainnetFork = vm.createFork(MAINNET_RPC_URL, 40057154);
    }

    // forge test --match-test test_BscSwap -vvvv
    function test_BscSwap() public {
        vm.selectFork(mainnetFork);
        // 给测试地址发送ETH
        vm.deal(testAddress, 1 ether);

        // 切换到testAddress
        vm.startPrank(testAddress);
        Trader trader = new Trader(); // 部署新合约
        vm.stopPrank();

        // 给合约充值WETH
        IERC20 token = IERC20(wethAddress);
        uint256 amount = 10 ether; // 多点余额不借贷
        // uint256 amount = 0.1 ether; // 少点余额借贷
        vm.startPrank(richAddress);
        token.transfer(address(trader), amount);
        vm.stopPrank();

        uint256 balance = token.balanceOf(address(trader));
        assertEq(balance, amount, "Trader balance is incorrect");

        //测试数据
        SwapParamsData memory params;
        params.amount = 1.943925 ether; //params.amount>amount时将会用借贷完成交易
        params.deadline = 40057154;
        params.types = (20 << 80) | (25 << 64) | (2 << 32) | 2;
        params.buyPool = 0xB450CBF17F6723Ef9c1bf3C3f0e0aBA368D09bF5;
        params.sellPool = 0xfff58a50Fdde55F5B5626BB66dE476F63155E078;

        vm.startPrank(testAddress);
        bytes memory data = abi.encodeWithSignature("swap()", params); //调用套利合约
        // console2.log(data.length);
        // console2.logBytes(data);
        (bool success,) = address(trader).call(data);
        assertTrue(success, "Call to swap failed");

        vm.stopPrank();

        balance = token.balanceOf(address(trader));
        console2.log(balance);
    }
}
