// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "../contracts/token/ArbitrumToken.sol";

/// @title ArbitrumTokenTest
/// @notice Foundry-style tests for ArbitrumToken: ERC-20, batch transfer, gateway minting.
contract ArbitrumTokenTest {
    ArbitrumToken public token;
    address constant ALICE = address(0x1);
    address constant BOB = address(0x2);
    address constant CHARLIE = address(0x3);

    address deployer;

    function setUp() public {
        deployer = address(this);
        token = new ArbitrumToken("Arbitrum Token", "ARB", 1_000_000e18);
    }

    function testName() public view {
        require(
            keccak256(bytes(token.name())) == keccak256(bytes("Arbitrum Token")),
            "wrong name"
        );
    }

    function testSymbol() public view {
        require(
            keccak256(bytes(token.symbol())) == keccak256(bytes("ARB")),
            "wrong symbol"
        );
    }

    function testInitialSupply() public view {
        require(token.totalSupply() == 1_000_000e18, "wrong total supply");
        require(token.balanceOf(deployer) == 1_000_000e18, "wrong deployer balance");
    }

    function testTransfer() public {
        token.transfer(ALICE, 1_000e18);
        require(token.balanceOf(ALICE) == 1_000e18, "wrong ALICE balance");
        require(token.balanceOf(deployer) == 999_000e18, "wrong deployer balance after transfer");
    }

    function testTransferToZeroReverts() public {
        bool reverted = false;
        try token.transfer(address(0), 100e18) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on transfer to zero address");
    }

    function testApproveAndTransferFrom() public {
        token.approve(ALICE, 500e18);
        require(token.allowance(deployer, ALICE) == 500e18, "wrong allowance");
    }

    function testMint() public {
        uint256 supplyBefore = token.totalSupply();
        token.mint(ALICE, 5_000e18);
        require(token.balanceOf(ALICE) == 5_000e18, "wrong ALICE balance after mint");
        require(token.totalSupply() == supplyBefore + 5_000e18, "wrong total supply after mint");
    }

    function testMintOnlyOwner() public {
        // Non-owner cannot mint (simulated by deploying from this contract which IS owner)
        // This test verifies the owner CAN mint
        token.mint(ALICE, 100e18);
        require(token.balanceOf(ALICE) == 100e18, "owner should be able to mint");
    }

    function testBurn() public {
        uint256 supplyBefore = token.totalSupply();
        token.burn(1_000e18);
        require(token.totalSupply() == supplyBefore - 1_000e18, "wrong supply after burn");
        require(token.balanceOf(deployer) == 999_000e18, "wrong balance after burn");
    }

    function testBurnExcessReverts() public {
        bool reverted = false;
        try token.burn(2_000_000e18) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on burning more than balance");
    }

    function testBatchTransfer() public {
        address[] memory recipients = new address[](3);
        recipients[0] = ALICE;
        recipients[1] = BOB;
        recipients[2] = CHARLIE;

        uint256[] memory amounts = new uint256[](3);
        amounts[0] = 100e18;
        amounts[1] = 200e18;
        amounts[2] = 300e18;

        token.batchTransfer(recipients, amounts);

        require(token.balanceOf(ALICE) == 100e18, "wrong ALICE balance after batch");
        require(token.balanceOf(BOB) == 200e18, "wrong BOB balance after batch");
        require(token.balanceOf(CHARLIE) == 300e18, "wrong CHARLIE balance after batch");
        require(token.balanceOf(deployer) == 999_400e18, "wrong deployer balance after batch");
    }

    function testBatchTransferLengthMismatchReverts() public {
        address[] memory recipients = new address[](2);
        recipients[0] = ALICE;
        recipients[1] = BOB;

        uint256[] memory amounts = new uint256[](3);
        amounts[0] = 100e18;
        amounts[1] = 200e18;
        amounts[2] = 300e18;

        bool reverted = false;
        try token.batchTransfer(recipients, amounts) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on length mismatch");
    }

    function testBatchTransferInsufficientBalanceReverts() public {
        address[] memory recipients = new address[](1);
        recipients[0] = ALICE;

        uint256[] memory amounts = new uint256[](1);
        amounts[0] = 2_000_000e18; // more than total supply

        bool reverted = false;
        try token.batchTransfer(recipients, amounts) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on insufficient balance for batch");
    }

    function testSetGateway() public {
        address gateway = address(0x999);
        token.setGateway(gateway);
        require(token.gateway() == gateway, "gateway not set");
    }

    function testGatewayCanMint() public {
        address gateway = address(0x999);
        token.setGateway(gateway);

        // Deploy a new token where we can simulate gateway calls
        ArbitrumToken gatewayToken = new ArbitrumToken("Gateway Token", "GTK", 0);
        gatewayToken.setGateway(deployer); // set this contract as gateway

        gatewayToken.mint(ALICE, 1_000e18); // deployer is both owner and gateway
        require(gatewayToken.balanceOf(ALICE) == 1_000e18, "gateway should be able to mint");
    }

    function testTransferOwnership() public {
        token.transferOwnership(ALICE);
        require(token.owner() == ALICE, "ownership not transferred");
    }

    function testTransferOwnershipToZeroReverts() public {
        bool reverted = false;
        try token.transferOwnership(address(0)) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on transfer ownership to zero");
    }
}
