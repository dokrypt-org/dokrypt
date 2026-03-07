// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title IERC165 - Standard interface detection
interface IERC165 {
    function supportsInterface(bytes4 interfaceId) external view returns (bool);
}

/// @title IERC2981 - NFT Royalty Standard
interface IERC2981 is IERC165 {
    function royaltyInfo(uint256 tokenId, uint256 salePrice)
        external
        view
        returns (address receiver, uint256 royaltyAmount);
}

/// @title Royalty - EIP-2981 royalty implementation with per-token overrides
/// @notice Provides a default royalty that can be overridden on a per-token basis
contract Royalty is IERC2981 {
    struct RoyaltyData {
        address receiver;
        uint96 feeBps; // basis points (e.g., 500 = 5%)
    }

    address public owner;
    RoyaltyData private _defaultRoyalty;
    mapping(uint256 => RoyaltyData) private _tokenRoyalties;

    uint96 public constant MAX_ROYALTY_BPS = 1000; // 10% max

    event DefaultRoyaltySet(address indexed receiver, uint96 feeBps);
    event TokenRoyaltySet(uint256 indexed tokenId, address indexed receiver, uint96 feeBps);

    modifier onlyOwner() {
        require(msg.sender == owner, "Royalty: caller is not the owner");
        _;
    }

    constructor(address receiver, uint96 feeBps) {
        require(feeBps <= MAX_ROYALTY_BPS, "Royalty: fee exceeds maximum");
        require(receiver != address(0), "Royalty: receiver is zero address");
        owner = msg.sender;
        _defaultRoyalty = RoyaltyData(receiver, feeBps);
        emit DefaultRoyaltySet(receiver, feeBps);
    }

    /// @notice ERC-165 interface support
    function supportsInterface(bytes4 interfaceId) external pure override returns (bool) {
        return
            interfaceId == 0x01ffc9a7 || // ERC-165
            interfaceId == 0x2a55205a;    // ERC-2981
    }

    /// @notice Returns royalty info for a given token and sale price
    function royaltyInfo(uint256 tokenId, uint256 salePrice)
        external
        view
        override
        returns (address receiver, uint256 royaltyAmount)
    {
        RoyaltyData memory royalty = _tokenRoyalties[tokenId];

        // Fall back to default if no per-token royalty is set
        if (royalty.receiver == address(0)) {
            royalty = _defaultRoyalty;
        }

        uint256 amount = (salePrice * royalty.feeBps) / 10000;
        return (royalty.receiver, amount);
    }

    /// @notice Set the default royalty for all tokens
    function setDefaultRoyalty(address receiver, uint96 feeBps) external onlyOwner {
        require(feeBps <= MAX_ROYALTY_BPS, "Royalty: fee exceeds maximum");
        require(receiver != address(0), "Royalty: receiver is zero address");
        _defaultRoyalty = RoyaltyData(receiver, feeBps);
        emit DefaultRoyaltySet(receiver, feeBps);
    }

    /// @notice Set a per-token royalty override
    function setTokenRoyalty(uint256 tokenId, address receiver, uint96 feeBps) external onlyOwner {
        require(feeBps <= MAX_ROYALTY_BPS, "Royalty: fee exceeds maximum");
        require(receiver != address(0), "Royalty: receiver is zero address");
        _tokenRoyalties[tokenId] = RoyaltyData(receiver, feeBps);
        emit TokenRoyaltySet(tokenId, receiver, feeBps);
    }

    /// @notice Remove a per-token royalty override (reverts to default)
    function resetTokenRoyalty(uint256 tokenId) external onlyOwner {
        delete _tokenRoyalties[tokenId];
    }
}
