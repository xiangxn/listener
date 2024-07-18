// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Test, console2} from "forge-std/Test.sol";
import {PoolAddress} from "../src/PoolAddress.sol";

contract PoolAddressTest is Test {
    function test_PoolAddress() public pure {
        address poolAddr = 0x11b815efB8f581194ae79006d24E0d814B7697F6;
        PoolAddress.PoolKey memory key = PoolAddress.getPoolKey(
            0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2, 0xdAC17F958D2ee523a2206206994597C13D831ec7, 500
        );
        address calcAddr = PoolAddress.computeAddress(0x1F98431c8aD98523631AE4a59f267346ea31F984, key);
        assertEq(poolAddr, calcAddr);
    }
}
