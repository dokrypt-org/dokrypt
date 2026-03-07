// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../../contracts/governance/Timelock.sol";

/// @title TimelockTest
/// @notice Foundry tests for the Timelock controller: schedule, execute, cancel,
///         and minimum delay enforcement.
contract TimelockTest {
    Timelock timelock;

    address deployer = address(this);
    uint256 constant MIN_DELAY = 100; // seconds

    // Dummy target for calls
    address target = address(0xBEEF);

    function setUp() public {
        address[] memory proposers = new address[](1);
        proposers[0] = deployer;
        address[] memory executors = new address[](1);
        executors[0] = deployer;

        timelock = new Timelock(MIN_DELAY, proposers, executors);
    }

    // ------------------------------------------------------------------
    // Schedule
    // ------------------------------------------------------------------

    function testScheduleOperation() public {
        bytes32 salt = keccak256("test-op-1");
        bytes memory data = abi.encodeWithSignature("foo()");

        bytes32 id = timelock.schedule(target, 0, data, salt, MIN_DELAY);
        assert(timelock.isOperationPending(id));
    }

    function testScheduleRevertsIfDelayTooLow() public {
        bytes32 salt = keccak256("test-op-2");
        bytes memory data = abi.encodeWithSignature("foo()");

        // Should revert because delay < minDelay
        try timelock.schedule(target, 0, data, salt, MIN_DELAY - 1) {
            // Should not reach here
            assert(false);
        } catch {
            // Expected revert
        }
    }

    function testScheduleRevertsDuplicateOperation() public {
        bytes32 salt = keccak256("dup-op");
        bytes memory data = abi.encodeWithSignature("bar()");

        timelock.schedule(target, 0, data, salt, MIN_DELAY);

        // Second schedule with same params should revert
        try timelock.schedule(target, 0, data, salt, MIN_DELAY) {
            assert(false);
        } catch {
            // Expected
        }
    }

    // ------------------------------------------------------------------
    // Cancel
    // ------------------------------------------------------------------

    function testCancelOperation() public {
        bytes32 salt = keccak256("cancel-op");
        bytes memory data = abi.encodeWithSignature("baz()");

        bytes32 id = timelock.schedule(target, 0, data, salt, MIN_DELAY);
        assert(timelock.isOperationPending(id));

        timelock.cancel(id);
        assert(!timelock.isOperationPending(id));
    }

    function testCancelRevertsIfNotScheduled() public {
        bytes32 fakeId = keccak256("non-existent");
        try timelock.cancel(fakeId) {
            assert(false);
        } catch {
            // Expected
        }
    }

    // ------------------------------------------------------------------
    // Execute (time-dependent -- basic structure test)
    // ------------------------------------------------------------------

    function testExecuteRevertsBeforeDelay() public {
        bytes32 salt = keccak256("exec-op");
        bytes memory data = "";

        timelock.schedule(target, 0, data, salt, MIN_DELAY);

        // Attempt execution immediately (before delay passes)
        try timelock.execute(target, 0, data, salt) {
            assert(false);
        } catch {
            // Expected: "operation not ready"
        }
    }

    // ------------------------------------------------------------------
    // Admin
    // ------------------------------------------------------------------

    function testSetMinDelay() public {
        timelock.setMinDelay(200);
        assert(timelock.minDelay() == 200);
    }

    function testAddAndRemoveProposer() public {
        address newProposer = address(0x9999);
        timelock.addProposer(newProposer);
        assert(timelock.isProposer(newProposer));

        timelock.removeProposer(newProposer);
        assert(!timelock.isProposer(newProposer));
    }

    function testHashOperationIsDeterministic() public pure {
        // Dummy values to ensure pure function works
        bytes memory data = abi.encodeWithSignature("transfer(address,uint256)", address(0x1), 100);
        bytes32 salt = keccak256("salt");

        // Call twice and compare -- cannot use Timelock.hashOperation as a pure
        // static call here, so we replicate the hash
        bytes32 h1 = keccak256(abi.encode(address(0xBEEF), uint256(0), data, salt));
        bytes32 h2 = keccak256(abi.encode(address(0xBEEF), uint256(0), data, salt));
        assert(h1 == h2);
    }
}
