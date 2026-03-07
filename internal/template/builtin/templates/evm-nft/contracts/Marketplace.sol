// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title IERC721 - Minimal interface for NFT interaction
interface IERC721 {
    function ownerOf(uint256 tokenId) external view returns (address);
    function transferFrom(address from, address to, uint256 tokenId) external;
    function getApproved(uint256 tokenId) external view returns (address);
    function isApprovedForAll(address owner, address operator) external view returns (bool);
}

/// @title Marketplace - A decentralized NFT marketplace with listings and offers
/// @notice Supports listing, buying, canceling, and making/accepting offers with a 2.5% platform fee
contract Marketplace {
    // -------------------------------------------------------------------------
    // Types
    // -------------------------------------------------------------------------

    struct Listing {
        address seller;
        uint256 price;
        bool active;
    }

    struct Offer {
        uint256 amount;
        bool active;
    }

    // -------------------------------------------------------------------------
    // Events
    // -------------------------------------------------------------------------

    event ItemListed(address indexed nft, uint256 indexed tokenId, address indexed seller, uint256 price);
    event ItemSold(address indexed nft, uint256 indexed tokenId, address indexed buyer, uint256 price);
    event ItemCanceled(address indexed nft, uint256 indexed tokenId, address indexed seller);
    event OfferMade(address indexed nft, uint256 indexed tokenId, address indexed offerer, uint256 amount);
    event OfferAccepted(address indexed nft, uint256 indexed tokenId, address indexed offerer, uint256 amount);

    // -------------------------------------------------------------------------
    // Constants
    // -------------------------------------------------------------------------

    uint256 public constant PLATFORM_FEE_BPS = 250; // 2.5% = 250 basis points
    uint256 private constant BPS_DENOMINATOR = 10000;

    // -------------------------------------------------------------------------
    // State
    // -------------------------------------------------------------------------

    address public owner;
    uint256 public platformEarnings;

    // nft address => tokenId => Listing
    mapping(address => mapping(uint256 => Listing)) public listings;

    // nft address => tokenId => offerer address => Offer
    mapping(address => mapping(uint256 => mapping(address => Offer))) public offers;

    // -------------------------------------------------------------------------
    // Modifiers
    // -------------------------------------------------------------------------

    modifier onlyOwner() {
        require(msg.sender == owner, "Marketplace: caller is not the owner");
        _;
    }

    // -------------------------------------------------------------------------
    // Constructor
    // -------------------------------------------------------------------------

    constructor() {
        owner = msg.sender;
    }

    // -------------------------------------------------------------------------
    // Listing functions
    // -------------------------------------------------------------------------

    /// @notice List an NFT for sale. Caller must be the token owner and must have approved this contract.
    function listItem(address nft, uint256 tokenId, uint256 price) external {
        require(price > 0, "Marketplace: price must be greater than zero");

        IERC721 token = IERC721(nft);
        address tokenOwner = token.ownerOf(tokenId);
        require(msg.sender == tokenOwner, "Marketplace: caller is not the token owner");
        require(
            token.getApproved(tokenId) == address(this) || token.isApprovedForAll(tokenOwner, address(this)),
            "Marketplace: marketplace not approved"
        );

        listings[nft][tokenId] = Listing({
            seller: msg.sender,
            price: price,
            active: true
        });

        emit ItemListed(nft, tokenId, msg.sender, price);
    }

    /// @notice Buy a listed NFT. Must send at least the listing price in value.
    function buyItem(address nft, uint256 tokenId) external payable {
        Listing storage listing = listings[nft][tokenId];
        require(listing.active, "Marketplace: item is not listed");
        require(msg.value >= listing.price, "Marketplace: insufficient payment");
        require(msg.sender != listing.seller, "Marketplace: seller cannot buy own item");

        listing.active = false;
        address seller = listing.seller;
        uint256 price = listing.price;

        // Calculate platform fee
        uint256 fee = (price * PLATFORM_FEE_BPS) / BPS_DENOMINATOR;
        uint256 sellerProceeds = price - fee;
        platformEarnings += fee;

        // Transfer NFT to buyer
        IERC721(nft).transferFrom(seller, msg.sender, tokenId);

        // Pay the seller
        (bool success, ) = payable(seller).call{value: sellerProceeds}("");
        require(success, "Marketplace: payment to seller failed");

        // Refund excess payment
        if (msg.value > price) {
            (bool refundSuccess, ) = payable(msg.sender).call{value: msg.value - price}("");
            require(refundSuccess, "Marketplace: refund failed");
        }

        emit ItemSold(nft, tokenId, msg.sender, price);
    }

    /// @notice Cancel an active listing. Only the seller can cancel.
    function cancelListing(address nft, uint256 tokenId) external {
        Listing storage listing = listings[nft][tokenId];
        require(listing.active, "Marketplace: item is not listed");
        require(msg.sender == listing.seller, "Marketplace: caller is not the seller");

        listing.active = false;

        emit ItemCanceled(nft, tokenId, msg.sender);
    }

    // -------------------------------------------------------------------------
    // Offer functions
    // -------------------------------------------------------------------------

    /// @notice Make an offer on an NFT. The offer amount is msg.value, held in escrow.
    function makeOffer(address nft, uint256 tokenId) external payable {
        require(msg.value > 0, "Marketplace: offer must be greater than zero");
        require(!offers[nft][tokenId][msg.sender].active, "Marketplace: active offer already exists");

        // Verify the token exists by calling ownerOf (will revert for nonexistent tokens)
        IERC721(nft).ownerOf(tokenId);

        offers[nft][tokenId][msg.sender] = Offer({
            amount: msg.value,
            active: true
        });

        emit OfferMade(nft, tokenId, msg.sender, msg.value);
    }

    /// @notice Accept an offer on an NFT you own. Transfers the NFT and pays the seller.
    function acceptOffer(address nft, uint256 tokenId, address offerer) external {
        Offer storage offer = offers[nft][tokenId][offerer];
        require(offer.active, "Marketplace: no active offer from this address");

        IERC721 token = IERC721(nft);
        address tokenOwner = token.ownerOf(tokenId);
        require(msg.sender == tokenOwner, "Marketplace: caller is not the token owner");
        require(
            token.getApproved(tokenId) == address(this) || token.isApprovedForAll(tokenOwner, address(this)),
            "Marketplace: marketplace not approved"
        );

        uint256 amount = offer.amount;
        offer.active = false;

        // Deactivate listing if one exists
        if (listings[nft][tokenId].active) {
            listings[nft][tokenId].active = false;
        }

        // Calculate platform fee
        uint256 fee = (amount * PLATFORM_FEE_BPS) / BPS_DENOMINATOR;
        uint256 sellerProceeds = amount - fee;
        platformEarnings += fee;

        // Transfer NFT to offerer
        token.transferFrom(tokenOwner, offerer, tokenId);

        // Pay the seller
        (bool success, ) = payable(tokenOwner).call{value: sellerProceeds}("");
        require(success, "Marketplace: payment to seller failed");

        emit OfferAccepted(nft, tokenId, offerer, amount);
    }

    /// @notice Withdraw an active offer and reclaim escrowed funds.
    function withdrawOffer(address nft, uint256 tokenId) external {
        Offer storage offer = offers[nft][tokenId][msg.sender];
        require(offer.active, "Marketplace: no active offer");

        uint256 amount = offer.amount;
        offer.active = false;

        (bool success, ) = payable(msg.sender).call{value: amount}("");
        require(success, "Marketplace: refund failed");
    }

    // -------------------------------------------------------------------------
    // Owner functions
    // -------------------------------------------------------------------------

    /// @notice Withdraw accumulated platform fees.
    function withdrawFees() external onlyOwner {
        uint256 amount = platformEarnings;
        require(amount > 0, "Marketplace: no fees to withdraw");
        platformEarnings = 0;

        (bool success, ) = payable(owner).call{value: amount}("");
        require(success, "Marketplace: withdrawal failed");
    }

    /// @notice Get listing details for a given NFT and token ID.
    function getListing(address nft, uint256 tokenId) external view returns (address seller, uint256 price, bool active) {
        Listing storage listing = listings[nft][tokenId];
        return (listing.seller, listing.price, listing.active);
    }

    /// @notice Get offer details for a given NFT, token ID, and offerer.
    function getOffer(address nft, uint256 tokenId, address offerer) external view returns (uint256 amount, bool active) {
        Offer storage offer = offers[nft][tokenId][offerer];
        return (offer.amount, offer.active);
    }
}
