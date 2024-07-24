// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2} from "forge-std/Test.sol";
import {BSCTrader} from "../src/BSCTrader.sol";
import {IERC20} from "forge-std/interfaces/IERC20.sol";

struct SwapParamsData {
    address buyPool;
    address sellPool;
    address baseToken;
    address borrowPool;
    uint256 amount;
    uint256 types; //池类型：1是UniswapV3,2是UniswapV2
}

interface BB {
    function isBlackListed(address addr) external returns (bool);
}

contract TraderTest is Test {
    string MAINNET_RPC_URL = vm.rpcUrl("bsc");

    uint256 mainnetFork;
    uint256 mainnetFork2;
    uint256 mainnetFork3;

    address testAddress = address(0x36F18e8B735592dE9A32A417e482e106eAa0C77A);
    address baseAddress = address(0x55d398326f99059fF775485246999027B3197955);
    address richAddress = address(0x98cF4F4B03a4e967D54a3d0aeC9fCA90851f2Cca);

    address ba = address(0x04F46CDfE8DD348E41902eEF1aFF19AcE1661F4c);

    uint256 ablock = 40747767;

    function setUp() public {
        mainnetFork = vm.createFork(MAINNET_RPC_URL, ablock);
    }

    // forge test --match-test test_BSCSwap -vvvv
    function test_BSCSwap() public {
        vm.selectFork(mainnetFork);

        // 给测试地址发送ETH
        vm.deal(testAddress, 1 ether);

        // 切换到testAddress
        vm.startPrank(testAddress);
        BSCTrader trader = new BSCTrader(); // 部署新合约
        vm.stopPrank();

        // 给合约充值WETH
        IERC20 token = IERC20(baseAddress);
        uint256 amount = 100 ether; // 多点余额不借贷
        // uint256 amount = 0.1 ether; // 少点余额借贷
        // uint256 amount = 100000; //wei
        vm.startPrank(richAddress);
        token.transfer(address(trader), amount);
        vm.stopPrank();

        //测试数据
        SwapParamsData memory params;
        params.amount = 0.179437 ether; //params.amount>amount时将会用借贷完成交易
        // params.amount = 37688; // wei
        uint256 deadline = ablock;
        uint8 borrow = 0;
        params.types = (deadline << 72) | (uint256(borrow) << 64) | (3 << 48) | (3 << 32) | (25 << 16) | 25;
        params.buyPool = 0x1936be860d93B0Ff98f3a9b83254D61A78930B76;
        params.sellPool = 0x4f55423de1049d3CBfDC72f8A40f8A6f554f92aa;
        params.borrowPool = 0x8cb829111c90E0101492d5A1aa011F09614129E7;
        params.baseToken = baseAddress;
        vm.startPrank(testAddress);
        bytes memory data = abi.encodeWithSignature("swap()", params); //调用套利合约
        // console2.log(data.length);
        console2.logBytes(data);

        bool b = BB(ba).isBlackListed(address(trader));
        console2.log(b);

        (bool success,) = address(trader).call(data);
        assertTrue(success, "Call to swap failed");

        vm.stopPrank();

        uint256 balance = token.balanceOf(address(trader));
        console2.log(balance);
    }
}

// 0x8119c065
// 000000000000000000000000d8c8a2b125527bf97c8e4845b25de7e964468f77
// 000000000000000000000000fad57d2039c21811c8f2b5d5b65308aa99d31559
// 000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
// 0000000000000000000000003416cf6c708da44db2624d63ea0aaef7113527c6
// 000000000000000000000000000000000000000000000000000000000066550e
// 00000000000000000000000000000000000000013522100000020001001e001e
