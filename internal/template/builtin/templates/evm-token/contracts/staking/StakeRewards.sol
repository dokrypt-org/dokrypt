// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title StakeRewards
/// @notice Simple staking contract that distributes ERC-20 reward tokens
///         to stakers proportional to their share of the pool.
interface IERC20Stake {
    function transfer(address to, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

contract StakeRewards {
    // ----------------------------------------------------------------
    // State
    // ----------------------------------------------------------------
    IERC20Stake public stakingToken;
    IERC20Stake public rewardsToken;
    address     public owner;

    uint256 public rewardRate;          // tokens per second
    uint256 public rewardsDuration;     // length of current reward period (seconds)
    uint256 public periodFinish;        // timestamp when current period ends
    uint256 public lastUpdateTime;
    uint256 public rewardPerTokenStored;

    uint256 public totalStaked;

    mapping(address => uint256) public stakedBalance;
    mapping(address => uint256) public userRewardPerTokenPaid;
    mapping(address => uint256) public rewards;

    // ----------------------------------------------------------------
    // Events
    // ----------------------------------------------------------------
    event Staked(address indexed user, uint256 amount);
    event Withdrawn(address indexed user, uint256 amount);
    event RewardPaid(address indexed user, uint256 reward);
    event RewardsDurationUpdated(uint256 newDuration);

    // ----------------------------------------------------------------
    // Modifiers
    // ----------------------------------------------------------------
    modifier onlyOwner() {
        require(msg.sender == owner, "StakeRewards: caller is not the owner");
        _;
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

    // ----------------------------------------------------------------
    // Constructor
    // ----------------------------------------------------------------
    constructor(address _stakingToken, address _rewardsToken) {
        require(_stakingToken != address(0), "StakeRewards: staking token is zero address");
        require(_rewardsToken != address(0), "StakeRewards: rewards token is zero address");
        stakingToken = IERC20Stake(_stakingToken);
        rewardsToken = IERC20Stake(_rewardsToken);
        owner = msg.sender;
    }

    // ----------------------------------------------------------------
    // Core Functions
    // ----------------------------------------------------------------

    function stake(uint256 amount) external updateReward(msg.sender) {
        require(amount > 0, "StakeRewards: cannot stake zero");
        totalStaked += amount;
        stakedBalance[msg.sender] += amount;
        bool ok = stakingToken.transferFrom(msg.sender, address(this), amount);
        require(ok, "StakeRewards: stake transfer failed");
        emit Staked(msg.sender, amount);
    }

    function withdraw(uint256 amount) external updateReward(msg.sender) {
        require(amount > 0, "StakeRewards: cannot withdraw zero");
        require(stakedBalance[msg.sender] >= amount, "StakeRewards: insufficient staked balance");
        totalStaked -= amount;
        stakedBalance[msg.sender] -= amount;
        bool ok = stakingToken.transfer(msg.sender, amount);
        require(ok, "StakeRewards: withdraw transfer failed");
        emit Withdrawn(msg.sender, amount);
    }

    function getReward() external updateReward(msg.sender) {
        uint256 reward = rewards[msg.sender];
        if (reward > 0) {
            rewards[msg.sender] = 0;
            bool ok = rewardsToken.transfer(msg.sender, reward);
            require(ok, "StakeRewards: reward transfer failed");
            emit RewardPaid(msg.sender, reward);
        }
    }

    // ----------------------------------------------------------------
    // Owner: configure rewards
    // ----------------------------------------------------------------

    function setRewardsDuration(uint256 _duration) external onlyOwner {
        require(block.timestamp >= periodFinish, "StakeRewards: previous period not finished");
        rewardsDuration = _duration;
        emit RewardsDurationUpdated(_duration);
    }

    /// @notice Notify the contract of a new reward amount. Tokens must already
    ///         have been transferred to this contract before calling.
    function notifyRewardAmount(uint256 reward) external onlyOwner updateReward(address(0)) {
        require(rewardsDuration > 0, "StakeRewards: rewards duration not set");

        if (block.timestamp >= periodFinish) {
            rewardRate = reward / rewardsDuration;
        } else {
            uint256 remaining = periodFinish - block.timestamp;
            uint256 leftover  = remaining * rewardRate;
            rewardRate = (reward + leftover) / rewardsDuration;
        }

        lastUpdateTime = block.timestamp;
        periodFinish   = block.timestamp + rewardsDuration;
    }

    // ----------------------------------------------------------------
    // View Helpers
    // ----------------------------------------------------------------

    function lastTimeRewardApplicable() public view returns (uint256) {
        return block.timestamp < periodFinish ? block.timestamp : periodFinish;
    }

    function rewardPerToken() public view returns (uint256) {
        if (totalStaked == 0) {
            return rewardPerTokenStored;
        }
        return rewardPerTokenStored +
            ((lastTimeRewardApplicable() - lastUpdateTime) * rewardRate * 1e18) / totalStaked;
    }

    function earned(address account) public view returns (uint256) {
        return
            (stakedBalance[account] * (rewardPerToken() - userRewardPerTokenPaid[account])) / 1e18
            + rewards[account];
    }
}
