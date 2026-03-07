// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title PriceOracle
/// @notice A simple owner-managed price oracle. The owner sets prices for assets,
/// and consumers can read them. Includes a staleness check to ensure prices are fresh.
contract PriceOracle {
    address public owner;

    uint256 public constant MAX_STALENESS = 1 hours;

    struct PriceData {
        uint256 price;     // price in USD scaled by 1e18
        uint256 timestamp; // when the price was last updated
    }

    mapping(address => PriceData) public prices;

    // ──────────────────── Events ────────────────────
    event PriceUpdated(address indexed asset, uint256 price, uint256 timestamp);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // ──────────────────── Modifiers ────────────────────
    modifier onlyOwner() {
        require(msg.sender == owner, "PriceOracle: not owner");
        _;
    }

    constructor() {
        owner = msg.sender;
    }

    /// @notice Set the price for a given asset.
    /// @param asset Token address.
    /// @param price Price in USD scaled by 1e18 (e.g., 1e18 = $1.00).
    function setPrice(address asset, uint256 price) external onlyOwner {
        require(asset != address(0), "PriceOracle: zero address");
        require(price > 0, "PriceOracle: zero price");

        prices[asset] = PriceData({
            price: price,
            timestamp: block.timestamp
        });

        emit PriceUpdated(asset, price, block.timestamp);
    }

    /// @notice Set prices for multiple assets in a single transaction.
    /// @param assets Array of token addresses.
    /// @param _prices Array of prices scaled by 1e18.
    function setPrices(address[] calldata assets, uint256[] calldata _prices) external onlyOwner {
        require(assets.length == _prices.length, "PriceOracle: length mismatch");

        for (uint256 i = 0; i < assets.length; i++) {
            require(assets[i] != address(0), "PriceOracle: zero address");
            require(_prices[i] > 0, "PriceOracle: zero price");

            prices[assets[i]] = PriceData({
                price: _prices[i],
                timestamp: block.timestamp
            });

            emit PriceUpdated(assets[i], _prices[i], block.timestamp);
        }
    }

    /// @notice Get the price of an asset. Reverts if the price is stale or unset.
    /// @param asset Token address.
    /// @return The price in USD scaled by 1e18.
    function getPrice(address asset) external view returns (uint256) {
        PriceData memory data = prices[asset];
        require(data.price > 0, "PriceOracle: price not set");
        require(
            block.timestamp - data.timestamp <= MAX_STALENESS,
            "PriceOracle: stale price"
        );
        return data.price;
    }

    /// @notice Get the price without staleness check.
    /// @param asset Token address.
    /// @return price The price in USD scaled by 1e18.
    /// @return timestamp When the price was last updated.
    function getPriceUnsafe(address asset) external view returns (uint256 price, uint256 timestamp) {
        PriceData memory data = prices[asset];
        return (data.price, data.timestamp);
    }

    /// @notice Check if the price for an asset is fresh (not stale).
    /// @param asset Token address.
    /// @return True if the price exists and is not stale.
    function isFresh(address asset) external view returns (bool) {
        PriceData memory data = prices[asset];
        if (data.price == 0) return false;
        return (block.timestamp - data.timestamp) <= MAX_STALENESS;
    }

    /// @notice Transfer ownership of the oracle.
    /// @param newOwner New owner address.
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "PriceOracle: zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }
}
