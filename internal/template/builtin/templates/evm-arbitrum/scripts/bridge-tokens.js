// bridge-tokens.js
// Demonstrates token bridging between L1 and L2 using the TokenGateway.
//
// Usage:
//   npx hardhat run scripts/bridge-tokens.js --network localhost
//
// Note: This script assumes the deploy.js script has already been run
// and the contract addresses are known. For local testing, it deploys
// fresh contracts.

const { ethers } = require("hardhat");

async function main() {
  const [deployer, user] = await ethers.getSigners();
  console.log("Bridge tokens script");
  console.log("Deployer:", deployer.address);
  console.log("User:", user.address);

  // Deploy fresh contracts for demonstration
  console.log("\n--- Setting up contracts ---");

  const ArbitrumToken = await ethers.getContractFactory("ArbitrumToken");
  const TokenGateway = await ethers.getContractFactory("TokenGateway");

  // L1 Token (the original token)
  const l1Token = await ArbitrumToken.deploy("Arbitrum Token", "ARB", ethers.parseEther("1000000"));
  await l1Token.waitForDeployment();
  const l1TokenAddress = await l1Token.getAddress();
  console.log("L1 Token:", l1TokenAddress);

  // L1 Gateway
  const l1Gateway = await TokenGateway.deploy(ethers.ZeroAddress);
  await l1Gateway.waitForDeployment();
  const l1GatewayAddress = await l1Gateway.getAddress();

  // L2 Gateway
  const l2Gateway = await TokenGateway.deploy(l1GatewayAddress);
  await l2Gateway.waitForDeployment();
  const l2GatewayAddress = await l2Gateway.getAddress();

  // Link L1 gateway to L2
  await (await l1Gateway.setCounterpartGateway(l2GatewayAddress)).wait();

  // L2 Token (bridged representation)
  const l2Token = await ArbitrumToken.deploy("Arbitrum Token (L2)", "ARB-L2", 0);
  await l2Token.waitForDeployment();
  const l2TokenAddress = await l2Token.getAddress();
  console.log("L2 Token:", l2TokenAddress);

  // Configure mappings
  await (await l1Gateway.setTokenMapping(l1TokenAddress, l2TokenAddress)).wait();
  await (await l2Gateway.setTokenMapping(l1TokenAddress, l2TokenAddress)).wait();
  await (await l2Token.setGateway(l2GatewayAddress)).wait();

  console.log("Gateways configured");

  // --- Demonstrate L1 -> L2 Deposit ---
  console.log("\n========== L1 -> L2 Deposit ==========");

  const depositAmount = ethers.parseEther("1000");

  // Transfer some L1 tokens to user
  await (await l1Token.transfer(user.address, depositAmount)).wait();
  console.log("Transferred", ethers.formatEther(depositAmount), "L1 tokens to user");

  // User approves L1 gateway
  await (await l1Token.connect(user).approve(l1GatewayAddress, depositAmount)).wait();
  console.log("User approved L1 gateway");

  // User deposits to L1 gateway
  const depositTx = await l1Gateway.connect(user).deposit(l1TokenAddress, user.address, depositAmount);
  const depositReceipt = await depositTx.wait();
  console.log("Deposit transaction confirmed, gas used:", depositReceipt.gasUsed.toString());

  // Check L1 gateway locked balance
  const locked = await l1Gateway.getLockedBalance(l1TokenAddress);
  console.log("L1 Gateway locked balance:", ethers.formatEther(locked));

  // Simulate L2 finalization (in production, this happens via the Arbitrum bridge)
  console.log("\n--- Simulating L2 finalization ---");
  await (await l2Gateway.finalizeDeposit(0, l2TokenAddress, user.address, depositAmount)).wait();

  const l2Balance = await l2Token.balanceOf(user.address);
  console.log("User L2 token balance:", ethers.formatEther(l2Balance));

  // --- Demonstrate L2 -> L1 Withdrawal ---
  console.log("\n========== L2 -> L1 Withdrawal ==========");

  const withdrawAmount = ethers.parseEther("500");

  // User approves L2 gateway for burn
  await (await l2Token.connect(user).approve(l2GatewayAddress, withdrawAmount)).wait();
  console.log("User approved L2 gateway for burn");

  // User withdraws from L2 gateway
  const withdrawTx = await l2Gateway.connect(user).withdraw(l2TokenAddress, user.address, withdrawAmount);
  const withdrawReceipt = await withdrawTx.wait();
  console.log("Withdrawal transaction confirmed, gas used:", withdrawReceipt.gasUsed.toString());

  // Check remaining L2 balance
  const l2BalanceAfter = await l2Token.balanceOf(user.address);
  console.log("User L2 token balance after withdrawal:", ethers.formatEther(l2BalanceAfter));

  // Simulate L1 finalization (in production, after ~7 day challenge period)
  console.log("\n--- Simulating L1 finalization ---");
  await (await l1Gateway.finalizeWithdrawal(0, l1TokenAddress, user.address, withdrawAmount)).wait();

  const l1BalanceAfter = await l1Token.balanceOf(user.address);
  console.log("User L1 token balance after withdrawal:", ethers.formatEther(l1BalanceAfter));

  // Summary
  console.log("\n========== Bridge Summary ==========");
  console.log("Deposited to L2:     ", ethers.formatEther(depositAmount), "tokens");
  console.log("Withdrawn to L1:     ", ethers.formatEther(withdrawAmount), "tokens");
  console.log("User L1 balance:     ", ethers.formatEther(l1BalanceAfter));
  console.log("User L2 balance:     ", ethers.formatEther(l2BalanceAfter));
  console.log("L1 Gateway locked:   ", ethers.formatEther(await l1Gateway.getLockedBalance(l1TokenAddress)));
  console.log("=====================================");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Bridge script failed:", error);
    process.exit(1);
  });
