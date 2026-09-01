// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2} from "forge-std/Test.sol";
import {Trader} from "../src/Trader.sol";
import {IERC20} from "forge-std/interfaces/IERC20.sol";

// Robinhood Chain (4663) fork 测试
// 链上验证过的池子 (factory 0x1f7d7550, 官方 V3 fee-tweaker 部署):
//   - 0x52e65b17: WETH/USDG 0.01% 主力池 (fee=100), flash 借贷池
//   - 0x69bfaf19: WETH/USDG 0.05% 池 (fee=500)
// 链上价格 tick=-1 (1 WETH ≈ 2450 USDG), 无真实价差 -> 用 vm.store 修改买池 slot0 制造 30% 折价
struct SwapParamsData {
    address buyPool;
    address sellPool;
    address baseToken;
    address borrowPool;
    uint256 amount;
    uint256 types; // deadline<<72 | borrow<<64 | buyType<<48 | sellType<<32 | buyFee<<16 | sellFee
}

// 砸盘合约: 测试合约本身实现 swap 回调, 用注入的 WETH 真实卖出砸低池子价格
contract RobinhoodTraderTest is Test {
    string RPC_URL = vm.rpcUrl("robinhood");

    uint256 fork;

    address constant WETH = 0x0Bd7D308f8E1639FAb988df18A8011f41EAcAD73;
    address constant USDG = 0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168;

    // V3 swap 回调: 池子扣 WETH (token0)
    function uniswapV3SwapCallback(int256 amount0Delta, int256 amount1Delta, bytes calldata) external {
        uint256 pay = amount0Delta > 0 ? uint256(amount0Delta) : uint256(amount1Delta);
        (bool ok,) = WETH.call(abi.encodeWithSignature("transfer(address,uint256)", msg.sender, pay));
        require(ok, "pay failed");
    }

    // 链上验证的池子 (t0=WETH, t1=USDG, factory=0x1f7d7550)
    address constant POOL_WETH_USDG_001 = 0x52e65B17fB6E5BA00Ed806f37Afcd2DaA50271Ca; // 0.01% 主力池
    address constant POOL_WETH_USDG_005 = 0x69BfaF19C9f377BB306a89aEd9F6B07e2c1a8d9a; // 0.05%
    address constant POOL_WETH_USDG_03 = 0xa9188730Fe85Be88ad499D7d52B099e800fB0334; // 0.3%

    function setUp() public {
        fork = vm.createFork(RPC_URL);
        vm.selectFork(fork);
    }

    // forge test --match-test test_RobinhoodFlashArb -vvvv
    // 验证全流程: 真实砸盘制造价差 -> flash 借 WETH -> 原价池卖出 -> 砸盘池买回
    // 不用 vm.store 改池子 slot0 (会破坏流动性区间), 而是注入 WETH 后真实大额卖出砸低价格
    function test_RobinhoodFlashArb() public {
        uint256 deadline = block.number + 1000;

        // 部署 Trader (本测试合约作为 owner)
        Trader trader = new Trader();

        // 1. 从链上真实持有者(主力池, ~2550 WETH) prank 转账砸盘资金 (参照 BSCTrader.t.sol 的 richAddress 模式)
        vm.prank(POOL_WETH_USDG_001);
        IERC20(WETH).transfer(address(this), 200 ether);

        // 2. 真实卖出 100 WETH 砸低 0.3% 池价格 (token0=WETH 入池 -> USDG/WETH 下跌)
        (bool ok2,) = POOL_WETH_USDG_03.call(
            abi.encodeWithSignature(
                "swap(address,bool,int256,uint160,bytes)",
                address(this), true, int256(100 ether), uint160(4295128739 + 1), ""
            )
        );
        require(ok2, "dump failed");

        // 砸盘后价格验证
        (bool ok3, bytes memory r3) = POOL_WETH_USDG_03.call(abi.encodeWithSignature("slot0()"));
        uint160 dumpedSqrt = abi.decode(r3, (uint160));
        (bool ok4, bytes memory r4) = POOL_WETH_USDG_001.call(abi.encodeWithSignature("slot0()"));
        uint160 refSqrt = abi.decode(r4, (uint160));
        console2.log("ref sqrt:", refSqrt, "dumped sqrt:", dumpedSqrt);
        assertLt(uint256(dumpedSqrt), uint256(refSqrt), "dump should lower price");

        SwapParamsData memory params;
        params.amount = 10 ether; // flash 借 10 WETH (borrow=0 -> 借 token0)
        params.types = (deadline << 72) | (uint256(0) << 64) | (2 << 48) | (2 << 32) | (3000 << 16) | 100;
        //                 deadline  |  borrow=0(借token0) | buyType=2(V3) | sellType=2(V3) | buyFee | sellFee
        params.buyPool = POOL_WETH_USDG_03; // 砸盘池(0.3%, 低价): 买入 WETH
        params.sellPool = POOL_WETH_USDG_001; // 原价池(主力, 0.01%): 卖出 WETH
        params.borrowPool = POOL_WETH_USDG_005; // flash 借 WETH (0.05% 池, 注意不能与 buy/sell 同池, 否则 LOK 重入)
        params.baseToken = WETH;

        bytes memory data = abi.encodeWithSignature("swap()", params);
        assertEq(data.length, 196, "calldata must be 196 bytes");
        console2.logBytes(data);

        (bool success, bytes memory ret) = address(trader).call(data);
        if (!success) {
            // 解码 revert 原因 (E = 亏损, EP = 回调权限, D = 过期)
            console2.log("swap reverts, ret:", string(ret));
            revert("swap failed");
        }

        uint256 balance = IERC20(WETH).balanceOf(address(trader));
        console2.log("trader WETH balance after arb:", balance);
        assertGt(balance, 0, "no profit");

        // 提款验证 (owner = 本测试合约)
        trader.withdraw(WETH);
        uint256 got = IERC20(WETH).balanceOf(address(this));
        console2.log("test contract WETH after withdraw:", got);
        assertGt(got, 0, "withdraw failed");
    }

    // forge test --match-test test_RobinhoodPriceRead -vvvv
    // 验证链上池子价格读取 (fork 后 slot0 可用, fee 为动态值)
    function test_RobinhoodPriceRead() public {
        (bool ok, bytes memory r) = POOL_WETH_USDG_001.call(abi.encodeWithSignature("fee()"));
        assertTrue(ok, "fee() failed");
        uint256 feeVal = abi.decode(r, (uint256));
        console2.log("pool fee:", feeVal);
        assertTrue(feeVal > 0, "fee should be dynamic non-zero");

        (ok, r) = POOL_WETH_USDG_001.call(abi.encodeWithSignature("slot0()"));
        assertTrue(ok, "slot0() failed");
        uint160 sqrt = abi.decode(r, (uint160));
        console2.log("sqrtPriceX96:", sqrt);
        assertTrue(sqrt > 0, "sqrt should be non-zero");
    }
}
