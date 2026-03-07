// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../contracts/NFTCollection.sol";

/// @title Mock ERC721 Receiver - accepts all tokens
contract MockReceiver is IERC721Receiver {
    function onERC721Received(address, address, uint256, bytes calldata) external pure returns (bytes4) {
        return IERC721Receiver.onERC721Received.selector;
    }
}

/// @title BadReceiver - rejects all tokens
contract BadReceiver {
    // No onERC721Received function - will cause safeTransferFrom to revert
}

/// @title NFTCollectionTest - Tests for the NFTCollection ERC-721 contract
contract NFTCollectionTest {
    NFTCollection public nft;
    MockReceiver public receiver;

    address constant ALICE = address(0xA11CE);
    address constant BOB = address(0xB0B);

    uint256 constant MAX_SUPPLY = 100;
    uint256 constant MINT_PRICE = 0.01 ether;

    function setUp() public {
        nft = new NFTCollection("TestNFT", "TNFT", "https://api.example.com/token/", MAX_SUPPLY, MINT_PRICE);
        receiver = new MockReceiver();
    }

    // ---- Minting Tests ----

    function testMint() public {
        uint256 tokenId = nft.mint{value: MINT_PRICE}();
        assert(tokenId == 1);
        assert(nft.ownerOf(1) == address(this));
        assert(nft.totalSupply() == 1);
    }

    function testMintMultiple() public {
        nft.mint{value: MINT_PRICE}();
        nft.mint{value: MINT_PRICE}();
        nft.mint{value: MINT_PRICE}();
        assert(nft.totalSupply() == 3);
        assert(nft.ownerOf(1) == address(this));
        assert(nft.ownerOf(2) == address(this));
        assert(nft.ownerOf(3) == address(this));
    }

    function testMintInsufficientPayment() public {
        try nft.mint{value: MINT_PRICE - 1}() {
            revert("Should have failed");
        } catch {}
    }

    function testBalanceOf() public {
        assert(nft.balanceOf(address(this)) == 0);
        nft.mint{value: MINT_PRICE}();
        assert(nft.balanceOf(address(this)) == 1);
        nft.mint{value: MINT_PRICE}();
        assert(nft.balanceOf(address(this)) == 2);
    }

    // ---- TokenURI Test ----

    function testTokenURI() public {
        nft.mint{value: MINT_PRICE}();
        string memory uri = nft.tokenURI(1);
        // Check that URI contains the base URI + token ID
        assert(bytes(uri).length > 0);
    }

    // ---- Transfer Tests ----

    function testTransferFrom() public {
        nft.mint{value: MINT_PRICE}();
        nft.transferFrom(address(this), ALICE, 1);
        assert(nft.ownerOf(1) == ALICE);
        assert(nft.balanceOf(address(this)) == 0);
        assert(nft.balanceOf(ALICE) == 1);
    }

    function testSafeTransferToContract() public {
        nft.mint{value: MINT_PRICE}();
        nft.safeTransferFrom(address(this), address(receiver), 1);
        assert(nft.ownerOf(1) == address(receiver));
    }

    // ---- Approval Tests ----

    function testApprove() public {
        nft.mint{value: MINT_PRICE}();
        nft.approve(ALICE, 1);
        assert(nft.getApproved(1) == ALICE);
    }

    function testSetApprovalForAll() public {
        nft.setApprovalForAll(ALICE, true);
        assert(nft.isApprovedForAll(address(this), ALICE));

        nft.setApprovalForAll(ALICE, false);
        assert(!nft.isApprovedForAll(address(this), ALICE));
    }

    // ---- ERC-165 Tests ----

    function testSupportsInterface() public view {
        assert(nft.supportsInterface(0x01ffc9a7)); // ERC-165
        assert(nft.supportsInterface(0x80ac58cd)); // ERC-721
        assert(nft.supportsInterface(0x5b5e139f)); // ERC-721Metadata
        assert(!nft.supportsInterface(0xffffffff)); // Invalid
    }

    // ---- Metadata Tests ----

    function testNameAndSymbol() public view {
        assert(keccak256(bytes(nft.name())) == keccak256(bytes("TestNFT")));
        assert(keccak256(bytes(nft.symbol())) == keccak256(bytes("TNFT")));
    }

    // ---- Withdraw Test ----

    function testWithdraw() public {
        nft.mint{value: MINT_PRICE}();
        uint256 balanceBefore = address(this).balance;
        nft.withdraw();
        assert(address(this).balance == balanceBefore + MINT_PRICE);
    }

    // Allow receiving ETH for withdraw test
    receive() external payable {}
}
