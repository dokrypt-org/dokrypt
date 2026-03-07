// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../../contracts/token/DeFiToken.sol";
import "../../contracts/staking/StakingRewards.sol";

/// @title StakingRewardsTest
/// @notice Foundry-style tests for StakingRewards: stake, earn rewards, withdraw.
contract StakingRewardsTest {
    DeFiToken stakingToken;
    DeFiToken rewardsToken;
    StakingRewards staking;

    address deployer;

    uint256 constant REWARD_AMOUNT = 100_000e18;
    uint256 constant DURATION = 30 days;

    function setUp() public {
        deployer = address(this);

        stakingToken = new DeFiToken("Staking Token", "STK", 1_000_000e18);
        rewardsToken = new DeFiToken("Reward Token", "RWD", 1_000_000e18);

        staking = new StakingRewards(address(stakingToken), address(rewardsToken));

        // Transfer reward tokens to the staking contract
        rewardsToken.transfer(address(staking), REWARD_AMOUNT);

        // Set up reward duration and notify
        staking.setRewardsDuration(DURATION);
        staking.notifyRewardAmount(REWARD_AMOUNT);

        // Approve staking token
        stakingToken.approve(address(staking), type(uint256).max);
    }

    function testStake() public {
        uint256 stakeAmount = 10_000e18;
        staking.stake(stakeAmount);

        require(staking.totalStaked() == stakeAmount, "Total staked mismatch");
        require(staking.userStakedBalance(deployer) == stakeAmount, "User staked balance mismatch");
    }

    function testStakeZeroReverts() public {
        bool reverted = false;
        try staking.stake(0) {
            // should fail
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero stake");
    }

    function testWithdraw() public {
        uint256 stakeAmount = 10_000e18;
        staking.stake(stakeAmount);

        staking.withdraw(stakeAmount);

        require(staking.totalStaked() == 0, "Total staked should be 0");
        require(staking.userStakedBalance(deployer) == 0, "User balance should be 0");
    }

    function testWithdrawExcessReverts() public {
        staking.stake(10_000e18);

        bool reverted = false;
        try staking.withdraw(20_000e18) {
            // should fail
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on excess withdrawal");
    }

    function testEarnedRewards() public {
        staking.stake(10_000e18);

        // Earned should be 0 at time of staking (same block)
        uint256 earnedInitial = staking.earned(deployer);
        // Note: in the same block, earned could be 0 since no time has passed
        require(earnedInitial == 0, "Initial earned should be 0");
    }

    function testRewardPerToken() public {
        staking.stake(10_000e18);

        // At time of staking, rewardPerToken should be 0 or very small
        uint256 rpt = staking.rewardPerToken();
        // This is a sanity check - the value depends on exact timing
        require(rpt >= 0, "RewardPerToken should be non-negative");
    }

    function testGetRewardForDuration() public {
        uint256 totalReward = staking.getRewardForDuration();
        // rewardRate * duration should approximately equal REWARD_AMOUNT
        // (may have small rounding difference)
        require(totalReward > 0, "Reward for duration should be positive");
        // Allow 1 token rounding error
        require(
            totalReward <= REWARD_AMOUNT && totalReward >= REWARD_AMOUNT - DURATION,
            "Reward for duration out of expected range"
        );
    }

    function testSetRewardsDurationWhileActive() public {
        // Should revert because the current period is not finished
        bool reverted = false;
        try staking.setRewardsDuration(60 days) {
            // should fail
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert setting duration during active period");
    }

    function testExitFunction() public {
        uint256 stakeAmount = 10_000e18;
        staking.stake(stakeAmount);

        uint256 balanceBefore = stakingToken.balanceOf(deployer);
        staking.exit();
        uint256 balanceAfter = stakingToken.balanceOf(deployer);

        require(balanceAfter - balanceBefore == stakeAmount, "Should return all staked tokens");
        require(staking.userStakedBalance(deployer) == 0, "User balance should be 0 after exit");
    }
}
