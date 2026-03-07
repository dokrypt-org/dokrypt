// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title StakingRewards
/// @notice Synthetix-style staking rewards contract. Users stake tokens and
/// earn rewards proportionally over a configurable duration.
contract StakingRewards {
    // ──────────────────── State ────────────────────
    address public owner;
    address public stakingToken;
    address public rewardsToken;

    uint256 public rewardRate;        // rewards per second (scaled by 1e18)
    uint256 public rewardsDuration;   // duration of current reward period
    uint256 public periodFinish;      // timestamp when current period ends
    uint256 public lastUpdateTime;
    uint256 public rewardPerTokenStored;

    uint256 public totalStaked;

    mapping(address => uint256) public userStakedBalance;
    mapping(address => uint256) public userRewardPerTokenPaid;
    mapping(address => uint256) public rewards;

    // Reentrancy guard
    uint256 private _locked = 1;

    // ──────────────────── Events ────────────────────
    event Staked(address indexed user, uint256 amount);
    event Withdrawn(address indexed user, uint256 amount);
    event RewardPaid(address indexed user, uint256 reward);
    event RewardsDurationUpdated(uint256 newDuration);
    event RewardNotified(uint256 reward);

    // ──────────────────── Modifiers ────────────────────
    modifier onlyOwner() {
        require(msg.sender == owner, "StakingRewards: not owner");
        _;
    }

    modifier nonReentrant() {
        require(_locked == 1, "StakingRewards: reentrancy");
        _locked = 2;
        _;
        _locked = 1;
    }

    modifier updateReward(address account) {
        rewardPerTokenStored = rewardPerToken();
        lastUpdateTime = lastTimeRewardApplicable();
        if (account != address(0)) {
            rewards[account] = earned(account);
            userRewardPerTokenPaid[account] = rewardPerTokenStored;
        }
        _;
    }

    constructor(address _stakingToken, address _rewardsToken) {
        require(_stakingToken != address(0), "StakingRewards: zero staking token");
        require(_rewardsToken != address(0), "StakingRewards: zero rewards token");
        owner = msg.sender;
        stakingToken = _stakingToken;
        rewardsToken = _rewardsToken;
    }

    // ──────────────────── View Functions ────────────────────

    /// @notice Returns the last timestamp at which rewards are applicable.
    function lastTimeRewardApplicable() public view returns (uint256) {
        return block.timestamp < periodFinish ? block.timestamp : periodFinish;
    }

    /// @notice Returns the accumulated reward per staked token.
    function rewardPerToken() public view returns (uint256) {
        if (totalStaked == 0) {
            return rewardPerTokenStored;
        }
        return rewardPerTokenStored + (
            (lastTimeRewardApplicable() - lastUpdateTime) * rewardRate * 1e18 / totalStaked
        );
    }

    /// @notice Returns the amount of rewards earned by an account so far.
    /// @param account The user address.
    /// @return The total unclaimed reward amount.
    function earned(address account) public view returns (uint256) {
        return (
            userStakedBalance[account] * (rewardPerToken() - userRewardPerTokenPaid[account]) / 1e18
        ) + rewards[account];
    }

    /// @notice Returns total reward for the current duration period.
    function getRewardForDuration() external view returns (uint256) {
        return rewardRate * rewardsDuration;
    }

    // ──────────────────── User Actions ────────────────────

    /// @notice Stake tokens to begin earning rewards.
    /// @param amount Amount of staking tokens to deposit.
    function stake(uint256 amount) external nonReentrant updateReward(msg.sender) {
        require(amount > 0, "StakingRewards: zero amount");

        _safeTransferFrom(stakingToken, msg.sender, address(this), amount);
        totalStaked += amount;
        userStakedBalance[msg.sender] += amount;

        emit Staked(msg.sender, amount);
    }

    /// @notice Withdraw staked tokens.
    /// @param amount Amount of staking tokens to withdraw.
    function withdraw(uint256 amount) external nonReentrant updateReward(msg.sender) {
        require(amount > 0, "StakingRewards: zero amount");
        require(userStakedBalance[msg.sender] >= amount, "StakingRewards: insufficient balance");

        totalStaked -= amount;
        userStakedBalance[msg.sender] -= amount;
        _safeTransfer(stakingToken, msg.sender, amount);

        emit Withdrawn(msg.sender, amount);
    }

    /// @notice Claim accumulated rewards.
    function getReward() external nonReentrant updateReward(msg.sender) {
        uint256 reward = rewards[msg.sender];
        if (reward > 0) {
            rewards[msg.sender] = 0;
            _safeTransfer(rewardsToken, msg.sender, reward);
            emit RewardPaid(msg.sender, reward);
        }
    }

    /// @notice Withdraw all staked tokens and claim all rewards.
    function exit() external {
        // We call the external functions which handle their own guards
        uint256 stakedAmount = userStakedBalance[msg.sender];
        if (stakedAmount > 0) {
            // Inline withdraw to avoid double-lock
            _locked = 2;
            rewardPerTokenStored = rewardPerToken();
            lastUpdateTime = lastTimeRewardApplicable();
            rewards[msg.sender] = earned(msg.sender);
            userRewardPerTokenPaid[msg.sender] = rewardPerTokenStored;

            totalStaked -= stakedAmount;
            userStakedBalance[msg.sender] = 0;
            _safeTransfer(stakingToken, msg.sender, stakedAmount);
            emit Withdrawn(msg.sender, stakedAmount);

            uint256 reward = rewards[msg.sender];
            if (reward > 0) {
                rewards[msg.sender] = 0;
                _safeTransfer(rewardsToken, msg.sender, reward);
                emit RewardPaid(msg.sender, reward);
            }
            _locked = 1;
        }
    }

    // ──────────────────── Admin ────────────────────

    /// @notice Notify the contract of a new reward amount to distribute.
    /// @param reward Total reward amount for the current period.
    function notifyRewardAmount(uint256 reward) external onlyOwner updateReward(address(0)) {
        require(rewardsDuration > 0, "StakingRewards: duration not set");

        if (block.timestamp >= periodFinish) {
            rewardRate = reward / rewardsDuration;
        } else {
            uint256 remaining = periodFinish - block.timestamp;
            uint256 leftover = remaining * rewardRate;
            rewardRate = (reward + leftover) / rewardsDuration;
        }

        // Ensure the reward rate is not zero and that rewards do not exceed balance
        require(rewardRate > 0, "StakingRewards: zero reward rate");

        lastUpdateTime = block.timestamp;
        periodFinish = block.timestamp + rewardsDuration;

        emit RewardNotified(reward);
    }

    /// @notice Set the reward duration for future reward periods.
    /// @param _duration Duration in seconds.
    function setRewardsDuration(uint256 _duration) external onlyOwner {
        require(
            block.timestamp > periodFinish,
            "StakingRewards: previous period not finished"
        );
        require(_duration > 0, "StakingRewards: zero duration");
        rewardsDuration = _duration;
        emit RewardsDurationUpdated(_duration);
    }

    // ──────────────────── Internal ────────────────────

    function _safeTransfer(address token, address to, uint256 amount) internal {
        (bool success, bytes memory data) = token.call(
            abi.encodeWithSignature("transfer(address,uint256)", to, amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "StakingRewards: transfer failed");
    }

    function _safeTransferFrom(address token, address from, address to, uint256 amount) internal {
        (bool success, bytes memory data) = token.call(
            abi.encodeWithSignature("transferFrom(address,address,uint256)", from, to, amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "StakingRewards: transferFrom failed");
    }
}
