// deploy-staking.js
// Deploys the StakingRewards contract with configurable staking and reward tokens.
//
// Usage:
//   npx hardhat run scripts/deploy-staking.js --network localhost
//   or with forge:
//   forge script scripts/deploy-staking.js

const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying Staking contracts with account:", deployer.address);
  console.log("Account balance:", (await deployer.provider.getBalance(deployer.address)).toString());

  // 1. Deploy staking token
  console.log("\n--- Deploying Staking Token ---");
  const DeFiToken = await ethers.getContractFactory("DeFiToken");
  const stakingToken = await DeFiToken.deploy("LP Token", "LP", ethers.parseEther("1000000"));
  await stakingToken.waitForDeployment();
  const stakingTokenAddress = await stakingToken.getAddress();
  console.log("Staking Token deployed to:", stakingTokenAddress);

  // 2. Deploy reward token
  console.log("\n--- Deploying Reward Token ---");
  const rewardToken = await DeFiToken.deploy("Reward Token", "RWD", ethers.parseEther("1000000"));
  await rewardToken.waitForDeployment();
  const rewardTokenAddress = await rewardToken.getAddress();
  console.log("Reward Token deployed to:", rewardTokenAddress);

  // 3. Deploy StakingRewards
  console.log("\n--- Deploying StakingRewards ---");
  const StakingRewards = await ethers.getContractFactory("StakingRewards");
  const staking = await StakingRewards.deploy(stakingTokenAddress, rewardTokenAddress);
  await staking.waitForDeployment();
  const stakingAddress = await staking.getAddress();
  console.log("StakingRewards deployed to:", stakingAddress);

  // 4. Configure rewards
  console.log("\n--- Configuring Rewards ---");
  const rewardAmount = ethers.parseEther("100000");
  const duration = 30 * 24 * 60 * 60; // 30 days in seconds

  // Transfer rewards to staking contract
  await (await rewardToken.transfer(stakingAddress, rewardAmount)).wait();
  console.log("Transferred", ethers.formatEther(rewardAmount), "reward tokens to staking contract");

  // Set duration and notify
  await (await staking.setRewardsDuration(duration)).wait();
  console.log("Rewards duration set to 30 days");

  await (await staking.notifyRewardAmount(rewardAmount)).wait();
  console.log("Reward notification sent");

  const rewardRate = await staking.rewardRate();
  console.log("Reward rate:", ethers.formatEther(rewardRate), "tokens/second");

  // Summary
  console.log("\n========== Deployment Summary ==========");
  console.log("Staking Token:  ", stakingTokenAddress);
  console.log("Reward Token:   ", rewardTokenAddress);
  console.log("StakingRewards: ", stakingAddress);
  console.log("Reward Amount:  ", ethers.formatEther(rewardAmount), "tokens");
  console.log("Duration:       ", duration / 86400, "days");
  console.log("=========================================");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Deployment failed:", error);
    process.exit(1);
  });
