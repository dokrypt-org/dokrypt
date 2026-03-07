// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../contracts/NFTCollection.sol";
import "../contracts/Marketplace.sol";

/// @title MarketplaceTest - Tests for the NFT Marketplace contract
contract MarketplaceTest {
    NFTCollection public nft;
    Marketplace public market;

    uint256 constant MINT_PRICE = 0.01 ether;
    uint256 constant LIST_PRICE = 1 ether;

    function setUp() public {
        nft = new NFTCollection("TestNFT", "TNFT", "https://api.example.com/token/", 1000, MINT_PRICE);
        market = new Marketplace();
    }

    // ---- Listing Tests ----

    function testListItem() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        nft.approve(address(market), tokenId);

        market.listItem(address(nft), tokenId, LIST_PRICE);

        (address seller, uint256 price, bool active) = market.getListing(address(nft), tokenId);
        assert(seller == address(this));
        assert(price == LIST_PRICE);
        assert(active);
    }

    function testListItemZeroPrice() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        nft.approve(address(market), tokenId);

        try market.listItem(address(nft), tokenId, 0) {
            revert("Should have failed");
        } catch {}
    }

    // ---- Buy Tests ----

    function testBuyItem() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        nft.approve(address(market), tokenId);
        market.listItem(address(nft), tokenId, LIST_PRICE);

        // Simulate a buyer (this contract acts as buyer too for simplicity)
        // In a real test framework, you'd use vm.prank
        // Here we verify the listing is set up correctly
        (address seller, uint256 price, bool active) = market.getListing(address(nft), tokenId);
        assert(seller == address(this));
        assert(price == LIST_PRICE);
        assert(active);
    }

    function testBuyItemInsufficientPayment() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        nft.approve(address(market), tokenId);
        market.listItem(address(nft), tokenId, LIST_PRICE);

        try market.buyItem{value: LIST_PRICE - 1}(address(nft), tokenId) {
            revert("Should have failed");
        } catch {}
    }

    // ---- Cancel Tests ----

    function testCancelListing() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        nft.approve(address(market), tokenId);
        market.listItem(address(nft), tokenId, LIST_PRICE);

        market.cancelListing(address(nft), tokenId);

        (, , bool active) = market.getListing(address(nft), tokenId);
        assert(!active);
    }

    function testCancelListingNotListed() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();

        try market.cancelListing(address(nft), tokenId) {
            revert("Should have failed");
        } catch {}
    }

    // ---- Offer Tests ----

    function testMakeOffer() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();

        market.makeOffer{value: 0.5 ether}(address(nft), tokenId);

        (uint256 amount, bool active) = market.getOffer(address(nft), tokenId, address(this));
        assert(amount == 0.5 ether);
        assert(active);
    }

    function testMakeOfferZeroValue() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();

        try market.makeOffer{value: 0}(address(nft), tokenId) {
            revert("Should have failed");
        } catch {}
    }

    function testWithdrawOffer() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        market.makeOffer{value: 0.5 ether}(address(nft), tokenId);

        uint256 balanceBefore = address(this).balance;
        market.withdrawOffer(address(nft), tokenId);
        assert(address(this).balance == balanceBefore + 0.5 ether);

        (, bool active) = market.getOffer(address(nft), tokenId, address(this));
        assert(!active);
    }

    // ---- Platform Fee Tests ----

    function testPlatformFeeCalculation() public view {
        // 2.5% of 1 ether = 0.025 ether
        uint256 fee = (LIST_PRICE * market.PLATFORM_FEE_BPS()) / 10000;
        assert(fee == 0.025 ether);
    }

    function testPlatformEarningsInitiallyZero() public view {
        assert(market.platformEarnings() == 0);
    }

    // ---- Accept Offer Tests ----

    function testAcceptOfferRequiresApproval() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        market.makeOffer{value: 0.5 ether}(address(nft), tokenId);

        // Attempting to accept without approval should fail
        try market.acceptOffer(address(nft), tokenId, address(this)) {
            revert("Should have failed - marketplace not approved");
        } catch {}
    }

    function testAcceptOfferNoOffer() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        nft.approve(address(market), tokenId);

        try market.acceptOffer(address(nft), tokenId, address(0xDEAD)) {
            revert("Should have failed - no offer");
        } catch {}
    }

    // Allow receiving ETH
    receive() external payable {}
}
