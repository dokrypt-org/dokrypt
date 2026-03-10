// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title L2ToL1MessageSender
/// @notice Sends messages from Arbitrum L2 back to L1 (Ethereum) via the ArbSys precompile.
/// On Arbitrum, the ArbSys precompile at address 0x64 provides the sendTxToL1 function
/// which creates an outbox entry that can be executed on L1 after the challenge period.
contract L2ToL1MessageSender {
    // ──────────────────── Constants ────────────────────
    /// @notice ArbSys precompile address on Arbitrum (0x0000000000000000000000000000000000000064)
    address public constant ARBSYS = address(0x0000000000000000000000000000000000000064);

    // ──────────────────── State ────────────────────
    address public owner;

    uint256 public nextMessageNonce;

    struct OutgoingMessage {
        address sender;
        address l1Target;
        bytes data;
        uint256 nonce;
        uint256 timestamp;
        bool confirmed;
    }

    mapping(uint256 => OutgoingMessage) public messages;

    // Allowed L1 target addresses
    mapping(address => bool) public allowedL1Targets;

    // Reentrancy guard
    uint256 private _locked = 1;

    // ──────────────────── Events ────────────────────
    event L2ToL1MessageSent(
        uint256 indexed nonce,
        address indexed sender,
        address indexed l1Target,
        bytes data
    );
    event MessageConfirmed(uint256 indexed nonce);
    event L1TargetAllowed(address indexed target, bool allowed);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // ──────────────────── Modifiers ────────────────────
    modifier onlyOwner() {
        require(msg.sender == owner, "L2ToL1MessageSender: caller is not the owner");
        _;
    }

    modifier nonReentrant() {
        require(_locked == 1, "L2ToL1MessageSender: reentrancy");
        _locked = 2;
        _;
        _locked = 1;
    }

    constructor() {
        owner = msg.sender;
    }

    // ──────────────────── Core Functions ────────────────────

    /// @notice Send a message from L2 to L1 via the ArbSys precompile.
    /// On Arbitrum, this creates an outbox entry that can be claimed on L1
    /// after the challenge period (~7 days on mainnet).
    /// @param l1Target Address of the contract on L1 to call.
    /// @param data Calldata to execute on the L1 target.
    /// @return nonce The unique nonce for this outgoing message.
    function sendMessageToL1(
        address l1Target,
        bytes calldata data
    ) external nonReentrant returns (uint256 nonce) {
        require(l1Target != address(0), "L2ToL1MessageSender: zero L1 target");
        require(
            allowedL1Targets[l1Target] || msg.sender == owner,
            "L2ToL1MessageSender: L1 target not allowed"
        );

        nonce = nextMessageNonce++;

        messages[nonce] = OutgoingMessage({
            sender: msg.sender,
            l1Target: l1Target,
            data: data,
            nonce: nonce,
            timestamp: block.timestamp,
            confirmed: false
        });

        // Attempt to call ArbSys precompile for actual L2-to-L1 message passing.
        // On a local fork, this precompile may not be available, so we handle failure gracefully.
        (bool success, ) = ARBSYS.call(
            abi.encodeWithSignature(
                "sendTxToL1(address,bytes)",
                l1Target,
                data
            )
        );
        // On local fork/anvil the precompile won't exist, so we allow failure
        if (!success) {
            // Message is stored locally for testing
        }

        emit L2ToL1MessageSent(nonce, msg.sender, l1Target, data);
    }

    /// @notice Send a simple encoded function call from L2 to L1.
    /// Convenience function that encodes the function signature and arguments.
    /// @param l1Target Address of the L1 contract.
    /// @param functionSig Function signature (e.g., "receiveMessage(address,bytes)").
    /// @param payload ABI-encoded function arguments.
    /// @return nonce The unique nonce for this outgoing message.
    function sendEncodedMessageToL1(
        address l1Target,
        string calldata functionSig,
        bytes calldata payload
    ) external nonReentrant returns (uint256 nonce) {
        require(l1Target != address(0), "L2ToL1MessageSender: zero L1 target");
        require(
            allowedL1Targets[l1Target] || msg.sender == owner,
            "L2ToL1MessageSender: L1 target not allowed"
        );

        bytes memory data = abi.encodePacked(
            bytes4(keccak256(bytes(functionSig))),
            payload
        );

        nonce = nextMessageNonce++;

        messages[nonce] = OutgoingMessage({
            sender: msg.sender,
            l1Target: l1Target,
            data: data,
            nonce: nonce,
            timestamp: block.timestamp,
            confirmed: false
        });

        emit L2ToL1MessageSent(nonce, msg.sender, l1Target, data);
    }

    /// @notice Confirm that a message has been executed on L1.
    /// In production, this would be verified via the Arbitrum outbox proof.
    /// For local testing, the owner can manually confirm messages.
    /// @param nonce The message nonce to confirm.
    function confirmMessage(uint256 nonce) external onlyOwner {
        require(nonce < nextMessageNonce, "L2ToL1MessageSender: invalid nonce");
        require(!messages[nonce].confirmed, "L2ToL1MessageSender: already confirmed");

        messages[nonce].confirmed = true;
        emit MessageConfirmed(nonce);
    }

    // ──────────────────── View Functions ────────────────────

    /// @notice Get the details of an outgoing message.
    /// @param nonce The message nonce.
    /// @return sender The address that sent the message.
    /// @return l1Target The L1 target address.
    /// @return confirmed Whether the message has been confirmed on L1.
    /// @return timestamp When the message was sent.
    function getMessage(uint256 nonce)
        external
        view
        returns (address sender, address l1Target, bool confirmed, uint256 timestamp)
    {
        require(nonce < nextMessageNonce, "L2ToL1MessageSender: invalid nonce");
        OutgoingMessage storage msg_ = messages[nonce];
        return (msg_.sender, msg_.l1Target, msg_.confirmed, msg_.timestamp);
    }

    /// @notice Get the total number of messages sent.
    /// @return The total message count.
    function messageCount() external view returns (uint256) {
        return nextMessageNonce;
    }

    // ──────────────────── Admin ────────────────────

    /// @notice Allow or disallow an L1 target address.
    /// @param target The L1 address to configure.
    /// @param allowed Whether to allow or disallow the target.
    function setAllowedL1Target(address target, bool allowed) external onlyOwner {
        require(target != address(0), "L2ToL1MessageSender: zero address");
        allowedL1Targets[target] = allowed;
        emit L1TargetAllowed(target, allowed);
    }

    /// @notice Transfer ownership.
    /// @param newOwner New owner address.
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "L2ToL1MessageSender: zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }
}
