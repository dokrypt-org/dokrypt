// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../../contracts/token/DeFiToken.sol";
import "../../contracts/amm/Factory.sol";

/// @title FactoryTest
/// @notice Foundry-style tests for the AMM Factory: pair creation and duplicate prevention.
contract FactoryTest {
    Factory factory;
    DeFiToken tokenA;
    DeFiToken tokenB;
    DeFiToken tokenC;

    function setUp() public {
        factory = new Factory();
        tokenA = new DeFiToken("Token A", "TKA", 0);
        tokenB = new DeFiToken("Token B", "TKB", 0);
        tokenC = new DeFiToken("Token C", "TKC", 0);
    }

    function testCreatePair() public {
        address pair = factory.createPair(address(tokenA), address(tokenB));

        require(pair != address(0), "Pair should not be zero address");
        require(factory.getPair(address(tokenA), address(tokenB)) == pair, "getPair(A,B) mismatch");
        require(factory.getPair(address(tokenB), address(tokenA)) == pair, "getPair(B,A) mismatch");
        require(factory.allPairsLength() == 1, "Should have exactly 1 pair");
    }

    function testCreateMultiplePairs() public {
        address pair1 = factory.createPair(address(tokenA), address(tokenB));
        address pair2 = factory.createPair(address(tokenA), address(tokenC));

        require(pair1 != pair2, "Pairs should have different addresses");
        require(factory.allPairsLength() == 2, "Should have exactly 2 pairs");
    }

    function testDuplicatePairPrevention() public {
        factory.createPair(address(tokenA), address(tokenB));

        // Attempt to create the same pair again (should revert)
        bool reverted = false;
        try factory.createPair(address(tokenA), address(tokenB)) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on duplicate pair");

        // Also try reversed order
        reverted = false;
        try factory.createPair(address(tokenB), address(tokenA)) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on duplicate pair (reversed)");
    }

    function testIdenticalAddressesPrevention() public {
        bool reverted = false;
        try factory.createPair(address(tokenA), address(tokenA)) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on identical addresses");
    }

    function testZeroAddressPrevention() public {
        bool reverted = false;
        try factory.createPair(address(0), address(tokenA)) {
            // should not succeed
        } catch {
            reverted = true;
        }
        require(reverted, "Should revert on zero address");
    }
}
