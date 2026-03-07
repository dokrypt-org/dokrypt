// setup-vesting.js
// Deploys the VestingSchedule contract and creates example vesting schedules.
//
// Usage:
//   npx hardhat run scripts/setup-vesting.js --network <network>
//   or: node scripts/setup-vesting.js  (requires PRIVATE_KEY, RPC_URL, TOKEN_ADDRESS env vars)

const { ethers } = require("ethers");
const fs = require("fs");
const path = require("path");

async function main() {
  // ---- Configuration ----
  const RPC_URL       = process.env.RPC_URL       || "http://127.0.0.1:8545";
  const PRIVATE_KEY   = process.env.PRIVATE_KEY   || "";
  const TOKEN_ADDRESS = process.env.TOKEN_ADDRESS || "";

  if (!PRIVATE_KEY) {
    console.error("Error: PRIVATE_KEY environment variable is required.");
    process.exit(1);
  }
  if (!TOKEN_ADDRESS) {
    console.error("Error: TOKEN_ADDRESS environment variable is required.");
    console.error("Deploy ManagedToken first: node scripts/deploy-token.js");
    process.exit(1);
  }

  // ---- Connect ----
  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const wallet   = new ethers.Wallet(PRIVATE_KEY, provider);
  console.log("Deployer:", wallet.address);

  // ---- Load Artifacts ----
  const vestingArtifactPath = path.join(
    __dirname, "..", "out", "VestingSchedule.sol", "VestingSchedule.json"
  );
  const tokenArtifactPath = path.join(
    __dirname, "..", "out", "ManagedToken.sol", "ManagedToken.json"
  );

  if (!fs.existsSync(vestingArtifactPath) || !fs.existsSync(tokenArtifactPath)) {
    console.error("Artifacts not found. Run 'forge build' first.");
    process.exit(1);
  }

  const vestingArtifact = JSON.parse(fs.readFileSync(vestingArtifactPath, "utf8"));
  const tokenArtifact   = JSON.parse(fs.readFileSync(tokenArtifactPath, "utf8"));

  // ---- Deploy VestingSchedule ----
  console.log("Deploying VestingSchedule...");
  const vestingFactory = new ethers.ContractFactory(
    vestingArtifact.abi, vestingArtifact.bytecode.object, wallet
  );
  const vesting = await vestingFactory.deploy(TOKEN_ADDRESS);
  await vesting.waitForDeployment();
  const vestingAddress = await vesting.getAddress();
  console.log("VestingSchedule deployed to:", vestingAddress);

  // ---- Approve tokens for vesting ----
  const tokenContract = new ethers.Contract(TOKEN_ADDRESS, tokenArtifact.abi, wallet);
  const approvalAmount = ethers.parseEther("50000"); // 50k tokens for example schedules
  console.log("Approving", ethers.formatEther(approvalAmount), "tokens for vesting contract...");
  const approveTx = await tokenContract.approve(vestingAddress, approvalAmount);
  await approveTx.wait();
  console.log("Approval confirmed.");

  // ---- Create Example Schedules ----
  const now = Math.floor(Date.now() / 1000);

  const exampleSchedules = [
    {
      label:         "Team Lead - 4yr vest, 1yr cliff",
      beneficiary:   process.env.BENEFICIARY_1 || wallet.address,
      amount:        ethers.parseEther("20000"),
      start:         now,
      cliffDuration: 365 * 24 * 3600,        // 1 year cliff
      duration:      4 * 365 * 24 * 3600,    // 4 year total
    },
    {
      label:         "Advisor - 2yr vest, 6mo cliff",
      beneficiary:   process.env.BENEFICIARY_2 || wallet.address,
      amount:        ethers.parseEther("10000"),
      start:         now,
      cliffDuration: 180 * 24 * 3600,        // ~6 months cliff
      duration:      2 * 365 * 24 * 3600,    // 2 year total
    },
    {
      label:         "Early Contributor - 1yr vest, 3mo cliff",
      beneficiary:   process.env.BENEFICIARY_3 || wallet.address,
      amount:        ethers.parseEther("5000"),
      start:         now,
      cliffDuration: 90 * 24 * 3600,         // ~3 months cliff
      duration:      365 * 24 * 3600,         // 1 year total
    },
  ];

  const scheduleIds = [];
  for (const s of exampleSchedules) {
    console.log(`\nCreating schedule: ${s.label}`);
    console.log(`  Beneficiary: ${s.beneficiary}`);
    console.log(`  Amount:      ${ethers.formatEther(s.amount)} tokens`);

    const tx = await vesting.createSchedule(
      s.beneficiary,
      s.amount,
      s.start,
      s.cliffDuration,
      s.duration
    );
    const receipt = await tx.wait();
    console.log(`  Tx hash: ${receipt.hash}`);
    scheduleIds.push(s.label);
  }

  // ---- Save deployment info ----
  const deployment = {
    contract: "VestingSchedule",
    address:  vestingAddress,
    token:    TOKEN_ADDRESS,
    deployer: wallet.address,
    network:  (await provider.getNetwork()).chainId.toString(),
    schedules: exampleSchedules.map((s, i) => ({
      id:    i,
      label: s.label,
      beneficiary: s.beneficiary,
      amount: s.amount.toString(),
    })),
    timestamp: new Date().toISOString(),
  };

  const deploymentsDir = path.join(__dirname, "..", "deployments");
  if (!fs.existsSync(deploymentsDir)) {
    fs.mkdirSync(deploymentsDir, { recursive: true });
  }
  const outFile = path.join(deploymentsDir, "VestingSchedule.json");
  fs.writeFileSync(outFile, JSON.stringify(deployment, null, 2));
  console.log("\nDeployment saved to:", outFile);
  console.log("\nDone! Created", exampleSchedules.length, "vesting schedules.");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
