// Upload NFT metadata to IPFS
// Requires: npm install ipfs-http-client

async function main() {
  console.log("Uploading metadata to IPFS...");

  const metadata = {
    name: "My NFT #1",
    description: "An awesome NFT from my collection",
    image: "ipfs://YOUR_IMAGE_CID",
    attributes: [
      { trait_type: "Background", value: "Blue" },
      { trait_type: "Rarity", value: "Common" }
    ]
  };

  // Using local IPFS node (started by dokrypt)
  const response = await fetch("http://localhost:5001/api/v0/add", {
    method: "POST",
    body: JSON.stringify(metadata),
    headers: { "Content-Type": "application/json" }
  });

  const result = await response.json();
  console.log("Metadata CID:", result.Hash);
  console.log("Gateway URL: http://localhost:8080/ipfs/" + result.Hash);
}

main().catch(console.error);
