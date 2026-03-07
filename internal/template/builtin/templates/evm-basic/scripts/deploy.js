// Deploy script for Counter and SimpleToken
// Usage: npx hardhat run scripts/deploy.js --network localhost

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying with account:", deployer.address);

  // Deploy Counter
  const Counter = await ethers.getContractFactory("Counter");
  const counter = await Counter.deploy();
  await counter.waitForDeployment();
  console.log("Counter deployed to:", await counter.getAddress());

  // Deploy SimpleToken
  const SimpleToken = await ethers.getContractFactory("SimpleToken");
  const token = await SimpleToken.deploy("MyToken", "MTK", 1000000);
  await token.waitForDeployment();
  console.log("SimpleToken deployed to:", await token.getAddress());
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
