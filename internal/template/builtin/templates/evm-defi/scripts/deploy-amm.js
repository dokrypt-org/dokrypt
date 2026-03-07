// deploy-amm.js
// Deploys the AMM stack: Factory + Router
//
// Usage:
//   npx hardhat run scripts/deploy-amm.js --network localhost
//   or with forge:
//   forge script scripts/deploy-amm.js

const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying AMM contracts with account:", deployer.address);
  console.log("Account balance:", (await deployer.provider.getBalance(deployer.address)).toString());

  // 1. Deploy Factory
  console.log("\n--- Deploying Factory ---");
  const Factory = await ethers.getContractFactory("Factory");
  const factory = await Factory.deploy();
  await factory.waitForDeployment();
  const factoryAddress = await factory.getAddress();
  console.log("Factory deployed to:", factoryAddress);

  // 2. Deploy Router
  console.log("\n--- Deploying Router ---");
  const Router = await ethers.getContractFactory("Router");
  const router = await Router.deploy(factoryAddress);
  await router.waitForDeployment();
  const routerAddress = await router.getAddress();
  console.log("Router deployed to:", routerAddress);

  // 3. Optionally deploy test tokens and create a pair
  console.log("\n--- Deploying Test Tokens ---");
  const DeFiToken = await ethers.getContractFactory("DeFiToken");

  const tokenA = await DeFiToken.deploy("Token A", "TKA", ethers.parseEther("1000000"));
  await tokenA.waitForDeployment();
  const tokenAAddress = await tokenA.getAddress();
  console.log("Token A deployed to:", tokenAAddress);

  const tokenB = await DeFiToken.deploy("Token B", "TKB", ethers.parseEther("1000000"));
  await tokenB.waitForDeployment();
  const tokenBAddress = await tokenB.getAddress();
  console.log("Token B deployed to:", tokenBAddress);

  // 4. Create a pair
  console.log("\n--- Creating Pair ---");
  const tx = await factory.createPair(tokenAAddress, tokenBAddress);
  await tx.wait();
  const pairAddress = await factory.getPair(tokenAAddress, tokenBAddress);
  console.log("Pair created at:", pairAddress);

  // 5. Add initial liquidity
  console.log("\n--- Adding Initial Liquidity ---");
  const liquidityAmount = ethers.parseEther("10000");

  await (await tokenA.approve(routerAddress, ethers.MaxUint256)).wait();
  await (await tokenB.approve(routerAddress, ethers.MaxUint256)).wait();

  const deadline = Math.floor(Date.now() / 1000) + 3600;
  const addLiqTx = await router.addLiquidity(
    tokenAAddress,
    tokenBAddress,
    liquidityAmount,
    liquidityAmount,
    0,
    0,
    deployer.address,
    deadline
  );
  await addLiqTx.wait();
  console.log("Liquidity added successfully!");

  // Summary
  console.log("\n========== Deployment Summary ==========");
  console.log("Factory:  ", factoryAddress);
  console.log("Router:   ", routerAddress);
  console.log("Token A:  ", tokenAAddress);
  console.log("Token B:  ", tokenBAddress);
  console.log("Pair:     ", pairAddress);
  console.log("=========================================");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Deployment failed:", error);
    process.exit(1);
  });
