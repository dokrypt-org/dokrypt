// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../../contracts/token/ManagedToken.sol";

contract ManagedTokenTest is Test {
    ManagedToken token;
    address owner;
    address alice;
    address bob;

    function setUp() public {
        owner = address(this);
        alice = makeAddr("alice");
        bob   = makeAddr("bob");

        token = new ManagedToken("TestToken", "TT", 1_000_000 ether);
    }

    // ---- Mint ----

    function test_mint() public {
        token.mint(alice, 500 ether);
        assertEq(token.balanceOf(alice), 500 ether);
        assertEq(token.totalSupply(), 1_000_000 ether + 500 ether);
    }

    function test_mint_onlyOwner() public {
        vm.prank(alice);
        vm.expectRevert("ManagedToken: caller is not the owner");
        token.mint(alice, 100 ether);
    }

    // ---- Burn ----

    function test_burn() public {
        token.burn(100 ether);
        assertEq(token.balanceOf(owner), 1_000_000 ether - 100 ether);
        assertEq(token.totalSupply(), 1_000_000 ether - 100 ether);
    }

    function test_burn_exceedsBalance() public {
        vm.prank(alice);
        vm.expectRevert("ManagedToken: burn amount exceeds balance");
        token.burn(1 ether);
    }

    // ---- Pause ----

    function test_pause_blocksTransfer() public {
        token.pause();
        vm.expectRevert("ManagedToken: token transfers are paused");
        token.transfer(alice, 100 ether);
    }

    function test_unpause_allowsTransfer() public {
        token.pause();
        token.unpause();
        token.transfer(alice, 100 ether);
        assertEq(token.balanceOf(alice), 100 ether);
    }

    function test_pause_onlyOwner() public {
        vm.prank(alice);
        vm.expectRevert("ManagedToken: caller is not the owner");
        token.pause();
    }

    // ---- Transfer ----

    function test_transfer() public {
        token.transfer(alice, 200 ether);
        assertEq(token.balanceOf(alice), 200 ether);
        assertEq(token.balanceOf(owner), 1_000_000 ether - 200 ether);
    }

    function test_transferFrom() public {
        token.approve(alice, 300 ether);
        vm.prank(alice);
        token.transferFrom(owner, bob, 300 ether);
        assertEq(token.balanceOf(bob), 300 ether);
    }

    // ---- Ownership ----

    function test_transferOwnership() public {
        token.transferOwnership(alice);
        assertEq(token.owner(), alice);
    }
}
