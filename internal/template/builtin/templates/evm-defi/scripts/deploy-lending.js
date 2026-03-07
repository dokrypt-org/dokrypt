// deploy-lending.js
// Deploys the Lending stack: InterestModel + PriceOracle + LendingVault
//
// Usage:
//   npx hardhat run scripts/deploy-lending.js --network localhost
//   or with forge:
//   forge script scripts/deploy-lending.js

const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying Lending contracts with account:", deployer.address);
  console.log("Account balance:", (await deployer.provider.getBalance(deployer.address)).toString());

  // 1. Deploy a borrow token (or use an existing one)
  console.log("\n--- Deploying Borrow Token ---");
  const DeFiToken = await ethers.getContractFactory("DeFiToken");
  const borrowToken = await DeFiToken.deploy("USD Stablecoin", "USDT", ethers.parseEther("10000000"));
  await borrowToken.waitForDeployment();
  const borrowTokenAddress = await borrowToken.getAddress();
  console.log("Borrow Token deployed to:", borrowTokenAddress);

  // 2. Deploy a collateral token
  console.log("\n--- Deploying Collateral Token ---");
  const collateralToken = await DeFiToken.deploy("Wrapped ETH", "WETH", ethers.parseEther("10000000"));
  await collateralToken.waitForDeployment();
  const collateralTokenAddress = await collateralToken.getAddress();
  console.log("Collateral Token deployed to:", collateralTokenAddress);

  // 3. Deploy InterestModel
  console.log("\n--- Deploying InterestModel ---");
  const InterestModel = await ethers.getContractFactory("InterestModel");
  const interestModel = await InterestModel.deploy();
  await interestModel.waitForDeployment();
  const interestModelAddress = await interestModel.getAddress();
  console.log("InterestModel deployed to:", interestModelAddress);

  // 4. Deploy PriceOracle
  console.log("\n--- Deploying PriceOracle ---");
  const PriceOracle = await ethers.getContractFactory("PriceOracle");
  const oracle = await PriceOracle.deploy();
  await oracle.waitForDeployment();
  const oracleAddress = await oracle.getAddress();
  console.log("PriceOracle deployed to:", oracleAddress);

  // 5. Deploy LendingVault
  console.log("\n--- Deploying LendingVault ---");
  const LendingVault = await ethers.getContractFactory("LendingVault");
  const vault = await LendingVault.deploy(borrowTokenAddress, interestModelAddress, oracleAddress);
  await vault.waitForDeployment();
  const vaultAddress = await vault.getAddress();
  console.log("LendingVault deployed to:", vaultAddress);

  // 6. Configure: set prices and add collateral
  console.log("\n--- Configuring ---");
  await (await oracle.setPrice(borrowTokenAddress, ethers.parseEther("1"))).wait();
  console.log("Borrow token price set to $1.00");

  await (await oracle.setPrice(collateralTokenAddress, ethers.parseEther("2000"))).wait();
  console.log("Collateral token price set to $2000.00");

  await (await vault.addCollateral(collateralTokenAddress)).wait();
  console.log("Collateral token added to vault");

  // 7. Supply borrow tokens to vault
  console.log("\n--- Supplying Borrow Tokens ---");
  const supplyAmount = ethers.parseEther("1000000");
  await (await borrowToken.approve(vaultAddress, ethers.MaxUint256)).wait();
  await (await vault.supply(supplyAmount)).wait();
  console.log("Supplied", ethers.formatEther(supplyAmount), "borrow tokens to vault");

  // Summary
  console.log("\n========== Deployment Summary ==========");
  console.log("Borrow Token:   ", borrowTokenAddress);
  console.log("Collateral Token:", collateralTokenAddress);
  console.log("InterestModel:  ", interestModelAddress);
  console.log("PriceOracle:    ", oracleAddress);
  console.log("LendingVault:   ", vaultAddress);
  console.log("=========================================");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Deployment failed:", error);
    process.exit(1);
  });
