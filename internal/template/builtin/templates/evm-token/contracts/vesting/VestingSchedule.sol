// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title VestingSchedule
/// @notice Linear token vesting with cliff for ERC-20 tokens.
interface IERC20Minimal {
    function transfer(address to, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

contract VestingSchedule {
    // ----------------------------------------------------------------
    // Types
    // ----------------------------------------------------------------
    struct Schedule {
        address beneficiary;
        uint256 totalAmount;
        uint256 released;
        uint256 start;
        uint256 cliff;       // absolute timestamp when cliff ends
        uint256 duration;    // total vesting duration in seconds (from start)
        bool    revoked;
    }

    // ----------------------------------------------------------------
    // State
    // ----------------------------------------------------------------
    IERC20Minimal public token;
    address       public owner;

    uint256 public scheduleCount;
    mapping(uint256 => Schedule) public schedules;

    // ----------------------------------------------------------------
    // Events
    // ----------------------------------------------------------------
    event ScheduleCreated(
        uint256 indexed scheduleId,
        address indexed beneficiary,
        uint256 amount,
        uint256 start,
        uint256 cliff,
        uint256 duration
    );
    event TokensReleased(uint256 indexed scheduleId, address indexed beneficiary, uint256 amount);
    event ScheduleRevoked(uint256 indexed scheduleId, uint256 unvestedReturned);

    // ----------------------------------------------------------------
    // Modifiers
    // ----------------------------------------------------------------
    modifier onlyOwner() {
        require(msg.sender == owner, "VestingSchedule: caller is not the owner");
        _;
    }

    // ----------------------------------------------------------------
    // Constructor
    // ----------------------------------------------------------------
    constructor(address _token) {
        require(_token != address(0), "VestingSchedule: token is zero address");
        token = IERC20Minimal(_token);
        owner = msg.sender;
    }

    // ----------------------------------------------------------------
    // Schedule Management
    // ----------------------------------------------------------------

    /// @notice Create a new vesting schedule.
    /// @param beneficiary Address that will receive vested tokens.
    /// @param amount      Total number of tokens to vest.
    /// @param start       Unix timestamp when vesting starts.
    /// @param cliffDuration Seconds after start before any tokens vest.
    /// @param duration    Total vesting duration in seconds (must be >= cliffDuration).
    function createSchedule(
        address beneficiary,
        uint256 amount,
        uint256 start,
        uint256 cliffDuration,
        uint256 duration
    ) external onlyOwner returns (uint256 scheduleId) {
        require(beneficiary != address(0), "VestingSchedule: beneficiary is zero address");
        require(amount > 0,                "VestingSchedule: amount is zero");
        require(duration > 0,              "VestingSchedule: duration is zero");
        require(duration >= cliffDuration,  "VestingSchedule: cliff exceeds duration");

        // Transfer tokens into this contract to be held for vesting
        bool ok = token.transferFrom(msg.sender, address(this), amount);
        require(ok, "VestingSchedule: token transfer failed");

        scheduleId = scheduleCount;
        schedules[scheduleId] = Schedule({
            beneficiary: beneficiary,
            totalAmount: amount,
            released:    0,
            start:       start,
            cliff:       start + cliffDuration,
            duration:    duration,
            revoked:     false
        });
        scheduleCount++;

        emit ScheduleCreated(scheduleId, beneficiary, amount, start, cliffDuration, duration);
    }

    /// @notice Release vested tokens to the beneficiary.
    function release(uint256 scheduleId) external {
        Schedule storage s = schedules[scheduleId];
        require(s.beneficiary != address(0), "VestingSchedule: schedule does not exist");
        require(!s.revoked,                  "VestingSchedule: schedule has been revoked");

        uint256 vested = _vestedAmount(s);
        uint256 releasable = vested - s.released;
        require(releasable > 0, "VestingSchedule: no tokens to release");

        s.released += releasable;
        bool ok = token.transfer(s.beneficiary, releasable);
        require(ok, "VestingSchedule: token transfer failed");

        emit TokensReleased(scheduleId, s.beneficiary, releasable);
    }

    /// @notice Revoke a schedule and return unvested tokens to the owner.
    function revoke(uint256 scheduleId) external onlyOwner {
        Schedule storage s = schedules[scheduleId];
        require(s.beneficiary != address(0), "VestingSchedule: schedule does not exist");
        require(!s.revoked,                  "VestingSchedule: schedule already revoked");

        uint256 vested     = _vestedAmount(s);
        uint256 releasable = vested - s.released;
        uint256 unvested   = s.totalAmount - vested;

        s.revoked = true;

        // Release any vested-but-unclaimed tokens to beneficiary
        if (releasable > 0) {
            s.released += releasable;
            bool ok1 = token.transfer(s.beneficiary, releasable);
            require(ok1, "VestingSchedule: transfer to beneficiary failed");
        }

        // Return unvested tokens to owner
        if (unvested > 0) {
            bool ok2 = token.transfer(owner, unvested);
            require(ok2, "VestingSchedule: transfer to owner failed");
        }

        emit ScheduleRevoked(scheduleId, unvested);
    }

    // ----------------------------------------------------------------
    // View Functions
    // ----------------------------------------------------------------

    /// @notice Calculate the amount of tokens that have vested so far.
    function vestedAmount(uint256 scheduleId) external view returns (uint256) {
        Schedule storage s = schedules[scheduleId];
        return _vestedAmount(s);
    }

    /// @notice Calculate the amount of tokens available to release now.
    function releasableAmount(uint256 scheduleId) external view returns (uint256) {
        Schedule storage s = schedules[scheduleId];
        return _vestedAmount(s) - s.released;
    }

    // ----------------------------------------------------------------
    // Internal
    // ----------------------------------------------------------------

    function _vestedAmount(Schedule storage s) internal view returns (uint256) {
        if (block.timestamp < s.cliff) {
            return 0;
        } else if (block.timestamp >= s.start + s.duration || s.revoked) {
            return s.totalAmount;
        } else {
            return (s.totalAmount * (block.timestamp - s.start)) / s.duration;
        }
    }
}
