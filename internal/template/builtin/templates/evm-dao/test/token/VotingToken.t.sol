// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../../contracts/token/VotingToken.sol";

/// @title VotingTokenTest
/// @notice Foundry tests for the VotingToken: delegation, checkpoints, and
///         vote power tracking.
contract VotingTokenTest {
    VotingToken token;

    address deployer = address(this);
    address alice = address(0xA11CE);
    address bob = address(0xB0B);

    uint256 constant INITIAL_SUPPLY = 1_000_000 ether;

    function setUp() public {
        token = new VotingToken("DAO Token", "DAO", INITIAL_SUPPLY);
    }

    // ------------------------------------------------------------------
    // Basic ERC-20
    // ------------------------------------------------------------------

    function testInitialBalance() public view {
        assert(token.balanceOf(deployer) == INITIAL_SUPPLY);
        assert(token.totalSupply() == INITIAL_SUPPLY);
    }

    function testTransfer() public {
        token.transfer(alice, 1000 ether);
        assert(token.balanceOf(alice) == 1000 ether);
        assert(token.balanceOf(deployer) == INITIAL_SUPPLY - 1000 ether);
    }

    function testApproveAndTransferFrom() public {
        token.transfer(alice, 500 ether);
        // Since we cannot call from alice directly without vm.prank,
        // we test approve from deployer's perspective
        token.approve(alice, 100 ether);
        assert(token.allowance(deployer, alice) == 100 ether);
    }

    // ------------------------------------------------------------------
    // Delegation
    // ------------------------------------------------------------------

    function testDelegateSelf() public {
        token.delegate(deployer);
        assert(token.getVotes(deployer) == INITIAL_SUPPLY);
    }

    function testDelegateToAnother() public {
        token.delegate(alice);
        assert(token.getVotes(alice) == INITIAL_SUPPLY);
        assert(token.getVotes(deployer) == 0);
    }

    function testDelegatesReturnsDelegate() public {
        token.delegate(bob);
        assert(token.delegates(deployer) == bob);
    }

    function testRedelegate() public {
        token.delegate(alice);
        assert(token.getVotes(alice) == INITIAL_SUPPLY);

        token.delegate(bob);
        assert(token.getVotes(alice) == 0);
        assert(token.getVotes(bob) == INITIAL_SUPPLY);
    }

    // ------------------------------------------------------------------
    // Checkpoints
    // ------------------------------------------------------------------

    function testCheckpointCreatedOnDelegate() public {
        assert(token.numCheckpoints(alice) == 0);
        token.delegate(alice);
        assert(token.numCheckpoints(alice) >= 1);
    }

    function testVotePowerUpdatedOnTransfer() public {
        // Delegate deployer's tokens to alice
        token.delegate(alice);
        assert(token.getVotes(alice) == INITIAL_SUPPLY);

        // Transfer some tokens to bob (who has no delegate)
        token.transfer(bob, 200_000 ether);

        // alice loses vote power for the transferred tokens
        assert(token.getVotes(alice) == INITIAL_SUPPLY - 200_000 ether);
    }

    // ------------------------------------------------------------------
    // Mint
    // ------------------------------------------------------------------

    function testMintIncreasesSupply() public {
        uint256 mintAmount = 500 ether;
        token.mint(alice, mintAmount);
        assert(token.totalSupply() == INITIAL_SUPPLY + mintAmount);
        assert(token.balanceOf(alice) == mintAmount);
    }

    function testMintUpdatesVotePower() public {
        // alice delegates to self first (no tokens yet, so 0 votes)
        // Since we can't call from alice, we delegate deployer to alice
        // and then mint to deployer
        token.delegate(alice);
        uint256 votesBefore = token.getVotes(alice);

        token.mint(deployer, 100 ether);
        uint256 votesAfter = token.getVotes(alice);

        // deployer is delegated to alice, so alice gains the minted votes
        assert(votesAfter == votesBefore + 100 ether);
    }

    // ------------------------------------------------------------------
    // Edge cases
    // ------------------------------------------------------------------

    function testTransferToZeroReverts() public {
        try token.transfer(address(0), 100 ether) {
            assert(false);
        } catch {
            // Expected
        }
    }

    function testTransferInsufficientBalanceReverts() public {
        try token.transfer(alice, INITIAL_SUPPLY + 1) {
            assert(false);
        } catch {
            // Expected
        }
    }
}
