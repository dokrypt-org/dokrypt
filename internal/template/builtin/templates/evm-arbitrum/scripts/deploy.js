// deploy.js
// Deploys the Arbitrum L2 stack: ArbitrumToken, L1ToL2MessageSender,
// L2ToL1MessageSender, and TokenGateway.
//
// Usage:
//   npx hardhat run scripts/deploy.js --network localhost
//   or with forge:
//   forge script scripts/deploy.js

const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying Arbitrum contracts with account:", deployer.address);
  console.log("Account balance:", (await deployer.provider.getBalance(deployer.address)).toString());

  // 1. Deploy ArbitrumToken
  console.log("\n--- Deploying ArbitrumToken ---");
  const ArbitrumToken = await ethers.getContractFactory("ArbitrumToken");
  const token = await ArbitrumToken.deploy("Arbitrum Token", "ARB", ethers.parseEther("1000000"));
  await token.waitForDeployment();
  const tokenAddress = await token.getAddress();
  console.log("ArbitrumToken deployed to:", tokenAddress);

  // 2. Deploy L1ToL2MessageSender (using deployer as mock inbox for local testing)
  console.log("\n--- Deploying L1ToL2MessageSender ---");
  const L1ToL2MessageSender = await ethers.getContractFactory("L1ToL2MessageSender");
  const l1ToL2 = await L1ToL2MessageSender.deploy(deployer.address);
  await l1ToL2.waitForDeployment();
  const l1ToL2Address = await l1ToL2.getAddress();
  console.log("L1ToL2MessageSender deployed to:", l1ToL2Address);

  // 3. Deploy L2ToL1MessageSender
  console.log("\n--- Deploying L2ToL1MessageSender ---");
  const L2ToL1MessageSender = await ethers.getContractFactory("L2ToL1MessageSender");
  const l2ToL1 = await L2ToL1MessageSender.deploy();
  await l2ToL1.waitForDeployment();
  const l2ToL1Address = await l2ToL1.getAddress();
  console.log("L2ToL1MessageSender deployed to:", l2ToL1Address);

  // 4. Deploy L1 TokenGateway (simulated)
  console.log("\n--- Deploying L1 TokenGateway ---");
  const TokenGateway = await ethers.getContractFactory("TokenGateway");
  const l1Gateway = await TokenGateway.deploy(ethers.ZeroAddress); // counterpart set later
  await l1Gateway.waitForDeployment();
  const l1GatewayAddress = await l1Gateway.getAddress();
  console.log("L1 TokenGateway deployed to:", l1GatewayAddress);

  // 5. Deploy L2 TokenGateway
  console.log("\n--- Deploying L2 TokenGateway ---");
  const l2Gateway = await TokenGateway.deploy(l1GatewayAddress);
  await l2Gateway.waitForDeployment();
  const l2GatewayAddress = await l2Gateway.getAddress();
  console.log("L2 TokenGateway deployed to:", l2GatewayAddress);

  // 6. Configure: link gateways and set token mappings
  console.log("\n--- Configuring Gateways ---");
  await (await l1Gateway.setCounterpartGateway(l2GatewayAddress)).wait();
  console.log("L1 Gateway counterpart set to L2 Gateway");

  // Deploy an L2 representation token
  const l2Token = await ArbitrumToken.deploy("Arbitrum Token (L2)", "ARB-L2", 0);
  await l2Token.waitForDeployment();
  const l2TokenAddress = await l2Token.getAddress();
  console.log("L2 ArbitrumToken deployed to:", l2TokenAddress);

  // Set token mapping
  await (await l1Gateway.setTokenMapping(tokenAddress, l2TokenAddress)).wait();
  await (await l2Gateway.setTokenMapping(tokenAddress, l2TokenAddress)).wait();
  console.log("Token mapping configured:", tokenAddress, "<->", l2TokenAddress);

  // Authorize L2 gateway to mint L2 tokens
  await (await l2Token.setGateway(l2GatewayAddress)).wait();
  console.log("L2 Gateway authorized to mint L2 tokens");

  // Allow L1 target in L2ToL1MessageSender
  await (await l2ToL1.setAllowedL1Target(l1GatewayAddress)).wait();
  console.log("L1 Gateway allowed as L1 target for L2->L1 messages");

  // Summary
  console.log("\n========== Deployment Summary ==========");
  console.log("ArbitrumToken (L1):    ", tokenAddress);
  console.log("ArbitrumToken (L2):    ", l2TokenAddress);
  console.log("L1ToL2MessageSender:   ", l1ToL2Address);
  console.log("L2ToL1MessageSender:   ", l2ToL1Address);
  console.log("L1 TokenGateway:       ", l1GatewayAddress);
  console.log("L2 TokenGateway:       ", l2GatewayAddress);
  console.log("=========================================");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Deployment failed:", error);
    process.exit(1);
  });
