/**
 * create-proposal.js
 *
 * Example script that creates a treasury withdrawal proposal via the
 * Governor contract. This demonstrates the full proposal creation flow.
 *
 * Prerequisites:
 *   - DAO stack deployed (run deploy-dao.js first)
 *   - Caller has enough voting power (>= proposalThreshold)
 *   - Caller has self-delegated their tokens
 *
 * Usage:
 *   npx hardhat run scripts/create-proposal.js --network <network>
 */

const { ethers } = require("hardhat");

// Update these addresses after deploying the DAO stack
const GOVERNOR_ADDRESS = process.env.GOVERNOR_ADDRESS || "0x0000000000000000000000000000000000000000";
const TREASURY_ADDRESS = process.env.TREASURY_ADDRESS || "0x0000000000000000000000000000000000000000";
const TOKEN_ADDRESS    = process.env.TOKEN_ADDRESS    || "0x0000000000000000000000000000000000000000";

async function main() {
  const [proposer] = await ethers.getSigners();
  console.log("Creating proposal with account:", proposer.address);

  // ------------------------------------------------------------------
  // Connect to deployed contracts
  // ------------------------------------------------------------------
  const Governor = await ethers.getContractFactory("Governor");
  const governor = Governor.attach(GOVERNOR_ADDRESS);

  const Treasury = await ethers.getContractFactory("Treasury");

  // ------------------------------------------------------------------
  // Build proposal actions
  // ------------------------------------------------------------------
  // This proposal withdraws 1000 tokens from the treasury to a recipient.

  const recipient = proposer.address; // For demonstration
  const withdrawAmount = ethers.utils.parseEther("1000");

  // Encode the Treasury.withdraw(token, to, amount) call
  const treasuryInterface = Treasury.interface;
  const calldata = treasuryInterface.encodeFunctionData("withdraw", [
    TOKEN_ADDRESS,
    recipient,
    withdrawAmount,
  ]);

  const targets = [TREASURY_ADDRESS];
  const values = [0]; // no ETH sent
  const calldatas = [calldata];
  const description = "Proposal #1: Withdraw 1000 GOV tokens from treasury to fund community grants";

  // ------------------------------------------------------------------
  // Submit proposal
  // ------------------------------------------------------------------
  console.log("\nSubmitting proposal...");
  console.log("Description:", description);
  console.log("Target:", TREASURY_ADDRESS);
  console.log("Withdraw amount:", ethers.utils.formatEther(withdrawAmount), "tokens");

  const tx = await governor.propose(targets, values, calldatas, description);
  const receipt = await tx.wait();

  // Extract proposal ID from events
  const event = receipt.events?.find((e) => e.event === "ProposalCreated");
  const proposalId = event?.args?.proposalId;

  console.log("\n========================================");
  console.log("  Proposal Created Successfully");
  console.log("========================================");
  console.log("Proposal ID  :", proposalId?.toString());
  console.log("Proposer     :", proposer.address);
  console.log("Vote Start   :", event?.args?.voteStart?.toString(), "(block)");
  console.log("Vote End     :", event?.args?.voteEnd?.toString(), "(block)");
  console.log("========================================");

  // ------------------------------------------------------------------
  // Check proposal state
  // ------------------------------------------------------------------
  const state = await governor.state(proposalId);
  const stateNames = [
    "Pending",
    "Active",
    "Canceled",
    "Defeated",
    "Succeeded",
    "Queued",
    "Executed",
  ];
  console.log("Current state:", stateNames[state]);
  console.log("\nNext steps:");
  console.log("  1. Wait for voting delay to pass");
  console.log("  2. Cast votes using: governor.castVote(proposalId, 1)");
  console.log("  3. Wait for voting period to end");
  console.log("  4. Execute using: governor.execute(proposalId)");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Proposal creation failed:", error);
    process.exit(1);
  });
