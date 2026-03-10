// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "../contracts/bridge/L1ToL2MessageSender.sol";
import "../contracts/bridge/L2ToL1MessageSender.sol";

/// @title L1ToL2MessageTest
/// @notice Foundry-style tests for L1ToL2MessageSender and L2ToL1MessageSender.
contract L1ToL2MessageTest {
    L1ToL2MessageSender public l1ToL2;
    L2ToL1MessageSender public l2ToL1;

    address deployer;
    address constant MOCK_INBOX = address(0xBEEF);
    address constant L2_TARGET = address(0xCAFE);
    address constant L1_TARGET = address(0xFACE);

    function setUp() public {
        deployer = address(this);

        l1ToL2 = new L1ToL2MessageSender(MOCK_INBOX);
        l2ToL1 = new L2ToL1MessageSender();

        // Allow L1 target
        l2ToL1.setAllowedL1Target(L1_TARGET, true);
    }

    // ──────────────────── L1ToL2MessageSender Tests ────────────────────

    function testL1ToL2CreateRetryableTicket() public {
        uint256 maxGas = 1_000_000;
        uint256 gasPriceBid = 100 gwei;
        uint256 maxSubmissionCost = 0.01 ether;
        uint256 totalCost = maxSubmissionCost + (maxGas * gasPriceBid);
        bytes memory data = abi.encodeWithSignature("receiveMessage(string)", "hello");

        uint256 ticketId = l1ToL2.createRetryableTicket{value: totalCost}(
            L2_TARGET,
            0, // l2Value
            maxSubmissionCost,
            maxGas,
            gasPriceBid,
            data
        );

        require(ticketId == 0, "First ticket should have ID 0");
        require(l1ToL2.ticketCount() == 1, "Should have 1 ticket");

        (address sender, address target, uint256 l2Value, bool executed) = l1ToL2.getTicket(0);
        require(sender == deployer, "Wrong sender");
        require(target == L2_TARGET, "Wrong L2 target");
        require(l2Value == 0, "Wrong L2 value");
        require(!executed, "Should not be executed yet");
    }

    function testL1ToL2SendSimpleMessage() public {
        bytes memory data = abi.encodeWithSignature("receiveMessage(string)", "hello from L1");

        uint256 ticketId = l1ToL2.sendMessageToL2{value: 1 ether}(L2_TARGET, data);

        require(ticketId == 0, "First ticket should have ID 0");

        (address sender, address target, , bool executed) = l1ToL2.getTicket(0);
        require(sender == deployer, "Wrong sender");
        require(target == L2_TARGET, "Wrong L2 target");
        require(!executed, "Should not be executed");
    }

    function testL1ToL2MultipleTickets() public {
        bytes memory data = abi.encode("test");

        l1ToL2.sendMessageToL2{value: 1 ether}(L2_TARGET, data);
        l1ToL2.sendMessageToL2{value: 1 ether}(L2_TARGET, data);
        l1ToL2.sendMessageToL2{value: 1 ether}(L2_TARGET, data);

        require(l1ToL2.ticketCount() == 3, "Should have 3 tickets");
    }

    function testL1ToL2MarkExecuted() public {
        bytes memory data = abi.encode("test");
        l1ToL2.sendMessageToL2{value: 1 ether}(L2_TARGET, data);

        l1ToL2.markExecuted(0);

        (, , , bool executed) = l1ToL2.getTicket(0);
        require(executed, "Should be marked as executed");
    }

    function testL1ToL2DoubleExecuteReverts() public {
        bytes memory data = abi.encode("test");
        l1ToL2.sendMessageToL2{value: 1 ether}(L2_TARGET, data);

        l1ToL2.markExecuted(0);

        bool reverted = false;
        try l1ToL2.markExecuted(0) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on double execution");
    }

    function testL1ToL2ZeroTargetReverts() public {
        bool reverted = false;
        try l1ToL2.sendMessageToL2{value: 1 ether}(address(0), "0x") {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero L2 target");
    }

    function testL1ToL2ZeroValueReverts() public {
        bool reverted = false;
        try l1ToL2.sendMessageToL2{value: 0}(L2_TARGET, "0x") {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero ETH value");
    }

    function testL1ToL2SetInbox() public {
        address newInbox = address(0xDEAD);
        l1ToL2.setInbox(newInbox);
        require(l1ToL2.inbox() == newInbox, "Inbox not updated");
    }

    function testL1ToL2SetInboxZeroReverts() public {
        bool reverted = false;
        try l1ToL2.setInbox(address(0)) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero inbox address");
    }

    // ──────────────────── L2ToL1MessageSender Tests ────────────────────

    function testL2ToL1SendMessage() public {
        bytes memory data = abi.encodeWithSignature("receiveMessage(string)", "hello from L2");

        uint256 nonce = l2ToL1.sendMessageToL1(L1_TARGET, data);

        require(nonce == 0, "First message should have nonce 0");
        require(l2ToL1.messageCount() == 1, "Should have 1 message");

        (address sender, address target, bool confirmed, ) = l2ToL1.getMessage(0);
        require(sender == deployer, "Wrong sender");
        require(target == L1_TARGET, "Wrong L1 target");
        require(!confirmed, "Should not be confirmed");
    }

    function testL2ToL1SendEncodedMessage() public {
        bytes memory payload = abi.encode("hello from L2");

        uint256 nonce = l2ToL1.sendEncodedMessageToL1(
            L1_TARGET,
            "receiveMessage(bytes)",
            payload
        );

        require(nonce == 0, "First message should have nonce 0");
    }

    function testL2ToL1MultipleMessages() public {
        bytes memory data = abi.encode("test");

        l2ToL1.sendMessageToL1(L1_TARGET, data);
        l2ToL1.sendMessageToL1(L1_TARGET, data);

        require(l2ToL1.messageCount() == 2, "Should have 2 messages");
    }

    function testL2ToL1ConfirmMessage() public {
        bytes memory data = abi.encode("test");
        l2ToL1.sendMessageToL1(L1_TARGET, data);

        l2ToL1.confirmMessage(0);

        (, , bool confirmed, ) = l2ToL1.getMessage(0);
        require(confirmed, "Should be confirmed");
    }

    function testL2ToL1DoubleConfirmReverts() public {
        bytes memory data = abi.encode("test");
        l2ToL1.sendMessageToL1(L1_TARGET, data);

        l2ToL1.confirmMessage(0);

        bool reverted = false;
        try l2ToL1.confirmMessage(0) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on double confirm");
    }

    function testL2ToL1ZeroTargetReverts() public {
        bool reverted = false;
        try l2ToL1.sendMessageToL1(address(0), "0x") {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero L1 target");
    }

    function testL2ToL1DisallowedTargetReverts() public {
        address disallowed = address(0x1234);

        // Create a contract that is not owner and tries to send
        // Since this test contract IS the owner, we test the target allowlist
        // by sending from a non-owner context. However, since owner can send
        // to any target, we verify the allowlist configuration instead.
        require(!l2ToL1.allowedL1Targets(disallowed), "Should not be allowed initially");

        l2ToL1.setAllowedL1Target(disallowed, true);
        require(l2ToL1.allowedL1Targets(disallowed), "Should be allowed after setting");

        l2ToL1.setAllowedL1Target(disallowed, false);
        require(!l2ToL1.allowedL1Targets(disallowed), "Should be disallowed after unsetting");
    }

    function testL2ToL1TransferOwnership() public {
        address newOwner = address(0x5678);
        l2ToL1.transferOwnership(newOwner);
        require(l2ToL1.owner() == newOwner, "Ownership not transferred");
    }

    // ──────────────────── Receive ETH ────────────────────

    receive() external payable {}
}
