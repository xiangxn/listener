// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
pragma experimental ABIEncoderV2;

import {Test, console2} from "forge-std/Test.sol";

import "../src/Trader.sol";

contract CallDataC {
    function swap() external pure returns (Trader.SwapParamsData memory data, uint256 deadline, uint8 borrow) {
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
    }
}

contract ParamsTest is Test {
    CallDataC callData;

    function setUp() public {
        callData = new CallDataC();
    }

    // forge test --match-test test_swapParams -vvvv
    function test_swapParams() public {
        uint256 a = 40725374; //deadline
        a = a << 72;
        a = a | (1 << 64); // 借token1
        a = a | (1 << 48); //buyPoolType
        a = a | (3 << 32); //sellPoolType
        a = a | (25 << 16); // buyPoolFee
        a = a | uint16(30); //sellPoolFee

        // bytes memory sign = abi.encodeWithSignature("swap()(Trader.SwapParamsData)");
        bytes memory params = abi.encodeWithSignature(
            "swap()",
            0x5a79FDF268b93e715160967bF56BAA36C0B78fb4,
            0xbba8f85c3cEDdF73dB4de17D31608d640Eaea416,
            0x55d398326f99059fF775485246999027B3197955,
            0x8cb829111c90E0101492d5A1aa011F09614129E7,
            uint256(10000000),
            a
        );
        console2.logBytes(params);
        (bool success, bytes memory result) = address(callData).call(params);
        (Trader.SwapParamsData memory data, uint256 deadline, uint8 borrow) =
            abi.decode(result, (Trader.SwapParamsData, uint256, uint8));
        assertEq(success, true);
        assertEq(data.amount, 10000000);
        assertEq(deadline, 40725374);
        assertEq(data.buyPoolType, 1);
        assertEq(data.sellPoolType, 3);
        assertEq(data.buyPoolFee, 25);
        assertEq(data.sellPoolFee, 30);
        assertEq(borrow, 1);
    }
}

// 0x8119c065
// 00000000000000000000000011b815efb8f581194ae79006d24e0d814b7697f6
// 00000000000000000000000074c99f3f5331676f6aec2756e1f39b4fc029a83e
// 000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
// 00000000000000000000000088e6a0c2ddd26feeb64f039a2c41296fcb3f5640
// 0000000000000000000000000000000000000000000000000000000000989680
// 0000000000000000000000000000000000000001350ec10100010002001e0019
