// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "../contracts/token/ArbitrumToken.sol";
import "../contracts/gateway/TokenGateway.sol";

/// @title TokenGatewayTest
/// @notice Foundry-style tests for TokenGateway: deposit, withdrawal, finalization.
contract TokenGatewayTest {
    ArbitrumToken public l1Token;
    ArbitrumToken public l2Token;
    TokenGateway public l1Gateway;
    TokenGateway public l2Gateway;

    address deployer;
    address constant USER = address(0x1);

    function setUp() public {
        deployer = address(this);

        // Deploy L1 token with initial supply
        l1Token = new ArbitrumToken("Arbitrum Token", "ARB", 1_000_000e18);

        // Deploy L2 token with zero initial supply (minted via gateway)
        l2Token = new ArbitrumToken("Arbitrum Token (L2)", "ARB-L2", 0);

        // Deploy gateways
        l1Gateway = new TokenGateway(address(0)); // counterpart set later
        l2Gateway = new TokenGateway(address(l1Gateway));

        // Link L1 gateway to L2 gateway
        l1Gateway.setCounterpartGateway(address(l2Gateway));

        // Set token mappings on both gateways
        l1Gateway.setTokenMapping(address(l1Token), address(l2Token));
        l2Gateway.setTokenMapping(address(l1Token), address(l2Token));

        // Authorize L2 gateway to mint L2 tokens
        l2Token.setGateway(address(l2Gateway));

        // Give user some L1 tokens
        l1Token.transfer(USER, 10_000e18);

        // Approve gateways
        l1Token.approve(address(l1Gateway), type(uint256).max);
    }

    // ──────────────────── Deposit Tests ────────────────────

    function testDeposit() public {
        // Approve from deployer and deposit
        uint256 depositAmount = 5_000e18;
        uint256 balanceBefore = l1Token.balanceOf(deployer);

        uint256 depositId = l1Gateway.deposit(address(l1Token), USER, depositAmount);

        require(depositId == 0, "First deposit should have ID 0");
        require(l1Gateway.depositCount() == 1, "Should have 1 deposit");
        require(
            l1Gateway.getLockedBalance(address(l1Token)) == depositAmount,
            "Wrong locked balance"
        );
        require(
            l1Token.balanceOf(deployer) == balanceBefore - depositAmount,
            "Wrong deployer balance after deposit"
        );
    }

    function testDepositZeroAmountReverts() public {
        bool reverted = false;
        try l1Gateway.deposit(address(l1Token), USER, 0) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero amount deposit");
    }

    function testDepositUnmappedTokenReverts() public {
        ArbitrumToken unmapped = new ArbitrumToken("Unmapped", "UNM", 1_000e18);
        unmapped.approve(address(l1Gateway), type(uint256).max);

        bool reverted = false;
        try l1Gateway.deposit(address(unmapped), USER, 100e18) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on unmapped token deposit");
    }

    function testDepositZeroRecipientReverts() public {
        bool reverted = false;
        try l1Gateway.deposit(address(l1Token), address(0), 100e18) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero recipient");
    }

    // ──────────────────── Finalize Deposit Tests ────────────────────

    function testFinalizeDeposit() public {
        uint256 depositAmount = 5_000e18;

        // Deposit on L1 side
        l1Gateway.deposit(address(l1Token), USER, depositAmount);

        // Finalize on L2 side (as owner, simulating counterpart gateway call)
        l2Gateway.finalizeDeposit(0, address(l2Token), USER, depositAmount);

        require(l2Token.balanceOf(USER) == depositAmount, "Wrong L2 balance after finalization");
    }

    function testFinalizeDepositMultiple() public {
        // Multiple deposits and finalizations
        l1Gateway.deposit(address(l1Token), USER, 1_000e18);
        l1Gateway.deposit(address(l1Token), deployer, 2_000e18);

        l2Gateway.finalizeDeposit(0, address(l2Token), USER, 1_000e18);
        l2Gateway.finalizeDeposit(1, address(l2Token), deployer, 2_000e18);

        require(l2Token.balanceOf(USER) == 1_000e18, "Wrong USER L2 balance");
        require(l2Token.balanceOf(deployer) == 2_000e18, "Wrong deployer L2 balance");
    }

    // ──────────────────── Withdrawal Tests ────────────────────

    function testWithdraw() public {
        // First deposit and finalize to get L2 tokens
        uint256 depositAmount = 5_000e18;
        l1Gateway.deposit(address(l1Token), deployer, depositAmount);
        l2Gateway.finalizeDeposit(0, address(l2Token), deployer, depositAmount);

        // Approve L2 gateway for burn
        l2Token.approve(address(l2Gateway), type(uint256).max);

        // Withdraw
        uint256 withdrawAmount = 2_000e18;
        uint256 withdrawalId = l2Gateway.withdraw(address(l2Token), deployer, withdrawAmount);

        require(withdrawalId == 0, "First withdrawal should have ID 0");
        require(l2Gateway.withdrawalCount() == 1, "Should have 1 withdrawal");
        require(
            l2Token.balanceOf(deployer) == depositAmount - withdrawAmount,
            "Wrong L2 balance after withdrawal"
        );
    }

    function testWithdrawZeroAmountReverts() public {
        bool reverted = false;
        try l2Gateway.withdraw(address(l2Token), deployer, 0) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero amount withdrawal");
    }

    // ──────────────────── Finalize Withdrawal Tests ────────────────────

    function testFinalizeWithdrawal() public {
        // Deposit, finalize deposit, withdraw
        uint256 depositAmount = 5_000e18;
        l1Gateway.deposit(address(l1Token), deployer, depositAmount);
        l2Gateway.finalizeDeposit(0, address(l2Token), deployer, depositAmount);

        l2Token.approve(address(l2Gateway), type(uint256).max);
        uint256 withdrawAmount = 3_000e18;
        l2Gateway.withdraw(address(l2Token), deployer, withdrawAmount);

        // Finalize withdrawal on L1 (release locked tokens)
        uint256 l1BalanceBefore = l1Token.balanceOf(deployer);
        l1Gateway.finalizeWithdrawal(0, address(l1Token), deployer, withdrawAmount);

        require(
            l1Token.balanceOf(deployer) == l1BalanceBefore + withdrawAmount,
            "Wrong L1 balance after finalized withdrawal"
        );
        require(
            l1Gateway.getLockedBalance(address(l1Token)) == depositAmount - withdrawAmount,
            "Wrong locked balance after withdrawal"
        );
    }

    function testFinalizeWithdrawalInsufficientLockedReverts() public {
        // Try to finalize a withdrawal with more than locked balance
        bool reverted = false;
        try l1Gateway.finalizeWithdrawal(0, address(l1Token), deployer, 1_000e18) {
            // should not succeed (nothing locked)
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on insufficient locked balance");
    }

    // ──────────────────── Admin Tests ────────────────────

    function testSetTokenMapping() public {
        ArbitrumToken newL1 = new ArbitrumToken("New L1", "NL1", 0);
        ArbitrumToken newL2 = new ArbitrumToken("New L2", "NL2", 0);

        l1Gateway.setTokenMapping(address(newL1), address(newL2));

        require(
            l1Gateway.getMappedToken(address(newL1)) == address(newL2),
            "Wrong L1->L2 mapping"
        );
        require(
            l1Gateway.getMappedToken(address(newL2)) == address(newL1),
            "Wrong L2->L1 mapping"
        );
    }

    function testSetTokenMappingZeroReverts() public {
        bool reverted = false;
        try l1Gateway.setTokenMapping(address(0), address(0x1)) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero L1 token");
    }

    function testSetCounterpartGateway() public {
        address newCounterpart = address(0x9999);
        l1Gateway.setCounterpartGateway(newCounterpart);
        require(l1Gateway.counterpartGateway() == newCounterpart, "Counterpart not updated");
    }

    function testTransferOwnership() public {
        address newOwner = address(0x5678);
        l1Gateway.transferOwnership(newOwner);
        require(l1Gateway.owner() == newOwner, "Ownership not transferred");
    }

    function testTransferOwnershipZeroReverts() public {
        bool reverted = false;
        try l1Gateway.transferOwnership(address(0)) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero owner");
    }

    // ──────────────────── Full Bridge Roundtrip ────────────────────

    function testFullBridgeRoundtrip() public {
        uint256 amount = 1_000e18;

        // 1. Deposit L1 -> L2
        l1Gateway.deposit(address(l1Token), deployer, amount);
        l2Gateway.finalizeDeposit(0, address(l2Token), deployer, amount);

        require(l2Token.balanceOf(deployer) == amount, "L2 balance wrong after deposit");
        require(l1Gateway.getLockedBalance(address(l1Token)) == amount, "L1 locked wrong");

        // 2. Withdraw L2 -> L1
        l2Token.approve(address(l2Gateway), type(uint256).max);
        l2Gateway.withdraw(address(l2Token), deployer, amount);

        require(l2Token.balanceOf(deployer) == 0, "L2 balance should be 0 after withdrawal");

        // 3. Finalize withdrawal on L1
        uint256 l1Before = l1Token.balanceOf(deployer);
        l1Gateway.finalizeWithdrawal(0, address(l1Token), deployer, amount);

        require(l1Token.balanceOf(deployer) == l1Before + amount, "L1 balance wrong after roundtrip");
        require(l1Gateway.getLockedBalance(address(l1Token)) == 0, "L1 should have 0 locked after roundtrip");
    }
}
