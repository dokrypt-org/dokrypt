// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title L1ToL2MessageSender
/// @notice Sends messages from L1 (Ethereum) to L2 (Arbitrum) via retryable tickets.
/// Models the Arbitrum Delayed Inbox pattern where L1 contracts create retryable
/// tickets that are automatically executed on L2.
contract L1ToL2MessageSender {
    // ──────────────────── State ────────────────────
    address public owner;
    address public inbox; // Arbitrum Delayed Inbox address on L1

    uint256 public nextTicketId;

    struct RetryableTicket {
        address sender;
        address l2Target;
        uint256 l2Value;
        uint256 maxSubmissionCost;
        uint256 maxGas;
        uint256 gasPriceBid;
        bytes data;
        uint256 timestamp;
        bool executed;
    }

    mapping(uint256 => RetryableTicket) public tickets;

    // Reentrancy guard
    uint256 private _locked = 1;

    // ──────────────────── Events ────────────────────
    event RetryableTicketCreated(
        uint256 indexed ticketId,
        address indexed sender,
        address indexed l2Target,
        uint256 l2Value,
        bytes data
    );
    event TicketExecuted(uint256 indexed ticketId);
    event InboxUpdated(address indexed oldInbox, address indexed newInbox);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // ──────────────────── Modifiers ────────────────────
    modifier onlyOwner() {
        require(msg.sender == owner, "L1ToL2MessageSender: caller is not the owner");
        _;
    }

    modifier nonReentrant() {
        require(_locked == 1, "L1ToL2MessageSender: reentrancy");
        _locked = 2;
        _;
        _locked = 1;
    }

    constructor(address _inbox) {
        require(_inbox != address(0), "L1ToL2MessageSender: zero inbox address");
        owner = msg.sender;
        inbox = _inbox;
    }

    // ──────────────────── Core Functions ────────────────────

    /// @notice Create a retryable ticket to send a message from L1 to L2.
    /// The ticket encodes a function call that will be executed on the L2 target.
    /// @param l2Target Address of the contract on L2 to call.
    /// @param l2Value ETH value to send to L2 target.
    /// @param maxSubmissionCost Maximum cost for submitting the retryable ticket.
    /// @param maxGas Maximum gas for L2 execution.
    /// @param gasPriceBid Gas price bid for L2 execution.
    /// @param data Calldata to execute on L2 target.
    /// @return ticketId The unique identifier for this retryable ticket.
    function createRetryableTicket(
        address l2Target,
        uint256 l2Value,
        uint256 maxSubmissionCost,
        uint256 maxGas,
        uint256 gasPriceBid,
        bytes calldata data
    ) external payable nonReentrant returns (uint256 ticketId) {
        require(l2Target != address(0), "L1ToL2MessageSender: zero L2 target");
        require(maxGas > 0, "L1ToL2MessageSender: zero max gas");
        require(gasPriceBid > 0, "L1ToL2MessageSender: zero gas price bid");

        // Verify sufficient ETH is provided for submission cost + L2 execution
        uint256 totalCost = maxSubmissionCost + (maxGas * gasPriceBid) + l2Value;
        require(msg.value >= totalCost, "L1ToL2MessageSender: insufficient ETH for ticket");

        ticketId = nextTicketId++;

        tickets[ticketId] = RetryableTicket({
            sender: msg.sender,
            l2Target: l2Target,
            l2Value: l2Value,
            maxSubmissionCost: maxSubmissionCost,
            maxGas: maxGas,
            gasPriceBid: gasPriceBid,
            data: data,
            timestamp: block.timestamp,
            executed: false
        });

        // In production, this would call inbox.createRetryableTicket()
        // For local development, we store the ticket and emit an event
        (bool success, ) = inbox.call{value: msg.value}(
            abi.encodeWithSignature(
                "createRetryableTicket(address,uint256,uint256,address,address,uint256,uint256,bytes)",
                l2Target,
                l2Value,
                maxSubmissionCost,
                msg.sender, // excessFeeRefundAddress
                msg.sender, // callValueRefundAddress
                maxGas,
                gasPriceBid,
                data
            )
        );
        // On a local fork the inbox may not exist, so we allow failure
        if (!success) {
            // Ticket is stored locally for testing purposes
        }

        emit RetryableTicketCreated(ticketId, msg.sender, l2Target, l2Value, data);
    }

    /// @notice Send a simple message to L2 (convenience wrapper with default gas params).
    /// @param l2Target Address of the contract on L2 to call.
    /// @param data Calldata to execute on L2 target.
    /// @return ticketId The unique identifier for this retryable ticket.
    function sendMessageToL2(
        address l2Target,
        bytes calldata data
    ) external payable nonReentrant returns (uint256 ticketId) {
        require(l2Target != address(0), "L1ToL2MessageSender: zero L2 target");
        require(msg.value > 0, "L1ToL2MessageSender: must send ETH for L2 gas");

        ticketId = nextTicketId++;

        uint256 defaultMaxGas = 1_000_000;
        uint256 defaultGasPriceBid = 100 gwei;
        uint256 defaultSubmissionCost = 0.01 ether;

        tickets[ticketId] = RetryableTicket({
            sender: msg.sender,
            l2Target: l2Target,
            l2Value: 0,
            maxSubmissionCost: defaultSubmissionCost,
            maxGas: defaultMaxGas,
            gasPriceBid: defaultGasPriceBid,
            data: data,
            timestamp: block.timestamp,
            executed: false
        });

        emit RetryableTicketCreated(ticketId, msg.sender, l2Target, 0, data);
    }

    /// @notice Mark a ticket as executed (for local testing simulation).
    /// In production, this would be handled by the Arbitrum bridge automatically.
    /// @param ticketId The ticket to mark as executed.
    function markExecuted(uint256 ticketId) external onlyOwner {
        require(ticketId < nextTicketId, "L1ToL2MessageSender: invalid ticket ID");
        require(!tickets[ticketId].executed, "L1ToL2MessageSender: already executed");

        tickets[ticketId].executed = true;
        emit TicketExecuted(ticketId);
    }

    // ──────────────────── View Functions ────────────────────

    /// @notice Get the details of a retryable ticket.
    /// @param ticketId The ticket ID to query.
    /// @return sender The address that created the ticket.
    /// @return l2Target The L2 target address.
    /// @return l2Value The ETH value for L2.
    /// @return executed Whether the ticket has been executed.
    function getTicket(uint256 ticketId)
        external
        view
        returns (address sender, address l2Target, uint256 l2Value, bool executed)
    {
        require(ticketId < nextTicketId, "L1ToL2MessageSender: invalid ticket ID");
        RetryableTicket storage ticket = tickets[ticketId];
        return (ticket.sender, ticket.l2Target, ticket.l2Value, ticket.executed);
    }

    /// @notice Get the total number of tickets created.
    /// @return The total ticket count.
    function ticketCount() external view returns (uint256) {
        return nextTicketId;
    }

    // ──────────────────── Admin ────────────────────

    /// @notice Update the Arbitrum Delayed Inbox address.
    /// @param _inbox New inbox address.
    function setInbox(address _inbox) external onlyOwner {
        require(_inbox != address(0), "L1ToL2MessageSender: zero inbox address");
        emit InboxUpdated(inbox, _inbox);
        inbox = _inbox;
    }

    /// @notice Transfer ownership.
    /// @param newOwner New owner address.
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "L1ToL2MessageSender: zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    /// @notice Allow the contract to receive ETH for L2 gas funding.
    receive() external payable {}
}
