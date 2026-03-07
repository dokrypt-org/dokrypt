// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title Router
/// @notice AMM router providing high-level functions for adding/removing liquidity
/// and performing token swaps with deadline and slippage protection.
contract Router {
    address public immutable factory;

    // ──────────────────── Events ────────────────────
    event LiquidityAdded(address indexed pair, uint256 amount0, uint256 amount1, uint256 liquidity);
    event LiquidityRemoved(address indexed pair, uint256 amount0, uint256 amount1);
    event SwapExecuted(address indexed tokenIn, address indexed tokenOut, uint256 amountIn, uint256 amountOut);

    modifier ensureDeadline(uint256 deadline) {
        require(block.timestamp <= deadline, "Router: expired deadline");
        _;
    }

    constructor(address _factory) {
        require(_factory != address(0), "Router: zero factory");
        factory = _factory;
    }

    // ──────────────────── Liquidity ────────────────────

    /// @notice Add liquidity to a token pair.
    /// @param tokenA Address of the first token.
    /// @param tokenB Address of the second token.
    /// @param amountADesired Desired amount of tokenA.
    /// @param amountBDesired Desired amount of tokenB.
    /// @param amountAMin Minimum acceptable amount of tokenA.
    /// @param amountBMin Minimum acceptable amount of tokenB.
    /// @param to Recipient of the LP tokens.
    /// @param deadline Transaction deadline timestamp.
    /// @return amountA Actual amount of tokenA deposited.
    /// @return amountB Actual amount of tokenB deposited.
    /// @return liquidity Amount of LP tokens minted.
    function addLiquidity(
        address tokenA,
        address tokenB,
        uint256 amountADesired,
        uint256 amountBDesired,
        uint256 amountAMin,
        uint256 amountBMin,
        address to,
        uint256 deadline
    ) external ensureDeadline(deadline) returns (uint256 amountA, uint256 amountB, uint256 liquidity) {
        address pair = _getPair(tokenA, tokenB);
        if (pair == address(0)) {
            // Create pair if it does not exist
            pair = _createPair(tokenA, tokenB);
        }

        (uint112 reserveA, uint112 reserveB) = _getReservesSorted(pair, tokenA, tokenB);

        if (reserveA == 0 && reserveB == 0) {
            (amountA, amountB) = (amountADesired, amountBDesired);
        } else {
            uint256 amountBOptimal = _quote(amountADesired, reserveA, reserveB);
            if (amountBOptimal <= amountBDesired) {
                require(amountBOptimal >= amountBMin, "Router: insufficient B amount");
                (amountA, amountB) = (amountADesired, amountBOptimal);
            } else {
                uint256 amountAOptimal = _quote(amountBDesired, reserveB, reserveA);
                require(amountAOptimal <= amountADesired, "Router: excessive A amount");
                require(amountAOptimal >= amountAMin, "Router: insufficient A amount");
                (amountA, amountB) = (amountAOptimal, amountBDesired);
            }
        }

        _safeTransferFrom(tokenA, msg.sender, pair, amountA);
        _safeTransferFrom(tokenB, msg.sender, pair, amountB);

        // Call mint on pair
        (bool success, bytes memory data) = pair.call(abi.encodeWithSignature("mint(address)", to));
        require(success && data.length >= 32, "Router: mint failed");
        liquidity = abi.decode(data, (uint256));

        emit LiquidityAdded(pair, amountA, amountB, liquidity);
    }

    /// @notice Remove liquidity from a token pair.
    /// @param tokenA Address of the first token.
    /// @param tokenB Address of the second token.
    /// @param liquidity Amount of LP tokens to burn.
    /// @param amountAMin Minimum acceptable amount of tokenA.
    /// @param amountBMin Minimum acceptable amount of tokenB.
    /// @param to Recipient of the underlying tokens.
    /// @param deadline Transaction deadline timestamp.
    /// @return amountA Amount of tokenA received.
    /// @return amountB Amount of tokenB received.
    function removeLiquidity(
        address tokenA,
        address tokenB,
        uint256 liquidity,
        uint256 amountAMin,
        uint256 amountBMin,
        address to,
        uint256 deadline
    ) external ensureDeadline(deadline) returns (uint256 amountA, uint256 amountB) {
        address pair = _getPair(tokenA, tokenB);
        require(pair != address(0), "Router: pair does not exist");

        // Transfer LP tokens to pair
        _safeTransferFrom(pair, msg.sender, pair, liquidity);

        // Call burn on pair
        (bool success, bytes memory data) = pair.call(abi.encodeWithSignature("burn(address)", to));
        require(success && data.length >= 64, "Router: burn failed");
        (uint256 amount0, uint256 amount1) = abi.decode(data, (uint256, uint256));

        // Sort so amountA corresponds to tokenA
        (address token0, ) = _sortTokens(tokenA, tokenB);
        (amountA, amountB) = tokenA == token0 ? (amount0, amount1) : (amount1, amount0);

        require(amountA >= amountAMin, "Router: insufficient A amount");
        require(amountB >= amountBMin, "Router: insufficient B amount");

        emit LiquidityRemoved(pair, amountA, amountB);
    }

    // ──────────────────── Swap ────────────────────

    /// @notice Swap an exact amount of input tokens for as many output tokens as possible.
    /// @param amountIn Exact input amount.
    /// @param amountOutMin Minimum acceptable output amount.
    /// @param tokenIn Input token address.
    /// @param tokenOut Output token address.
    /// @param to Recipient of the output tokens.
    /// @param deadline Transaction deadline timestamp.
    /// @return amountOut Actual output amount received.
    function swapExactTokensForTokens(
        uint256 amountIn,
        uint256 amountOutMin,
        address tokenIn,
        address tokenOut,
        address to,
        uint256 deadline
    ) external ensureDeadline(deadline) returns (uint256 amountOut) {
        address pair = _getPair(tokenIn, tokenOut);
        require(pair != address(0), "Router: pair does not exist");

        (uint112 reserveIn, uint112 reserveOut) = _getReservesSorted(pair, tokenIn, tokenOut);
        amountOut = _getAmountOut(amountIn, reserveIn, reserveOut);
        require(amountOut >= amountOutMin, "Router: insufficient output amount");

        _safeTransferFrom(tokenIn, msg.sender, pair, amountIn);

        (address token0, ) = _sortTokens(tokenIn, tokenOut);
        (uint256 amount0Out, uint256 amount1Out) = tokenIn == token0
            ? (uint256(0), amountOut)
            : (amountOut, uint256(0));

        (bool success, ) = pair.call(
            abi.encodeWithSignature("swap(uint256,uint256,address)", amount0Out, amount1Out, to)
        );
        require(success, "Router: swap failed");

        emit SwapExecuted(tokenIn, tokenOut, amountIn, amountOut);
    }

    /// @notice Swap tokens to receive an exact output amount.
    /// @param amountOut Exact desired output amount.
    /// @param amountInMax Maximum acceptable input amount.
    /// @param tokenIn Input token address.
    /// @param tokenOut Output token address.
    /// @param to Recipient of the output tokens.
    /// @param deadline Transaction deadline timestamp.
    /// @return amountIn Actual input amount spent.
    function swapTokensForExactTokens(
        uint256 amountOut,
        uint256 amountInMax,
        address tokenIn,
        address tokenOut,
        address to,
        uint256 deadline
    ) external ensureDeadline(deadline) returns (uint256 amountIn) {
        address pair = _getPair(tokenIn, tokenOut);
        require(pair != address(0), "Router: pair does not exist");

        (uint112 reserveIn, uint112 reserveOut) = _getReservesSorted(pair, tokenIn, tokenOut);
        amountIn = _getAmountIn(amountOut, reserveIn, reserveOut);
        require(amountIn <= amountInMax, "Router: excessive input amount");

        _safeTransferFrom(tokenIn, msg.sender, pair, amountIn);

        (address token0, ) = _sortTokens(tokenIn, tokenOut);
        (uint256 amount0Out, uint256 amount1Out) = tokenIn == token0
            ? (uint256(0), amountOut)
            : (amountOut, uint256(0));

        (bool success, ) = pair.call(
            abi.encodeWithSignature("swap(uint256,uint256,address)", amount0Out, amount1Out, to)
        );
        require(success, "Router: swap failed");

        emit SwapExecuted(tokenIn, tokenOut, amountIn, amountOut);
    }

    // ──────────────────── View Helpers ────────────────────

    /// @notice Get the expected output amount for a given input.
    function getAmountOut(uint256 amountIn, uint256 reserveIn, uint256 reserveOut) external pure returns (uint256) {
        return _getAmountOut(amountIn, reserveIn, reserveOut);
    }

    /// @notice Get the required input amount for a given output.
    function getAmountIn(uint256 amountOut, uint256 reserveIn, uint256 reserveOut) external pure returns (uint256) {
        return _getAmountIn(amountOut, reserveIn, reserveOut);
    }

    // ──────────────────── Internal Helpers ────────────────────

    function _getAmountOut(uint256 amountIn, uint256 reserveIn, uint256 reserveOut) private pure returns (uint256) {
        require(amountIn > 0, "Router: insufficient input amount");
        require(reserveIn > 0 && reserveOut > 0, "Router: insufficient liquidity");
        uint256 amountInWithFee = amountIn * 997;
        uint256 numerator = amountInWithFee * reserveOut;
        uint256 denominator = (reserveIn * 1000) + amountInWithFee;
        return numerator / denominator;
    }

    function _getAmountIn(uint256 amountOut, uint256 reserveIn, uint256 reserveOut) private pure returns (uint256) {
        require(amountOut > 0, "Router: insufficient output amount");
        require(reserveIn > 0 && reserveOut > 0, "Router: insufficient liquidity");
        uint256 numerator = reserveIn * amountOut * 1000;
        uint256 denominator = (reserveOut - amountOut) * 997;
        return (numerator / denominator) + 1;
    }

    function _quote(uint256 amountA, uint256 reserveA, uint256 reserveB) private pure returns (uint256) {
        require(amountA > 0, "Router: insufficient amount");
        require(reserveA > 0 && reserveB > 0, "Router: insufficient liquidity");
        return (amountA * reserveB) / reserveA;
    }

    function _sortTokens(address tokenA, address tokenB) private pure returns (address token0, address token1) {
        require(tokenA != tokenB, "Router: identical addresses");
        (token0, token1) = tokenA < tokenB ? (tokenA, tokenB) : (tokenB, tokenA);
        require(token0 != address(0), "Router: zero address");
    }

    function _getPair(address tokenA, address tokenB) private view returns (address) {
        (bool success, bytes memory data) = factory.staticcall(
            abi.encodeWithSignature("getPair(address,address)", tokenA, tokenB)
        );
        if (!success || data.length < 32) return address(0);
        return abi.decode(data, (address));
    }

    function _createPair(address tokenA, address tokenB) private returns (address) {
        (bool success, bytes memory data) = factory.call(
            abi.encodeWithSignature("createPair(address,address)", tokenA, tokenB)
        );
        require(success && data.length >= 32, "Router: create pair failed");
        return abi.decode(data, (address));
    }

    function _getReservesSorted(
        address pair,
        address tokenA,
        address tokenB
    ) private view returns (uint112 reserveA, uint112 reserveB) {
        (bool success, bytes memory data) = pair.staticcall(
            abi.encodeWithSignature("getReserves()")
        );
        require(success && data.length >= 64, "Router: reserves query failed");
        (uint112 reserve0, uint112 reserve1, ) = abi.decode(data, (uint112, uint112, uint32));
        (address token0, ) = _sortTokens(tokenA, tokenB);
        (reserveA, reserveB) = tokenA == token0 ? (reserve0, reserve1) : (reserve1, reserve0);
    }

    function _safeTransferFrom(address token, address from, address to, uint256 value) private {
        (bool success, bytes memory data) = token.call(
            abi.encodeWithSignature("transferFrom(address,address,uint256)", from, to, value)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "Router: transferFrom failed");
    }
}
