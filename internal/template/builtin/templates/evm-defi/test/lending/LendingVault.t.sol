// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../../contracts/token/DeFiToken.sol";
import "../../contracts/lending/LendingVault.sol";
import "../../contracts/lending/InterestModel.sol";
import "../../contracts/oracle/PriceOracle.sol";

/// @title LendingVaultTest
/// @notice Foundry-style tests for LendingVault: deposit, borrow, repay, and liquidation.
contract LendingVaultTest {
    DeFiToken borrowToken;
    DeFiToken collateralToken;
    LendingVault vault;
    InterestModel interestModel;
    PriceOracle oracle;

    address deployer;

    function setUp() public {
        deployer = address(this);

        borrowToken = new DeFiToken("Borrow Token", "BORROW", 10_000_000e18);
        collateralToken = new DeFiToken("Collateral Token", "COL", 10_000_000e18);

        interestModel = new InterestModel();
        oracle = new PriceOracle();

        vault = new LendingVault(
            address(borrowToken),
            address(interestModel),
            address(oracle)
        );

        // Set prices: both tokens worth $1
        oracle.setPrice(address(borrowToken), 1e18);
        oracle.setPrice(address(collateralToken), 1e18);

        // Add collateral support
        vault.addCollateral(address(collateralToken));

        // Supply borrow tokens to the vault
        borrowToken.approve(address(vault), type(uint256).max);
        vault.supply(1_000_000e18);

        // Approve vault for collateral
        collateralToken.approve(address(vault), type(uint256).max);
    }

    function testDepositCollateral() public {
        uint256 depositAmount = 10_000e18;
        vault.depositCollateral(address(collateralToken), depositAmount);

        uint256 deposited = vault.userCollateral(deployer, address(collateralToken));
        require(deposited == depositAmount, "Deposit amount mismatch");
    }

    function testBorrow() public {
        // Deposit collateral
        vault.depositCollateral(address(collateralToken), 10_000e18);

        // Borrow up to 75% of collateral value
        uint256 borrowAmount = 7_000e18;
        vault.borrow(borrowAmount);

        uint256 debt = vault.getBorrowBalance(deployer);
        require(debt == borrowAmount, "Borrow amount mismatch");
    }

    function testBorrowExcessReverts() public {
        vault.depositCollateral(address(collateralToken), 10_000e18);

        // Try to borrow more than 75% collateral factor allows
        bool reverted = false;
        try vault.borrow(8_000e18) {
            // should not succeed with 75% CF and $1 prices
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on over-borrow");
    }

    function testRepay() public {
        vault.depositCollateral(address(collateralToken), 10_000e18);
        vault.borrow(5_000e18);

        // Approve vault for repayment
        borrowToken.approve(address(vault), type(uint256).max);

        vault.repay(5_000e18);

        uint256 debtAfter = vault.getBorrowBalance(deployer);
        require(debtAfter == 0, "Debt should be zero after full repay");
    }

    function testPartialRepay() public {
        vault.depositCollateral(address(collateralToken), 10_000e18);
        vault.borrow(5_000e18);

        borrowToken.approve(address(vault), type(uint256).max);
        vault.repay(2_000e18);

        uint256 debtAfter = vault.getBorrowBalance(deployer);
        require(debtAfter > 0, "Debt should remain after partial repay");
        require(debtAfter < 5_000e18, "Debt should decrease after partial repay");
    }

    function testWithdrawCollateral() public {
        vault.depositCollateral(address(collateralToken), 10_000e18);

        // Withdraw all if no borrows
        vault.withdrawCollateral(address(collateralToken), 10_000e18);

        uint256 remaining = vault.userCollateral(deployer, address(collateralToken));
        require(remaining == 0, "Should have no collateral remaining");
    }

    function testWithdrawWouldMakeInsolvt() public {
        vault.depositCollateral(address(collateralToken), 10_000e18);
        vault.borrow(7_000e18);

        // Try to withdraw too much collateral
        bool reverted = false;
        try vault.withdrawCollateral(address(collateralToken), 9_000e18) {
            // would make insolvent
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on insolvency-causing withdrawal");
    }

    function testLiquidation() public {
        vault.depositCollateral(address(collateralToken), 10_000e18);
        vault.borrow(7_000e18);

        // Crash collateral price to make the position underwater
        // At $0.50, collateral = $5000, debt = $7000, CF check: 5000*0.75 = 3750 < 7000
        oracle.setPrice(address(collateralToken), 0.5e18);

        require(!vault.isSolvent(deployer), "Borrower should be insolvent");

        // Liquidate
        uint256 debtToRepay = 3_000e18;
        borrowToken.approve(address(vault), type(uint256).max);

        vault.liquidate(deployer, address(collateralToken), debtToRepay);

        uint256 debtAfter = vault.getBorrowBalance(deployer);
        require(debtAfter < 7_000e18, "Debt should decrease after liquidation");
    }

    function testSolventCannotBeLiquidated() public {
        vault.depositCollateral(address(collateralToken), 10_000e18);
        vault.borrow(5_000e18);

        bool reverted = false;
        try vault.liquidate(deployer, address(collateralToken), 1_000e18) {
            // should not succeed on solvent position
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on liquidating solvent borrower");
    }
}
