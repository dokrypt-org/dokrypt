/**
 * deploy-dao.js
 *
 * Deploys the complete DAO governance stack:
 *   1. VotingToken  - ERC-20 with delegation & checkpoints
 *   2. Timelock     - Delay controller for governance actions
 *   3. Governor     - Proposal / voting engine
 *   4. Treasury     - Holds DAO funds, controlled by Timelock
 *
 * Usage:
 *   npx hardhat run scripts/deploy-dao.js --network <network>
 *   # or with plain ethers / node:
 *   node scripts/deploy-dao.js
 */

const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying DAO with account:", deployer.address);
  console.log("Account balance:", (await deployer.getBalance()).toString());

  // ------------------------------------------------------------------
  // 1. Deploy VotingToken
  // ------------------------------------------------------------------
  const initialSupply = ethers.utils.parseEther("1000000"); // 1 million tokens
  const VotingToken = await ethers.getContractFactory("VotingToken");
  const token = await VotingToken.deploy("DAO Governance Token", "GOV", initialSupply);
  await token.deployed();
  console.log("VotingToken deployed to:", token.address);

  // ------------------------------------------------------------------
  // 2. Deploy Timelock
  // ------------------------------------------------------------------
  const minDelay = 86400; // 1 day in seconds
  const proposers = [deployer.address];
  const executors = [deployer.address];

  const Timelock = await ethers.getContractFactory("Timelock");
  const timelock = await Timelock.deploy(minDelay, proposers, executors);
  await timelock.deployed();
  console.log("Timelock deployed to:", timelock.address);

  // ------------------------------------------------------------------
  // 3. Deploy Governor
  // ------------------------------------------------------------------
  const votingDelay = 1;           // 1 block
  const votingPeriod = 45818;      // ~1 week at 12s blocks
  const quorum = ethers.utils.parseEther("10000"); // 10k tokens
  const proposalThreshold = ethers.utils.parseEther("100"); // 100 tokens to propose

  const Governor = await ethers.getContractFactory("Governor");
  const governor = await Governor.deploy(
    token.address,
    timelock.address,
    votingDelay,
    votingPeriod,
    quorum,
    proposalThreshold
  );
  await governor.deployed();
  console.log("Governor deployed to:", governor.address);

  // ------------------------------------------------------------------
  // 4. Deploy Treasury
  // ------------------------------------------------------------------
  const Treasury = await ethers.getContractFactory("Treasury");
  const treasury = await Treasury.deploy(timelock.address);
  await treasury.deployed();
  console.log("Treasury deployed to:", treasury.address);

  // ------------------------------------------------------------------
  // 5. Configure roles
  // ------------------------------------------------------------------

  // Grant Governor the proposer role on the Timelock
  const addProposerTx = await timelock.addProposer(governor.address);
  await addProposerTx.wait();
  console.log("Governor granted proposer role on Timelock");

  // Grant Governor the executor role on the Timelock
  const addExecutorTx = await timelock.addExecutor(governor.address);
  await addExecutorTx.wait();
  console.log("Governor granted executor role on Timelock");

  // Self-delegate so deployer can participate in governance
  const delegateTx = await token.delegate(deployer.address);
  await delegateTx.wait();
  console.log("Deployer self-delegated voting power");

  // ------------------------------------------------------------------
  // Summary
  // ------------------------------------------------------------------
  console.log("\n========================================");
  console.log("  DAO Deployment Complete");
  console.log("========================================");
  console.log("VotingToken :", token.address);
  console.log("Timelock    :", timelock.address);
  console.log("Governor    :", governor.address);
  console.log("Treasury    :", treasury.address);
  console.log("========================================\n");

  return { token, timelock, governor, treasury };
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Deployment failed:", error);
    process.exit(1);
  });
