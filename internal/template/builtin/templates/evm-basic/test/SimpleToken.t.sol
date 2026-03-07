// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../contracts/SimpleToken.sol";

contract SimpleTokenTest {
    SimpleToken public token;
    address constant ALICE = address(0x1);
    address constant BOB = address(0x2);

    function setUp() public {
        token = new SimpleToken("Test Token", "TEST", 1000000);
    }

    function testName() public view {
        require(keccak256(bytes(token.name())) == keccak256(bytes("Test Token")), "wrong name");
    }

    function testInitialSupply() public view {
        require(token.totalSupply() == 1000000 * 1e18, "wrong supply");
    }

    function testTransfer() public {
        token.transfer(ALICE, 1000 * 1e18);
        require(token.balanceOf(ALICE) == 1000 * 1e18, "wrong balance");
    }

    function testApproveAndTransferFrom() public {
        token.approve(ALICE, 500 * 1e18);
        require(token.allowance(address(this), ALICE) == 500 * 1e18, "wrong allowance");
    }
}
