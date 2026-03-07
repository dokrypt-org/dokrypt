// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "./Pair.sol";

/// @title Factory
/// @notice Pair factory that creates and tracks AMM liquidity pair contracts.
/// Uses CREATE2 for deterministic pair addresses.
contract Factory {
    address public owner;
    address public feeTo;

    mapping(address => mapping(address => address)) public getPair;
    address[] public allPairs;

    event PairCreated(address indexed token0, address indexed token1, address pair, uint256 pairIndex);
    event FeeToUpdated(address indexed oldFeeTo, address indexed newFeeTo);

    constructor() {
        owner = msg.sender;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "Factory: caller is not the owner");
        _;
    }

    /// @notice Returns the total number of pairs created.
    function allPairsLength() external view returns (uint256) {
        return allPairs.length;
    }

    /// @notice Creates a new liquidity pair for tokenA and tokenB.
    /// @dev Uses CREATE2 so that the pair address is deterministic based on the two token addresses.
    /// @param tokenA Address of the first token.
    /// @param tokenB Address of the second token.
    /// @return pair The address of the newly created Pair contract.
    function createPair(address tokenA, address tokenB) external returns (address pair) {
        require(tokenA != tokenB, "Factory: identical addresses");
        (address token0, address token1) = tokenA < tokenB ? (tokenA, tokenB) : (tokenB, tokenA);
        require(token0 != address(0), "Factory: zero address");
        require(getPair[token0][token1] == address(0), "Factory: pair already exists");

        // Deterministic deployment via CREATE2
        bytes32 salt = keccak256(abi.encodePacked(token0, token1));
        Pair newPair = new Pair{salt: salt}();
        newPair.initialize(token0, token1);

        pair = address(newPair);
        getPair[token0][token1] = pair;
        getPair[token1][token0] = pair;
        allPairs.push(pair);

        emit PairCreated(token0, token1, pair, allPairs.length - 1);
    }

    /// @notice Computes the expected pair address without deploying.
    /// @param tokenA Address of the first token.
    /// @param tokenB Address of the second token.
    /// @return predicted The predicted pair contract address.
    function computePairAddress(address tokenA, address tokenB) external view returns (address predicted) {
        (address token0, address token1) = tokenA < tokenB ? (tokenA, tokenB) : (tokenB, tokenA);
        bytes32 salt = keccak256(abi.encodePacked(token0, token1));
        bytes32 hash = keccak256(
            abi.encodePacked(
                bytes1(0xff),
                address(this),
                salt,
                keccak256(type(Pair).creationCode)
            )
        );
        predicted = address(uint160(uint256(hash)));
    }

    /// @notice Sets the fee recipient address.
    /// @param _feeTo New fee recipient.
    function setFeeTo(address _feeTo) external onlyOwner {
        emit FeeToUpdated(feeTo, _feeTo);
        feeTo = _feeTo;
    }

    /// @notice Transfers ownership of the factory.
    /// @param newOwner New owner address.
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Factory: zero address");
        owner = newOwner;
    }
}
