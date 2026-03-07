// deploy-token.js
// Deploys the ManagedToken contract using ethers.js
//
// Usage:
//   npx hardhat run scripts/deploy-token.js --network <network>
//   or: node scripts/deploy-token.js  (requires PRIVATE_KEY and RPC_URL env vars)

const { ethers } = require("ethers");
const fs = require("fs");
const path = require("path");

async function main() {
  // ---- Configuration ----
  const TOKEN_NAME     = process.env.TOKEN_NAME     || "ManagedToken";
  const TOKEN_SYMBOL   = process.env.TOKEN_SYMBOL   || "MTK";
  const INITIAL_SUPPLY = process.env.INITIAL_SUPPLY  || "1000000"; // in whole tokens
  const RPC_URL        = process.env.RPC_URL         || "http://127.0.0.1:8545";
  const PRIVATE_KEY    = process.env.PRIVATE_KEY     || "";

  if (!PRIVATE_KEY) {
    console.error("Error: PRIVATE_KEY environment variable is required.");
    process.exit(1);
  }

  // ---- Connect ----
  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const wallet   = new ethers.Wallet(PRIVATE_KEY, provider);
  console.log("Deployer:", wallet.address);

  // ---- Load ABI & Bytecode ----
  const artifactPath = path.join(
    __dirname, "..", "out", "ManagedToken.sol", "ManagedToken.json"
  );
  if (!fs.existsSync(artifactPath)) {
    console.error("Artifact not found. Run 'forge build' first.");
    process.exit(1);
  }
  const artifact = JSON.parse(fs.readFileSync(artifactPath, "utf8"));

  // ---- Deploy ----
  const factory = new ethers.ContractFactory(artifact.abi, artifact.bytecode.object, wallet);

  const initialSupplyWei = ethers.parseEther(INITIAL_SUPPLY);
  console.log(`Deploying ${TOKEN_NAME} (${TOKEN_SYMBOL}) with initial supply: ${INITIAL_SUPPLY} tokens...`);

  const token = await factory.deploy(TOKEN_NAME, TOKEN_SYMBOL, initialSupplyWei);
  await token.waitForDeployment();

  const address = await token.getAddress();
  console.log("ManagedToken deployed to:", address);

  // ---- Save deployment info ----
  const deployment = {
    contract: "ManagedToken",
    address:  address,
    deployer: wallet.address,
    network:  (await provider.getNetwork()).chainId.toString(),
    args: {
      name:          TOKEN_NAME,
      symbol:        TOKEN_SYMBOL,
      initialSupply: initialSupplyWei.toString(),
    },
    timestamp: new Date().toISOString(),
  };

  const deploymentsDir = path.join(__dirname, "..", "deployments");
  if (!fs.existsSync(deploymentsDir)) {
    fs.mkdirSync(deploymentsDir, { recursive: true });
  }
  const outFile = path.join(deploymentsDir, "ManagedToken.json");
  fs.writeFileSync(outFile, JSON.stringify(deployment, null, 2));
  console.log("Deployment saved to:", outFile);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
