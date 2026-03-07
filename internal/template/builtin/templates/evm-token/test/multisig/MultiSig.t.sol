// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../../contracts/multisig/MultiSig.sol";

contract MultiSigTest is Test {
    MultiSig wallet;

    address owner1;
    address owner2;
    address owner3;
    address nonOwner;
    address payable recipient;

    function setUp() public {
        owner1   = makeAddr("owner1");
        owner2   = makeAddr("owner2");
        owner3   = makeAddr("owner3");
        nonOwner = makeAddr("nonOwner");
        recipient = payable(makeAddr("recipient"));

        address[] memory owners = new address[](3);
        owners[0] = owner1;
        owners[1] = owner2;
        owners[2] = owner3;

        wallet = new MultiSig(owners, 2); // 2-of-3

        // Fund the wallet with ETH
        vm.deal(address(wallet), 10 ether);
    }

    // ---- Submit ----

    function test_submitTransaction() public {
        vm.prank(owner1);
        uint256 txId = wallet.submitTransaction(recipient, 1 ether, "");
        assertEq(txId, 0);
        assertEq(wallet.getTransactionCount(), 1);
    }

    function test_submit_onlyOwner() public {
        vm.prank(nonOwner);
        vm.expectRevert("MultiSig: caller is not an owner");
        wallet.submitTransaction(recipient, 1 ether, "");
    }

    // ---- Confirm ----

    function test_confirmTransaction() public {
        vm.prank(owner1);
        uint256 txId = wallet.submitTransaction(recipient, 1 ether, "");

        vm.prank(owner1);
        wallet.confirmTransaction(txId);
        assertEq(wallet.getConfirmationCount(txId), 1);

        vm.prank(owner2);
        wallet.confirmTransaction(txId);
        assertEq(wallet.getConfirmationCount(txId), 2);
    }

    function test_doubleConfirm_reverts() public {
        vm.prank(owner1);
        uint256 txId = wallet.submitTransaction(recipient, 1 ether, "");

        vm.prank(owner1);
        wallet.confirmTransaction(txId);

        vm.prank(owner1);
        vm.expectRevert("MultiSig: transaction already confirmed by caller");
        wallet.confirmTransaction(txId);
    }

    // ---- Execute ----

    function test_executeTransaction() public {
        vm.prank(owner1);
        uint256 txId = wallet.submitTransaction(recipient, 1 ether, "");

        vm.prank(owner1);
        wallet.confirmTransaction(txId);

        vm.prank(owner2);
        wallet.confirmTransaction(txId);

        uint256 balBefore = recipient.balance;

        vm.prank(owner1);
        wallet.executeTransaction(txId);

        assertEq(recipient.balance, balBefore + 1 ether);
    }

    function test_execute_notEnoughConfirmations() public {
        vm.prank(owner1);
        uint256 txId = wallet.submitTransaction(recipient, 1 ether, "");

        vm.prank(owner1);
        wallet.confirmTransaction(txId);

        // Only 1 of 2 required confirmations
        vm.prank(owner1);
        vm.expectRevert("MultiSig: not enough confirmations");
        wallet.executeTransaction(txId);
    }

    // ---- Revoke ----

    function test_revokeConfirmation() public {
        vm.prank(owner1);
        uint256 txId = wallet.submitTransaction(recipient, 1 ether, "");

        vm.prank(owner1);
        wallet.confirmTransaction(txId);
        assertEq(wallet.getConfirmationCount(txId), 1);

        vm.prank(owner1);
        wallet.revokeConfirmation(txId);
        assertEq(wallet.getConfirmationCount(txId), 0);
    }

    function test_revoke_notConfirmed() public {
        vm.prank(owner1);
        uint256 txId = wallet.submitTransaction(recipient, 1 ether, "");

        vm.prank(owner2);
        vm.expectRevert("MultiSig: transaction not confirmed by caller");
        wallet.revokeConfirmation(txId);
    }

    // ---- M-of-N Threshold ----

    function test_threeOfThree() public {
        address[] memory owners = new address[](3);
        owners[0] = owner1;
        owners[1] = owner2;
        owners[2] = owner3;

        MultiSig strict = new MultiSig(owners, 3); // 3-of-3
        vm.deal(address(strict), 5 ether);

        vm.prank(owner1);
        uint256 txId = strict.submitTransaction(recipient, 1 ether, "");

        vm.prank(owner1);
        strict.confirmTransaction(txId);

        vm.prank(owner2);
        strict.confirmTransaction(txId);

        // Still not enough (2 of 3)
        vm.prank(owner1);
        vm.expectRevert("MultiSig: not enough confirmations");
        strict.executeTransaction(txId);

        vm.prank(owner3);
        strict.confirmTransaction(txId);

        uint256 balBefore = recipient.balance;
        vm.prank(owner1);
        strict.executeTransaction(txId);
        assertEq(recipient.balance, balBefore + 1 ether);
    }

    // ---- Receive ETH ----

    function test_receiveETH() public {
        uint256 balBefore = address(wallet).balance;
        vm.deal(address(this), 5 ether);
        (bool ok, ) = address(wallet).call{value: 2 ether}("");
        assertTrue(ok);
        assertEq(address(wallet).balance, balBefore + 2 ether);
    }
}
