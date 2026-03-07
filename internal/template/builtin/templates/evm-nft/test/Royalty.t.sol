// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../contracts/Royalty.sol";

/// @title RoyaltyTest - Tests for the EIP-2981 Royalty contract
contract RoyaltyTest {
    Royalty public royalty;

    address constant RECEIVER = address(0xABCD);
    uint96 constant DEFAULT_FEE = 500; // 5%

    function setUp() public {
        royalty = new Royalty(RECEIVER, DEFAULT_FEE);
    }

    // ---- Default Royalty Tests ----

    function testDefaultRoyalty() public view {
        (address receiver, uint256 amount) = royalty.royaltyInfo(1, 1 ether);
        assert(receiver == RECEIVER);
        assert(amount == 0.05 ether); // 5% of 1 ether
    }

    function testDefaultRoyaltyDifferentPrices() public view {
        (, uint256 amount1) = royalty.royaltyInfo(1, 10 ether);
        assert(amount1 == 0.5 ether); // 5% of 10 ether

        (, uint256 amount2) = royalty.royaltyInfo(1, 100);
        assert(amount2 == 5); // 5% of 100
    }

    function testDefaultRoyaltyZeroPrice() public view {
        (, uint256 amount) = royalty.royaltyInfo(1, 0);
        assert(amount == 0);
    }

    // ---- Per-Token Override Tests ----

    function testSetTokenRoyalty() public {
        address newReceiver = address(0x1234);
        uint96 newFee = 250; // 2.5%

        royalty.setTokenRoyalty(42, newReceiver, newFee);

        (address receiver, uint256 amount) = royalty.royaltyInfo(42, 1 ether);
        assert(receiver == newReceiver);
        assert(amount == 0.025 ether); // 2.5% of 1 ether
    }

    function testTokenRoyaltyDoesNotAffectOthers() public {
        royalty.setTokenRoyalty(42, address(0x1234), 250);

        // Token 1 should still use default royalty
        (address receiver, uint256 amount) = royalty.royaltyInfo(1, 1 ether);
        assert(receiver == RECEIVER);
        assert(amount == 0.05 ether);
    }

    function testResetTokenRoyalty() public {
        royalty.setTokenRoyalty(42, address(0x1234), 250);
        royalty.resetTokenRoyalty(42);

        // Should fall back to default
        (address receiver, uint256 amount) = royalty.royaltyInfo(42, 1 ether);
        assert(receiver == RECEIVER);
        assert(amount == 0.05 ether);
    }

    // ---- Update Default Royalty Tests ----

    function testSetDefaultRoyalty() public {
        address newReceiver = address(0x5678);
        uint96 newFee = 300; // 3%

        royalty.setDefaultRoyalty(newReceiver, newFee);

        (address receiver, uint256 amount) = royalty.royaltyInfo(1, 1 ether);
        assert(receiver == newReceiver);
        assert(amount == 0.03 ether);
    }

    function testExceedMaxRoyalty() public {
        try royalty.setDefaultRoyalty(RECEIVER, 1001) {
            revert("Should have failed - exceeds max");
        } catch {}
    }

    // ---- ERC-165 Tests ----

    function testSupportsInterface() public view {
        assert(royalty.supportsInterface(0x01ffc9a7)); // ERC-165
        assert(royalty.supportsInterface(0x2a55205a)); // ERC-2981
        assert(!royalty.supportsInterface(0xffffffff)); // Invalid
    }
}
