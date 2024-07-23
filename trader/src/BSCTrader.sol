// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import {IUniswapV3Pool} from "@uniswap/v3-core/contracts/interfaces/IUniswapV3Pool.sol";
import "@uniswap/v3-core/contracts/interfaces/callback/IUniswapV3SwapCallback.sol";
import "@uniswap/v3-core/contracts/libraries/LowGasSafeMath.sol";
import "@uniswap/v3-core/contracts/interfaces/IERC20Minimal.sol";
import "@uniswap/v3-core/contracts/libraries/TransferHelper.sol";
import "@uniswap/v2-core/interfaces/IUniswapV2Pair.sol";
import "@uniswap/v3-core/contracts/libraries/SafeCast.sol";
import {CallbackValidation} from "./PancakeCV.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import "pancake-v3-contracts/v3-core/contracts/interfaces/callback/IPancakeV3FlashCallback.sol";
import "pancake-v3-contracts/v3-core/contracts/interfaces/callback/IPancakeV3SwapCallback.sol";
import "pancake-v3-contracts/v3-core/contracts/interfaces/IPancakeV3Pool.sol";
import "./interfaces/IAlgebraSwapCallback.sol";

contract BSCTrader is
    Ownable,
    IPancakeV3FlashCallback,
    IUniswapV3SwapCallback,
    IPancakeV3SwapCallback,
    IAlgebraSwapCallback
{
    using LowGasSafeMath for uint256;
    using SafeCast for uint256;

    // 完成所需要的参数
    struct SwapParamsData {
        address buyPool;
        address sellPool;
        address baseToken;
        address borrowPool;
        uint256 amount;
        uint16 buyPoolType; //池类型：1是UniswapV2,2是UniswapV3,3是PancakeV3,4是Algebra,5是SolidlyV3
        uint16 sellPoolType;
        uint16 buyPoolFee; //1e4
        uint16 sellPoolFee; //1e4
    }

    struct SwapCallbackData {
        address tokenIn;
        address tokenOut;
        uint24 fee;
    }

    /// @dev The minimum value that can be returned from #getSqrtRatioAtTick. Equivalent to getSqrtRatioAtTick(MIN_TICK)
    uint160 internal constant MIN_SQRT_RATIO = 4295128739;
    /// @dev The maximum value that can be returned from #getSqrtRatioAtTick. Equivalent to getSqrtRatioAtTick(MAX_TICK)
    uint160 internal constant MAX_SQRT_RATIO = 1461446703485210103287273052203988822378723970342;

    address public immutable factorypancakeV3 = 0x41ff9AA7e16B8B1a8a8dc4f0eFacd93D02d071c9;

    bool private hasBorrow = false;

    constructor() Ownable(msg.sender) {}

    function withdraw(address token) external onlyOwner {
        require(token != address(0), "W");
        uint256 balance = balances(token);
        if (balance > 0) {
            TransferHelper.safeTransfer(token, msg.sender, balance);
        }
    }

    function balances(address token) private view returns (uint256) {
        (bool success, bytes memory data) =
            token.staticcall(abi.encodeWithSelector(IERC20Minimal.balanceOf.selector, address(this)));
        require(success && data.length >= 32, "EB");
        return abi.decode(data, (uint256));
    }

    function getAmountOut(uint256 amountIn, uint256 reserveIn, uint256 reserveOut, uint16 fee)
        internal
        pure
        returns (uint256 amountOut)
    {
        require(amountIn > 0, "IIA");
        require(reserveIn > 0 && reserveOut > 0, "IL");
        uint256 amountInWithFee = amountIn.mul(uint256(10000 - fee));
        uint256 numerator = amountInWithFee.mul(reserveOut);
        uint256 denominator = reserveIn.mul(10000).add(amountInWithFee);
        amountOut = numerator / denominator;
    }

    function swap() external {
        SwapParamsData memory data;
        uint256 deadline; //过期块号
        uint8 borrow; //如果需要借贷，false表示借token0,true表示借token1
        assembly {
            let dataLength := calldatasize()
            if iszero(eq(dataLength, 196)) { revert(0, 0) }
            let buy := calldataload(4)
            let sell := calldataload(36)
            let ba := calldataload(68)
            let bo := calldataload(100)
            let value := calldataload(132)
            let tmp := calldataload(164)
            mstore(data, buy)
            mstore(add(data, 0x20), sell)
            mstore(add(data, 0x40), ba)
            mstore(add(data, 0x60), bo)
            mstore(add(data, 0x80), value)
            deadline := and(shr(72, tmp), 0xFFFFFFFFFFFFFFFF)
            borrow := and(shr(64, tmp), 0xFF)
            mstore(add(data, 0xa0), and(shr(48, tmp), 0xFFFF))
            mstore(add(data, 0xc0), and(shr(32, tmp), 0xFFFF))
            mstore(add(data, 0xe0), and(shr(16, tmp), 0xFFFF))
            mstore(add(data, 0x100), and(tmp, 0xFFFF))
        }

        require(block.number <= deadline, "D");

        uint256 balanceBefore = balances(data.baseToken);
        // 如果余额太少,就借入token完成交易
        if (data.amount > balanceBefore) {
            // borrow(data);
            hasBorrow = true;
            (uint256 a0, uint256 a1) = borrow == 0 ? (data.amount, uint256(0)) : (uint256(0), data.amount);
            IPancakeV3Pool(data.borrowPool).flash(address(this), a0, a1, abi.encode(data));
        } else {
            _swap(data);
            uint256 balanceAfter = balances(data.baseToken);
            require(balanceAfter >= balanceBefore, "E");
        }
    }

    function swapUniswapV3(IUniswapV3Pool pool, int256 amount, address token, address token0, address token1)
        private
        returns (uint256 amountOut)
    {
        (int256 amount0, int256 amount1) = pool.swap(
            address(this),
            token == token0,
            amount,
            (token == token0 ? MIN_SQRT_RATIO + 1 : MAX_SQRT_RATIO - 1),
            abi.encode(
                SwapCallbackData({
                    tokenIn: token == token0 ? token0 : token1,
                    tokenOut: token == token0 ? token1 : token0,
                    fee: pool.fee()
                })
            )
        );
        amountOut = uint256(-(token == token0 ? amount1 : amount0));
    }

    function swapUniswapV2(IUniswapV2Pair pool, uint256 amount, address token, address token0, uint16 fee)
        private
        returns (uint256 amountOut)
    {
        (uint256 reserve0, uint256 reserve1,) = pool.getReserves();
        (reserve0, reserve1) = token == token0 ? (reserve0, reserve1) : (reserve1, reserve0);
        amountOut = getAmountOut(amount, reserve0, reserve1, fee);
        (uint256 amount0Out, uint256 amount1Out) = token == token0 ? (uint256(0), amountOut) : (amountOut, uint256(0));
        TransferHelper.safeTransfer(token, address(pool), amount);
        pool.swap(amount0Out, amount1Out, address(this), new bytes(0));
    }

    function _swap(SwapParamsData memory data) private {
        // 先在sellPool卖出baseTokena
        uint256 amountOut;
        if (data.sellPoolType == 1) {
            IUniswapV2Pair pool = IUniswapV2Pair(data.sellPool);
            address token0 = pool.token0();
            amountOut = swapUniswapV2(pool, data.amount, data.baseToken, token0, data.sellPoolFee);
        } else {
            IUniswapV3Pool pool = IUniswapV3Pool(data.sellPool);
            address token0 = pool.token0();
            address token1 = pool.token1();
            amountOut = swapUniswapV3(pool, data.amount.toInt256(), data.baseToken, token0, token1);
        }

        //然后在buyPool买入baseToken
        if (data.buyPoolType == 1) {
            IUniswapV2Pair pool = IUniswapV2Pair(data.buyPool);
            address token0 = pool.token0();
            address token1 = pool.token1();
            swapUniswapV2(pool, amountOut, data.baseToken == token0 ? token1 : token0, token0, data.buyPoolFee);
        } else {
            IUniswapV3Pool pool = IUniswapV3Pool(data.buyPool);
            address token0 = pool.token0();
            address token1 = pool.token1();
            swapUniswapV3(pool, amountOut.toInt256(), data.baseToken == token0 ? token1 : token0, token0, token1);
        }
    }

    function pancakeV3FlashCallback(uint256 fee0, uint256 fee1, bytes calldata data) external override {
        SwapParamsData memory decoded = abi.decode(data, (SwapParamsData));
        require(hasBorrow && msg.sender == decoded.borrowPool, "EP");
        uint256 balanceBefore = balances(decoded.baseToken);
        _swap(decoded);
        uint256 balanceAfter = balances(decoded.baseToken);
        require(balanceAfter >= balanceBefore, "E");
        uint256 amountMin = LowGasSafeMath.add(decoded.amount, fee0 > 0 ? fee0 : fee1);
        if (amountMin > 0) {
            TransferHelper.safeTransfer(decoded.baseToken, msg.sender, amountMin);
        }
        hasBorrow = false;
    }

    function uniswapV3SwapCallback(int256 amount0Delta, int256 amount1Delta, bytes calldata data) external override {
        v3SwapCallback(amount0Delta, amount1Delta, data);
    }

    function pancakeV3SwapCallback(int256 amount0Delta, int256 amount1Delta, bytes calldata data) external override {
        v3SwapCallback(amount0Delta, amount1Delta, data);
    }

    function algebraSwapCallback(int256 amount0Delta, int256 amount1Delta, bytes calldata data) external override {
        v3SwapCallback(amount0Delta, amount1Delta, data);
    }

    function v3SwapCallback(int256 amount0Delta, int256 amount1Delta, bytes calldata _data) private {
        require(amount0Delta > 0 || amount1Delta > 0, "S");
        SwapCallbackData memory data = abi.decode(_data, (SwapCallbackData));
        CallbackValidation.verifyCallback(factorypancakeV3, data.tokenIn, data.tokenOut, data.fee);

        (bool isExactInput, uint256 amountToPay) = amount0Delta > 0
            ? (data.tokenIn < data.tokenOut, uint256(amount0Delta))
            : (data.tokenOut < data.tokenIn, uint256(amount1Delta));

        if (isExactInput) {
            TransferHelper.safeTransfer(data.tokenIn, msg.sender, amountToPay);
        } else {
            TransferHelper.safeTransfer(data.tokenOut, msg.sender, amountToPay);
        }
    }
}
