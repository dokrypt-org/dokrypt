// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../../contracts/token/ManagedToken.sol";
import "../../contracts/vesting/VestingSchedule.sol";

contract VestingScheduleTest is Test {
    ManagedToken    token;
    VestingSchedule vesting;

    address owner;
    address beneficiary;

    uint256 constant TOTAL     = 10_000 ether;
    uint256 constant CLIFF     = 30 days;
    uint256 constant DURATION  = 365 days;

    function setUp() public {
        owner       = address(this);
        beneficiary = makeAddr("beneficiary");

        token   = new ManagedToken("VestToken", "VT", TOTAL);
        vesting = new VestingSchedule(address(token));

        // Approve vesting contract to pull tokens
        token.approve(address(vesting), TOTAL);
    }

    // ---- Create Schedule ----

    function test_createSchedule() public {
        uint256 id = vesting.createSchedule(
            beneficiary, TOTAL, block.timestamp, CLIFF, DURATION
        );
        assertEq(id, 0);
        assertEq(vesting.scheduleCount(), 1);
    }

    function test_createSchedule_zeroAmount() public {
        vm.expectRevert("VestingSchedule: amount is zero");
        vesting.createSchedule(beneficiary, 0, block.timestamp, CLIFF, DURATION);
    }

    // ---- Cliff ----

    function test_noReleaseBeforeCliff() public {
        vesting.createSchedule(beneficiary, TOTAL, block.timestamp, CLIFF, DURATION);

        // Warp to just before the cliff
        vm.warp(block.timestamp + CLIFF - 1);
        assertEq(vesting.vestedAmount(0), 0);

        vm.expectRevert("VestingSchedule: no tokens to release");
        vesting.release(0);
    }

    function test_releaseAfterCliff() public {
        uint256 start = block.timestamp;
        vesting.createSchedule(beneficiary, TOTAL, start, CLIFF, DURATION);

        // Warp to exactly after the cliff
        vm.warp(start + CLIFF);
        uint256 expected = (TOTAL * CLIFF) / DURATION;
        assertEq(vesting.vestedAmount(0), expected);

        vesting.release(0);
        assertEq(token.balanceOf(beneficiary), expected);
    }

    // ---- Full Vesting ----

    function test_fullVesting() public {
        uint256 start = block.timestamp;
        vesting.createSchedule(beneficiary, TOTAL, start, CLIFF, DURATION);

        vm.warp(start + DURATION);
        assertEq(vesting.vestedAmount(0), TOTAL);

        vesting.release(0);
        assertEq(token.balanceOf(beneficiary), TOTAL);
    }

    // ---- Revoke ----

    function test_revoke() public {
        uint256 start = block.timestamp;
        vesting.createSchedule(beneficiary, TOTAL, start, CLIFF, DURATION);

        // Warp to 50% of duration
        vm.warp(start + DURATION / 2);

        uint256 ownerBefore = token.balanceOf(owner);
        vesting.revoke(0);

        // Beneficiary should have received vested portion
        uint256 vested = token.balanceOf(beneficiary);
        assertTrue(vested > 0);

        // Owner should have received unvested portion
        uint256 ownerAfter = token.balanceOf(owner);
        assertTrue(ownerAfter > ownerBefore);

        // Total should equal TOTAL
        assertEq(vested + (ownerAfter - ownerBefore), TOTAL);
    }

    function test_revoke_onlyOwner() public {
        vesting.createSchedule(beneficiary, TOTAL, block.timestamp, CLIFF, DURATION);

        vm.prank(beneficiary);
        vm.expectRevert("VestingSchedule: caller is not the owner");
        vesting.revoke(0);
    }
}
