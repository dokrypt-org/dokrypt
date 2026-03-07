// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../../contracts/token/ManagedToken.sol";
import "../../contracts/staking/StakeRewards.sol";

contract StakeRewardsTest is Test {
    ManagedToken stakingToken;
    ManagedToken rewardsToken;
    StakeRewards staking;

    address owner;
    address alice;

    uint256 constant REWARD_AMOUNT   = 100_000 ether;
    uint256 constant REWARD_DURATION = 30 days;
    uint256 constant STAKE_AMOUNT    = 1_000 ether;

    function setUp() public {
        owner = address(this);
        alice = makeAddr("alice");

        stakingToken = new ManagedToken("StakeToken", "STK", 1_000_000 ether);
        rewardsToken = new ManagedToken("RewardToken", "RWD", 1_000_000 ether);

        staking = new StakeRewards(address(stakingToken), address(rewardsToken));

        // Fund alice with staking tokens
        stakingToken.transfer(alice, STAKE_AMOUNT);

        // Fund staking contract with reward tokens and configure
        rewardsToken.transfer(address(staking), REWARD_AMOUNT);
        staking.setRewardsDuration(REWARD_DURATION);
        staking.notifyRewardAmount(REWARD_AMOUNT);
    }

    // ---- Stake ----

    function test_stake() public {
        vm.startPrank(alice);
        stakingToken.approve(address(staking), STAKE_AMOUNT);
        staking.stake(STAKE_AMOUNT);
        vm.stopPrank();

        assertEq(staking.stakedBalance(alice), STAKE_AMOUNT);
        assertEq(staking.totalStaked(), STAKE_AMOUNT);
    }

    function test_stake_zero() public {
        vm.prank(alice);
        vm.expectRevert("StakeRewards: cannot stake zero");
        staking.stake(0);
    }

    // ---- Earn ----

    function test_earnRewards() public {
        vm.startPrank(alice);
        stakingToken.approve(address(staking), STAKE_AMOUNT);
        staking.stake(STAKE_AMOUNT);
        vm.stopPrank();

        // Advance time by 1 day
        vm.warp(block.timestamp + 1 days);

        uint256 pendingReward = staking.earned(alice);
        assertTrue(pendingReward > 0, "should have earned rewards");
    }

    // ---- Withdraw ----

    function test_withdraw() public {
        vm.startPrank(alice);
        stakingToken.approve(address(staking), STAKE_AMOUNT);
        staking.stake(STAKE_AMOUNT);

        staking.withdraw(STAKE_AMOUNT);
        vm.stopPrank();

        assertEq(staking.stakedBalance(alice), 0);
        assertEq(stakingToken.balanceOf(alice), STAKE_AMOUNT);
    }

    function test_withdraw_exceeds() public {
        vm.startPrank(alice);
        stakingToken.approve(address(staking), STAKE_AMOUNT);
        staking.stake(STAKE_AMOUNT);

        vm.expectRevert("StakeRewards: insufficient staked balance");
        staking.withdraw(STAKE_AMOUNT + 1);
        vm.stopPrank();
    }

    // ---- Reward Claim ----

    function test_getReward() public {
        vm.startPrank(alice);
        stakingToken.approve(address(staking), STAKE_AMOUNT);
        staking.stake(STAKE_AMOUNT);
        vm.stopPrank();

        vm.warp(block.timestamp + 7 days);

        vm.prank(alice);
        staking.getReward();

        uint256 rewardBalance = rewardsToken.balanceOf(alice);
        assertTrue(rewardBalance > 0, "should have received rewards");
    }
}
