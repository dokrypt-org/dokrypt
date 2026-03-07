// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title InterestModel
/// @notice Linear interest rate model based on utilization rate.
/// As utilization increases, the interest rate rises linearly from a base rate
/// up to a maximum rate, with a kink at a target utilization threshold.
contract InterestModel {
    uint256 public constant BASE_RATE = 2e16;      // 2% base annual rate (scaled by 1e18)
    uint256 public constant RATE_SLOPE_1 = 10e16;  // 10% slope below kink
    uint256 public constant RATE_SLOPE_2 = 100e16; // 100% slope above kink
    uint256 public constant OPTIMAL_UTILIZATION = 80e16; // 80% target utilization (scaled by 1e18)

    event InterestRateQueried(uint256 utilized, uint256 total, uint256 rate);

    /// @notice Calculate the current interest rate based on utilization.
    /// @param utilized Total amount currently borrowed.
    /// @param total Total amount available (deposited).
    /// @return rate The annual interest rate scaled by 1e18.
    function getInterestRate(uint256 utilized, uint256 total) external pure returns (uint256 rate) {
        if (total == 0) {
            return BASE_RATE;
        }

        uint256 utilization = (utilized * 1e18) / total;

        if (utilization <= OPTIMAL_UTILIZATION) {
            // Linear increase from BASE_RATE to BASE_RATE + RATE_SLOPE_1
            rate = BASE_RATE + (RATE_SLOPE_1 * utilization) / OPTIMAL_UTILIZATION;
        } else {
            // Base rate at kink point
            uint256 rateAtKink = BASE_RATE + RATE_SLOPE_1;
            // Excess utilization above kink
            uint256 excessUtilization = utilization - OPTIMAL_UTILIZATION;
            uint256 maxExcess = 1e18 - OPTIMAL_UTILIZATION; // 20%
            rate = rateAtKink + (RATE_SLOPE_2 * excessUtilization) / maxExcess;
        }
    }

    /// @notice Get the utilization rate.
    /// @param utilized Total amount currently borrowed.
    /// @param total Total amount available (deposited).
    /// @return The utilization rate scaled by 1e18.
    function getUtilizationRate(uint256 utilized, uint256 total) external pure returns (uint256) {
        if (total == 0) return 0;
        return (utilized * 1e18) / total;
    }
}
