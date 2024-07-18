// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2} from "forge-std/Test.sol";
import {Trader} from "../src/Trader.sol";
import {IERC20} from "forge-std/interfaces/IERC20.sol";

struct SwapParamsData {
    address buyPool;
    address sellPool;
    address baseToken;
    address borrowPool;
    uint256 amount;
    uint256 types; //池类型：1是UniswapV3,2是UniswapV2
}

contract TraderTest is Test {
    string MAINNET_RPC_URL = vm.rpcUrl("mainnet");

    uint256 mainnetFork;
    uint256 mainnetFork2;
    uint256 mainnetFork3;

    address testAddress = address(0x36F18e8B735592dE9A32A417e482e106eAa0C77A);
    address baseAddress = address(0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2);
    address richAddress = address(0x8EB8a3b98659Cce290402893d0123abb75E3ab28);

    function setUp() public {
        mainnetFork = vm.createFork(MAINNET_RPC_URL, 20147596);
        mainnetFork2 = vm.createFork(MAINNET_RPC_URL, 20160693);
        mainnetFork3 = vm.createFork(MAINNET_RPC_URL, 20332763);
    }

    // forge test --match-test test_TSwap -vvvv
    function test_TSwap() public {
        vm.selectFork(mainnetFork3);
        // 给测试地址发送ETH
        vm.deal(testAddress, 1 ether);

        // 切换到testAddress
        vm.startPrank(testAddress);
        Trader trader = new Trader(); // 部署新合约
        vm.stopPrank();

        // 给合约充值WETH
        IERC20 token = IERC20(baseAddress);
        // uint256 amount = 10 ether; // 多点余额不借贷
        // uint256 amount = 0.1 ether; // 少点余额借贷
        // uint256 amount = 100000; //wei
        // vm.startPrank(richAddress);
        // token.transfer(address(trader), amount);
        // vm.stopPrank();

        // uint256 balance = token.balanceOf(address(trader));
        // assertEq(balance, amount, "Trader balance is incorrect");

        //测试数据
        SwapParamsData memory params;
        params.amount = 0.448037 ether; //params.amount>amount时将会用借贷完成交易
        // params.amount = 6706446; // wei
        uint256 deadline = 20332763;
        uint8 borrow = 0;
        params.types = (deadline << 72) | (uint256(borrow) << 64) | (2 << 48) | (2 << 32) | (30 << 16) | 30;
        params.buyPool = 0xC4b26b26d720467d96E18f08664A888d4116cEa6;
        params.sellPool = 0x99dFDE431b40321a35dEb6AEb55cf338dDD6eccd;
        params.borrowPool = 0x11b815efB8f581194ae79006d24E0d814B7697F6;
        params.baseToken = baseAddress;
        vm.startPrank(testAddress);
        bytes memory data = abi.encodeWithSignature("swap()", params); //调用套利合约
        // console2.log(data.length);
        console2.logBytes(data);
        (bool success,) = address(trader).call(data);
        assertTrue(success, "Call to swap failed");

        vm.stopPrank();

        uint256 balance = token.balanceOf(address(trader));
        console2.log(balance);
    }

    // forge test --match-test testFail_Withdraw -vvvv
    function testFail_Withdraw() public {
        vm.selectFork(mainnetFork);
        // 给测试地址发送ETH
        vm.deal(testAddress, 1 ether);
        // 给测试地址发送ERC20代币
        IERC20 token = IERC20(baseAddress);
        uint256 amount = 1000 * 10 ** 18; // 要发送的代币数量
        vm.startPrank(richAddress);
        token.transfer(testAddress, amount);
        vm.stopPrank();

        // 切换到testAddress
        vm.startPrank(testAddress);

        // 部署新合约
        Trader trader = new Trader();

        // 给测试地址发送ETH
        // vm.deal(address(trader), 1 ether);
        token.transfer(address(trader), 1 ether);

        uint256 balance = token.balanceOf(address(trader));
        assertEq(balance, 1 ether, "Trader balance is incorrect");
        vm.stopPrank();

        // 提取余额会失败
        trader.withdraw(address(0));
    }

    // forge test --match-test test_Swap -vvvv
    function test_Swap() public {
        vm.selectFork(mainnetFork2);
        // 给测试地址发送ETH
        vm.deal(testAddress, 1 ether);

        // 切换到testAddress
        vm.startPrank(testAddress);
        Trader trader = new Trader(); // 部署新合约
        vm.stopPrank();

        // 给合约充值WETH
        IERC20 token = IERC20(baseAddress);
        // uint256 amount = 10 ether; // 多点余额不借贷
        uint256 amount = 0.1 ether; // 少点余额借贷
        vm.startPrank(richAddress);
        token.transfer(address(trader), amount);
        vm.stopPrank();

        uint256 balance = token.balanceOf(address(trader));
        assertEq(balance, amount, "Trader balance is incorrect");

        //测试数据
        SwapParamsData memory params;
        // params.deadline = 20160693;
        params.amount = 0.4099128571428572 ether; //params.amount>amount时将会用借贷完成交易
        params.types = (30 << 80) | (30 << 64) | (2 << 32) | 2;
        params.buyPool = 0xF1F85b2C54a2bD284B1cf4141D64fD171Bd85539;
        params.sellPool = 0xf80758aB42C3B07dA84053Fd88804bCB6BAA4b5c;

        vm.startPrank(testAddress);
        bytes memory data = abi.encodeWithSignature("swap()", params); //调用套利合约
        // console2.log(data.length);
        // console2.logBytes(data);
        (bool success,) = address(trader).call(data);
        assertTrue(success, "Call to swap failed");

        // 提取余额
        trader.withdraw(address(0));
        balance = token.balanceOf(testAddress);
        // assertEq(balance, 10006752631053444480, "Balance is incorrect"); //不借贷
        assertEq(balance, 106547674624873051, "Balance is incorrect"); //借贷
        vm.stopPrank();
    }

    // forge test --match-test test_FailSwap -vvvv
    function test_FailSwap() public {
        vm.selectFork(mainnetFork);

        // 给测试地址发送ETH
        vm.deal(testAddress, 1 ether);

        // 给测试地址发送ERC20代币
        IERC20 token = IERC20(baseAddress);
        uint256 amount = 1000 * 10 ** 18; // 要发送的代币数量
        vm.startPrank(richAddress);
        token.transfer(testAddress, amount);
        vm.stopPrank();

        uint256 balance = token.balanceOf(testAddress);
        assertEq(balance, amount, "Balance is incorrect");

        // 切换到testAddress
        vm.startPrank(testAddress);
        // 部署新合约
        Trader trader = new Trader();
        // 给测试地址发送ETH
        // vm.deal(address(trader), 1 ether);
        token.transfer(address(trader), 1 ether);

        balance = token.balanceOf(address(trader));
        assertEq(balance, 1 ether, "Trader balance is incorrect");

        vm.expectRevert(bytes("E"));

        SwapParamsData memory params;
        params.amount = 1 ether;
        // params.deadline = 20147596;
        params.types = (30 << 64) | (1 << 32) | 2;
        params.buyPool = 0x11b815efB8f581194ae79006d24E0d814B7697F6;
        params.sellPool = 0x74C99F3f5331676f6AEc2756e1F39b4FC029a83E;

        bytes memory data = abi.encodeWithSignature("swap()", params);
        (bool success,) = address(trader).call(data);
        assertTrue(success, "Call to swap failed");
        vm.stopPrank();
    }
}

// 0x8119c065
// 000000000000000000000000d8c8a2b125527bf97c8e4845b25de7e964468f77
// 000000000000000000000000fad57d2039c21811c8f2b5d5b65308aa99d31559
// 000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
// 0000000000000000000000003416cf6c708da44db2624d63ea0aaef7113527c6
// 000000000000000000000000000000000000000000000000000000000066550e
// 00000000000000000000000000000000000000013522100000020001001e001e
