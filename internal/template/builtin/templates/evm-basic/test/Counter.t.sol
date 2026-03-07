// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../contracts/Counter.sol";

contract CounterTest {
    Counter public counter;

    function setUp() public {
        counter = new Counter();
    }

    function testInitialValue() public view {
        require(counter.number() == 0, "initial value should be 0");
    }

    function testIncrement() public {
        counter.increment();
        require(counter.number() == 1, "should be 1 after increment");
    }

    function testSetNumber() public {
        counter.setNumber(42);
        require(counter.number() == 42, "should be 42");
    }

    function testDecrement() public {
        counter.setNumber(5);
        counter.decrement();
        require(counter.number() == 4, "should be 4 after decrement");
    }

    function testReset() public {
        counter.setNumber(100);
        counter.reset();
        require(counter.number() == 0, "should be 0 after reset");
    }
}
