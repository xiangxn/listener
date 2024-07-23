// SPDX-License-Identifier: GPL-2.0-or-later
pragma solidity ^0.8.13;

import "./AlgebraPA.sol";

/// @notice Provides validation for callbacks from Algebra Pools
/// @dev Credit to Uniswap Labs under GPL-2.0-or-later license:
/// https://github.com/Uniswap/v3-periphery
library AlgebraCV {
    /// @notice Returns the address of a valid Algebra Pool
    /// @param poolDeployer The contract address of the Algebra pool deployer
    /// @param tokenA The contract address of either token0 or token1
    /// @param tokenB The contract address of the other token
    function verifyCallback(address poolDeployer, address tokenA, address tokenB) internal view {
        verifyCallback(poolDeployer, PoolAddress.getPoolKey(tokenA, tokenB));
    }

    /// @notice Returns the address of a valid Algebra Pool
    /// @param poolDeployer The contract address of the Algebra pool deployer
    /// @param poolKey The identifying key of the V3 pool
    function verifyCallback(address poolDeployer, PoolAddress.PoolKey memory poolKey) internal view {
        address pool = PoolAddress.computeAddress(poolDeployer, poolKey);
        require(msg.sender == pool);
    }
}
