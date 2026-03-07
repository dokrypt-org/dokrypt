// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title Pair
/// @notice AMM liquidity pair implementing the constant-product formula (x * y = k).
/// Includes a built-in ERC-20 LP token, reentrancy protection, and minimum liquidity lock.
contract Pair {
    // ──────────────────── ERC-20 LP Token ────────────────────
    string public constant name = "DeFi LP Token";
    string public constant symbol = "DLP";
    uint8 public constant decimals = 18;
    uint256 public totalSupply;

    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    // ──────────────────── AMM State ────────────────────
    address public factory;
    address public token0;
    address public token1;

    uint112 private reserve0;
    uint112 private reserve1;
    uint32 private blockTimestampLast;

    uint256 public price0CumulativeLast;
    uint256 public price1CumulativeLast;
    uint256 public kLast; // reserve0 * reserve1 at last liquidity event

    uint256 public constant MINIMUM_LIQUIDITY = 1000;

    // Reentrancy guard
    uint256 private _unlocked = 1;

    event Mint(address indexed sender, uint256 amount0, uint256 amount1);
    event Burn(address indexed sender, uint256 amount0, uint256 amount1, address indexed to);
    event Swap(
        address indexed sender,
        uint256 amount0In,
        uint256 amount1In,
        uint256 amount0Out,
        uint256 amount1Out,
        address indexed to
    );
    event Sync(uint112 reserve0, uint112 reserve1);

    modifier nonReentrant() {
        require(_unlocked == 1, "Pair: reentrancy");
        _unlocked = 0;
        _;
        _unlocked = 1;
    }

    constructor() {
        factory = msg.sender;
    }

    /// @notice Called once by the factory at deployment time.
    function initialize(address _token0, address _token1) external {
        require(msg.sender == factory, "Pair: forbidden");
        token0 = _token0;
        token1 = _token1;
    }

    // ──────────────────── ERC-20 Functions ────────────────────

    function approve(address spender, uint256 value) external returns (bool) {
        _approve(msg.sender, spender, value);
        return true;
    }

    function transfer(address to, uint256 value) external returns (bool) {
        _transfer(msg.sender, to, value);
        return true;
    }

    function transferFrom(address from, address to, uint256 value) external returns (bool) {
        uint256 currentAllowance = allowance[from][msg.sender];
        if (currentAllowance != type(uint256).max) {
            require(currentAllowance >= value, "Pair: insufficient allowance");
            unchecked {
                _approve(from, msg.sender, currentAllowance - value);
            }
        }
        _transfer(from, to, value);
        return true;
    }

    // ──────────────────── AMM Functions ────────────────────

    function getReserves() external view returns (uint112 _reserve0, uint112 _reserve1, uint32 _blockTimestampLast) {
        _reserve0 = reserve0;
        _reserve1 = reserve1;
        _blockTimestampLast = blockTimestampLast;
    }

    /// @notice Add liquidity. Caller must transfer token0 and token1 to this contract before calling.
    /// @param to Recipient of the minted LP tokens.
    /// @return liquidity Amount of LP tokens minted.
    function mint(address to) external nonReentrant returns (uint256 liquidity) {
        (uint112 _reserve0, uint112 _reserve1, ) = (reserve0, reserve1, blockTimestampLast);
        uint256 balance0 = _tokenBalance(token0);
        uint256 balance1 = _tokenBalance(token1);
        uint256 amount0 = balance0 - _reserve0;
        uint256 amount1 = balance1 - _reserve1;

        if (totalSupply == 0) {
            liquidity = _sqrt(amount0 * amount1) - MINIMUM_LIQUIDITY;
            _mintLP(address(0), MINIMUM_LIQUIDITY); // permanently lock minimum liquidity
        } else {
            liquidity = _min(
                (amount0 * totalSupply) / _reserve0,
                (amount1 * totalSupply) / _reserve1
            );
        }

        require(liquidity > 0, "Pair: insufficient liquidity minted");
        _mintLP(to, liquidity);

        _update(balance0, balance1, _reserve0, _reserve1);
        kLast = uint256(reserve0) * reserve1;

        emit Mint(msg.sender, amount0, amount1);
    }

    /// @notice Remove liquidity. Caller must transfer LP tokens to this contract before calling.
    /// @param to Recipient of the underlying tokens.
    /// @return amount0 Amount of token0 returned.
    /// @return amount1 Amount of token1 returned.
    function burn(address to) external nonReentrant returns (uint256 amount0, uint256 amount1) {
        uint256 balance0 = _tokenBalance(token0);
        uint256 balance1 = _tokenBalance(token1);
        uint256 liquidity = balanceOf[address(this)];

        amount0 = (liquidity * balance0) / totalSupply;
        amount1 = (liquidity * balance1) / totalSupply;
        require(amount0 > 0 && amount1 > 0, "Pair: insufficient liquidity burned");

        _burnLP(address(this), liquidity);
        _safeTransfer(token0, to, amount0);
        _safeTransfer(token1, to, amount1);

        balance0 = _tokenBalance(token0);
        balance1 = _tokenBalance(token1);

        _update(balance0, balance1, reserve0, reserve1);
        kLast = uint256(reserve0) * reserve1;

        emit Burn(msg.sender, amount0, amount1, to);
    }

    /// @notice Swap tokens. Caller must transfer input tokens before calling.
    /// @param amount0Out Desired amount of token0 to receive.
    /// @param amount1Out Desired amount of token1 to receive.
    /// @param to Recipient of the output tokens.
    function swap(uint256 amount0Out, uint256 amount1Out, address to) external nonReentrant {
        require(amount0Out > 0 || amount1Out > 0, "Pair: insufficient output amount");
        (uint112 _reserve0, uint112 _reserve1, ) = (reserve0, reserve1, blockTimestampLast);
        require(amount0Out < _reserve0 && amount1Out < _reserve1, "Pair: insufficient liquidity");

        require(to != token0 && to != token1, "Pair: invalid to");

        if (amount0Out > 0) _safeTransfer(token0, to, amount0Out);
        if (amount1Out > 0) _safeTransfer(token1, to, amount1Out);

        uint256 balance0 = _tokenBalance(token0);
        uint256 balance1 = _tokenBalance(token1);

        uint256 amount0In = balance0 > _reserve0 - amount0Out ? balance0 - (_reserve0 - amount0Out) : 0;
        uint256 amount1In = balance1 > _reserve1 - amount1Out ? balance1 - (_reserve1 - amount1Out) : 0;
        require(amount0In > 0 || amount1In > 0, "Pair: insufficient input amount");

        // Enforce invariant with 0.3% fee
        uint256 balance0Adjusted = (balance0 * 1000) - (amount0In * 3);
        uint256 balance1Adjusted = (balance1 * 1000) - (amount1In * 3);
        require(
            balance0Adjusted * balance1Adjusted >= uint256(_reserve0) * uint256(_reserve1) * 1_000_000,
            "Pair: K invariant"
        );

        _update(balance0, balance1, _reserve0, _reserve1);

        emit Swap(msg.sender, amount0In, amount1In, amount0Out, amount1Out, to);
    }

    /// @notice Force reserves to match balances.
    function sync() external nonReentrant {
        _update(_tokenBalance(token0), _tokenBalance(token1), reserve0, reserve1);
    }

    // ──────────────────── Internal Helpers ────────────────────

    function _update(uint256 balance0, uint256 balance1, uint112 _reserve0, uint112 _reserve1) private {
        require(balance0 <= type(uint112).max && balance1 <= type(uint112).max, "Pair: overflow");

        uint32 blockTimestamp = uint32(block.timestamp % 2 ** 32);
        unchecked {
            uint32 timeElapsed = blockTimestamp - blockTimestampLast;
            if (timeElapsed > 0 && _reserve0 != 0 && _reserve1 != 0) {
                price0CumulativeLast += uint256((uint224(_reserve1) << 112) / _reserve0) * timeElapsed;
                price1CumulativeLast += uint256((uint224(_reserve0) << 112) / _reserve1) * timeElapsed;
            }
        }

        reserve0 = uint112(balance0);
        reserve1 = uint112(balance1);
        blockTimestampLast = blockTimestamp;

        emit Sync(reserve0, reserve1);
    }

    function _tokenBalance(address token) private view returns (uint256) {
        (bool success, bytes memory data) = token.staticcall(
            abi.encodeWithSignature("balanceOf(address)", address(this))
        );
        require(success && data.length >= 32, "Pair: balance query failed");
        return abi.decode(data, (uint256));
    }

    function _safeTransfer(address token, address to, uint256 value) private {
        (bool success, bytes memory data) = token.call(
            abi.encodeWithSignature("transfer(address,uint256)", to, value)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "Pair: transfer failed");
    }

    function _approve(address _owner, address spender, uint256 value) private {
        allowance[_owner][spender] = value;
        emit Approval(_owner, spender, value);
    }

    function _transfer(address from, address to, uint256 value) private {
        require(balanceOf[from] >= value, "Pair: insufficient balance");
        unchecked {
            balanceOf[from] -= value;
            balanceOf[to] += value;
        }
        emit Transfer(from, to, value);
    }

    function _mintLP(address to, uint256 value) private {
        totalSupply += value;
        balanceOf[to] += value;
        emit Transfer(address(0), to, value);
    }

    function _burnLP(address from, uint256 value) private {
        require(balanceOf[from] >= value, "Pair: burn exceeds balance");
        unchecked {
            balanceOf[from] -= value;
        }
        totalSupply -= value;
        emit Transfer(from, address(0), value);
    }

    function _sqrt(uint256 y) private pure returns (uint256 z) {
        if (y > 3) {
            z = y;
            uint256 x = y / 2 + 1;
            while (x < z) {
                z = x;
                x = (y / x + x) / 2;
            }
        } else if (y != 0) {
            z = 1;
        }
    }

    function _min(uint256 a, uint256 b) private pure returns (uint256) {
        return a < b ? a : b;
    }
}
