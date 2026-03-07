// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../../contracts/token/DeFiToken.sol";
import "../../contracts/amm/Factory.sol";
import "../../contracts/amm/Router.sol";
import "../../contracts/amm/Pair.sol";

/// @title RouterTest
/// @notice Foundry-style tests for the AMM Router: add liquidity, swap, remove liquidity.
contract RouterTest {
    DeFiToken tokenA;
    DeFiToken tokenB;
    Factory factory;
    Router router;

    address deployer;

    function setUp() public {
        deployer = address(this);

        tokenA = new DeFiToken("Token A", "TKA", 1_000_000e18);
        tokenB = new DeFiToken("Token B", "TKB", 1_000_000e18);

        factory = new Factory();
        router = new Router(address(factory));

        // Approve router to spend tokens
        tokenA.approve(address(router), type(uint256).max);
        tokenB.approve(address(router), type(uint256).max);
    }

    function testAddLiquidity() public {
        uint256 amountA = 100_000e18;
        uint256 amountB = 200_000e18;

        (uint256 actualA, uint256 actualB, uint256 liquidity) = router.addLiquidity(
            address(tokenA),
            address(tokenB),
            amountA,
            amountB,
            0, // amountAMin
            0, // amountBMin
            deployer,
            block.timestamp + 3600
        );

        require(actualA == amountA, "Wrong amountA deposited");
        require(actualB == amountB, "Wrong amountB deposited");
        require(liquidity > 0, "No liquidity minted");

        // Verify pair was created
        address pair = factory.getPair(address(tokenA), address(tokenB));
        require(pair != address(0), "Pair not created");
    }

    function testSwapExactTokensForTokens() public {
        // First add liquidity
        router.addLiquidity(
            address(tokenA),
            address(tokenB),
            100_000e18,
            100_000e18,
            0,
            0,
            deployer,
            block.timestamp + 3600
        );

        uint256 swapAmount = 1_000e18;
        uint256 balanceBBefore = tokenB.balanceOf(deployer);

        uint256 amountOut = router.swapExactTokensForTokens(
            swapAmount,
            0, // amountOutMin
            address(tokenA),
            address(tokenB),
            deployer,
            block.timestamp + 3600
        );

        uint256 balanceBAfter = tokenB.balanceOf(deployer);

        require(amountOut > 0, "No output from swap");
        require(balanceBAfter - balanceBBefore == amountOut, "Balance mismatch after swap");
        // With 0.3% fee: amountOut should be less than swapAmount (constant product)
        require(amountOut < swapAmount, "Output should be less than input due to price impact");
    }

    function testSwapTokensForExactTokens() public {
        // Add liquidity
        router.addLiquidity(
            address(tokenA),
            address(tokenB),
            100_000e18,
            100_000e18,
            0,
            0,
            deployer,
            block.timestamp + 3600
        );

        uint256 desiredOut = 500e18;
        uint256 balanceABefore = tokenA.balanceOf(deployer);

        uint256 amountIn = router.swapTokensForExactTokens(
            desiredOut,
            type(uint256).max, // amountInMax
            address(tokenA),
            address(tokenB),
            deployer,
            block.timestamp + 3600
        );

        uint256 balanceAAfter = tokenA.balanceOf(deployer);

        require(amountIn > 0, "No input consumed");
        require(balanceABefore - balanceAAfter == amountIn, "Input balance mismatch");
    }

    function testRemoveLiquidity() public {
        // Add liquidity
        (, , uint256 liquidity) = router.addLiquidity(
            address(tokenA),
            address(tokenB),
            100_000e18,
            100_000e18,
            0,
            0,
            deployer,
            block.timestamp + 3600
        );

        address pair = factory.getPair(address(tokenA), address(tokenB));

        // Approve router to spend LP tokens
        Pair(pair).approve(address(router), liquidity);

        uint256 balanceABefore = tokenA.balanceOf(deployer);
        uint256 balanceBBefore = tokenB.balanceOf(deployer);

        (uint256 amountA, uint256 amountB) = router.removeLiquidity(
            address(tokenA),
            address(tokenB),
            liquidity,
            0, // amountAMin
            0, // amountBMin
            deployer,
            block.timestamp + 3600
        );

        require(amountA > 0, "No tokenA returned");
        require(amountB > 0, "No tokenB returned");

        uint256 balanceAAfter = tokenA.balanceOf(deployer);
        uint256 balanceBAfter = tokenB.balanceOf(deployer);

        require(balanceAAfter - balanceABefore == amountA, "TokenA balance mismatch");
        require(balanceBAfter - balanceBBefore == amountB, "TokenB balance mismatch");
    }

    function testDeadlineExpired() public {
        // Should revert with an expired deadline
        bool reverted = false;
        try router.addLiquidity(
            address(tokenA),
            address(tokenB),
            1000e18,
            1000e18,
            0,
            0,
            deployer,
            block.timestamp - 1 // expired
        ) {
            // should not reach here
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on expired deadline");
    }
}
