// Deploy NFTCollection and Marketplace contracts
// Usage: node scripts/deploy-nft.js

const { ethers } = require("ethers");
const fs = require("fs");
const path = require("path");

async function main() {
  // Connect to the local Dokrypt chain
  const provider = new ethers.JsonRpcProvider("http://localhost:8545");
  const signer = await provider.getSigner(0);
  const deployerAddress = await signer.getAddress();

  console.log("Deploying contracts with account:", deployerAddress);
  console.log("Account balance:", ethers.formatEther(await provider.getBalance(deployerAddress)), "ETH");

  // ---- Deploy NFTCollection ----

  console.log("\n--- Deploying NFTCollection ---");

  const nftArtifactPath = path.join(__dirname, "..", "out", "NFTCollection.sol", "NFTCollection.json");
  const nftArtifact = JSON.parse(fs.readFileSync(nftArtifactPath, "utf8"));

  const nftFactory = new ethers.ContractFactory(nftArtifact.abi, nftArtifact.bytecode.object, signer);

  const collectionName = "MyNFTCollection";
  const collectionSymbol = "MNFT";
  const baseURI = "ipfs://YOUR_BASE_CID/";
  const maxSupply = 10000;
  const mintPrice = ethers.parseEther("0.01");

  const nft = await nftFactory.deploy(collectionName, collectionSymbol, baseURI, maxSupply, mintPrice);
  await nft.waitForDeployment();

  const nftAddress = await nft.getAddress();
  console.log("NFTCollection deployed to:", nftAddress);
  console.log("  Name:", collectionName);
  console.log("  Symbol:", collectionSymbol);
  console.log("  Max Supply:", maxSupply);
  console.log("  Mint Price:", ethers.formatEther(mintPrice), "ETH");

  // ---- Deploy Marketplace ----

  console.log("\n--- Deploying Marketplace ---");

  const marketArtifactPath = path.join(__dirname, "..", "out", "Marketplace.sol", "Marketplace.json");
  const marketArtifact = JSON.parse(fs.readFileSync(marketArtifactPath, "utf8"));

  const marketFactory = new ethers.ContractFactory(marketArtifact.abi, marketArtifact.bytecode.object, signer);

  const marketplace = await marketFactory.deploy();
  await marketplace.waitForDeployment();

  const marketAddress = await marketplace.getAddress();
  console.log("Marketplace deployed to:", marketAddress);
  console.log("  Platform Fee: 2.5%");

  // ---- Summary ----

  console.log("\n=== Deployment Summary ===");
  console.log("NFTCollection:", nftAddress);
  console.log("Marketplace:  ", marketAddress);

  // Save deployment addresses
  const deployments = {
    network: "localhost",
    chainId: 31337,
    deployer: deployerAddress,
    contracts: {
      NFTCollection: {
        address: nftAddress,
        name: collectionName,
        symbol: collectionSymbol,
        maxSupply: maxSupply,
        mintPrice: mintPrice.toString(),
      },
      Marketplace: {
        address: marketAddress,
        platformFeeBps: 250,
      },
    },
    deployedAt: new Date().toISOString(),
  };

  const deploymentsPath = path.join(__dirname, "..", "deployments.json");
  fs.writeFileSync(deploymentsPath, JSON.stringify(deployments, null, 2));
  console.log("\nDeployment addresses saved to:", deploymentsPath);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Deployment failed:", error);
    process.exit(1);
  });
