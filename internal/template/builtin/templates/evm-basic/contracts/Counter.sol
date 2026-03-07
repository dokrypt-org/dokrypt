// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title Counter - A simple counter contract
/// @notice Demonstrates basic state management in Solidity
contract Counter {
    uint256 public number;

    event NumberChanged(uint256 newNumber);

    function setNumber(uint256 newNumber) public {
        number = newNumber;
        emit NumberChanged(newNumber);
    }

    function increment() public {
        number++;
        emit NumberChanged(number);
    }

    function decrement() public {
        require(number > 0, "Counter: cannot decrement below zero");
        number--;
        emit NumberChanged(number);
    }

    function reset() public {
        number = 0;
        emit NumberChanged(0);
    }
}
