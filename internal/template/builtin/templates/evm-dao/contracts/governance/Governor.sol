// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../token/VotingToken.sol";

/// @title Governor
/// @notice On-chain governance contract. Proposals are created, voted on, then
///         executed through a Timelock controller. Voting power is sourced from
///         the VotingToken checkpoint system.
contract Governor {
    // ---------------------------------------------------------------
    // Types
    // ---------------------------------------------------------------
    enum ProposalState {
        Pending,    // 0
        Active,     // 1
        Canceled,   // 2
        Defeated,   // 3
        Succeeded,  // 4
        Queued,     // 5
        Executed    // 6
    }

    struct ProposalCore {
        address proposer;
        uint256 voteStart;
        uint256 voteEnd;
        bool executed;
        bool canceled;
        uint256 forVotes;
        uint256 againstVotes;
        uint256 abstainVotes;
        bytes32 descriptionHash;
        address[] targets;
        uint256[] values;
        bytes[] calldatas;
    }

    // ---------------------------------------------------------------
    // State
    // ---------------------------------------------------------------
    VotingToken public token;
    address public timelock;
    address public admin;

    uint256 public votingDelay;       // blocks after proposal creation before voting starts
    uint256 public votingPeriod;      // blocks the vote remains open
    uint256 public quorum;            // minimum total votes for a proposal to pass
    uint256 public proposalThreshold; // minimum token balance to create a proposal

    uint256 public proposalCount;
    mapping(uint256 => ProposalCore) public proposals;
    mapping(uint256 => mapping(address => bool)) public hasVoted;

    // ---------------------------------------------------------------
    // Events
    // ---------------------------------------------------------------
    event ProposalCreated(
        uint256 indexed proposalId,
        address indexed proposer,
        address[] targets,
        uint256[] values,
        bytes[] calldatas,
        string description,
        uint256 voteStart,
        uint256 voteEnd
    );
    event VoteCast(address indexed voter, uint256 indexed proposalId, uint8 support, uint256 weight);
    event ProposalExecuted(uint256 indexed proposalId);
    event ProposalCanceled(uint256 indexed proposalId);
    event ProposalQueued(uint256 indexed proposalId);
    event TimelockUpdated(address indexed oldTimelock, address indexed newTimelock);

    // ---------------------------------------------------------------
    // Modifiers
    // ---------------------------------------------------------------
    modifier onlyAdmin() {
        require(msg.sender == admin, "Governor: caller is not admin");
        _;
    }

    // ---------------------------------------------------------------
    // Constructor
    // ---------------------------------------------------------------
    constructor(
        address _token,
        address _timelock,
        uint256 _votingDelay,
        uint256 _votingPeriod,
        uint256 _quorum,
        uint256 _proposalThreshold
    ) {
        require(_token != address(0), "Governor: zero token address");
        require(_timelock != address(0), "Governor: zero timelock address");
        require(_votingPeriod > 0, "Governor: voting period must be > 0");

        token = VotingToken(_token);
        timelock = _timelock;
        admin = msg.sender;
        votingDelay = _votingDelay;
        votingPeriod = _votingPeriod;
        quorum = _quorum;
        proposalThreshold = _proposalThreshold;
    }

    // ---------------------------------------------------------------
    // Proposal lifecycle
    // ---------------------------------------------------------------

    /// @notice Create a new proposal.
    function propose(
        address[] calldata targets,
        uint256[] calldata values,
        bytes[] calldata calldatas,
        string calldata description
    ) external returns (uint256 proposalId) {
        require(targets.length > 0, "Governor: empty proposal");
        require(
            targets.length == values.length && targets.length == calldatas.length,
            "Governor: invalid proposal length"
        );
        require(
            token.getVotes(msg.sender) >= proposalThreshold,
            "Governor: below proposal threshold"
        );

        proposalCount++;
        proposalId = proposalCount;

        uint256 start = block.number + votingDelay;
        uint256 end = start + votingPeriod;

        ProposalCore storage p = proposals[proposalId];
        p.proposer = msg.sender;
        p.voteStart = start;
        p.voteEnd = end;
        p.descriptionHash = keccak256(bytes(description));

        for (uint256 i = 0; i < targets.length; i++) {
            p.targets.push(targets[i]);
            p.values.push(values[i]);
            p.calldatas.push(calldatas[i]);
        }

        emit ProposalCreated(
            proposalId,
            msg.sender,
            targets,
            values,
            calldatas,
            description,
            start,
            end
        );
    }

    /// @notice Cast a vote on an active proposal.
    /// @param support 0 = Against, 1 = For, 2 = Abstain
    function castVote(uint256 proposalId, uint8 support) external {
        require(state(proposalId) == ProposalState.Active, "Governor: proposal not active");
        require(!hasVoted[proposalId][msg.sender], "Governor: already voted");
        require(support <= 2, "Governor: invalid vote type");

        ProposalCore storage p = proposals[proposalId];

        // Snapshot voting power at the block the vote started
        uint256 weight = token.getPastVotes(msg.sender, p.voteStart);
        require(weight > 0, "Governor: no voting power");

        hasVoted[proposalId][msg.sender] = true;

        if (support == 0) {
            p.againstVotes += weight;
        } else if (support == 1) {
            p.forVotes += weight;
        } else {
            p.abstainVotes += weight;
        }

        emit VoteCast(msg.sender, proposalId, support, weight);
    }

    /// @notice Execute a succeeded proposal through the timelock.
    function execute(uint256 proposalId) external payable {
        require(
            state(proposalId) == ProposalState.Succeeded || state(proposalId) == ProposalState.Queued,
            "Governor: proposal not ready"
        );

        ProposalCore storage p = proposals[proposalId];
        p.executed = true;

        for (uint256 i = 0; i < p.targets.length; i++) {
            (bool success, bytes memory result) = p.targets[i].call{value: p.values[i]}(p.calldatas[i]);
            require(success, string(abi.encodePacked("Governor: call failed: ", result)));
        }

        emit ProposalExecuted(proposalId);
    }

    /// @notice Cancel a proposal. Only the proposer or admin can cancel.
    function cancel(uint256 proposalId) external {
        ProposalCore storage p = proposals[proposalId];
        require(
            msg.sender == p.proposer || msg.sender == admin,
            "Governor: only proposer or admin"
        );
        require(!p.executed, "Governor: already executed");
        require(!p.canceled, "Governor: already canceled");

        p.canceled = true;
        emit ProposalCanceled(proposalId);
    }

    // ---------------------------------------------------------------
    // View helpers
    // ---------------------------------------------------------------

    /// @notice Returns the current state of a proposal.
    function state(uint256 proposalId) public view returns (ProposalState) {
        require(proposalId > 0 && proposalId <= proposalCount, "Governor: invalid proposal id");

        ProposalCore storage p = proposals[proposalId];

        if (p.canceled) return ProposalState.Canceled;
        if (p.executed) return ProposalState.Executed;
        if (block.number < p.voteStart) return ProposalState.Pending;
        if (block.number <= p.voteEnd) return ProposalState.Active;

        // Voting ended -- determine outcome
        bool quorumReached = (p.forVotes + p.againstVotes + p.abstainVotes) >= quorum;
        bool voteSucceeded = p.forVotes > p.againstVotes;

        if (!quorumReached || !voteSucceeded) return ProposalState.Defeated;

        return ProposalState.Succeeded;
    }

    function getProposalVotes(uint256 proposalId)
        external
        view
        returns (uint256 againstVotes, uint256 forVotes, uint256 abstainVotes)
    {
        ProposalCore storage p = proposals[proposalId];
        return (p.againstVotes, p.forVotes, p.abstainVotes);
    }

    function getProposalActions(uint256 proposalId)
        external
        view
        returns (address[] memory targets, uint256[] memory values, bytes[] memory calldatas)
    {
        ProposalCore storage p = proposals[proposalId];
        return (p.targets, p.values, p.calldatas);
    }

    // ---------------------------------------------------------------
    // Admin functions
    // ---------------------------------------------------------------

    function setVotingDelay(uint256 _votingDelay) external onlyAdmin {
        votingDelay = _votingDelay;
    }

    function setVotingPeriod(uint256 _votingPeriod) external onlyAdmin {
        require(_votingPeriod > 0, "Governor: voting period must be > 0");
        votingPeriod = _votingPeriod;
    }

    function setQuorum(uint256 _quorum) external onlyAdmin {
        quorum = _quorum;
    }

    function setProposalThreshold(uint256 _proposalThreshold) external onlyAdmin {
        proposalThreshold = _proposalThreshold;
    }

    function setTimelock(address _timelock) external onlyAdmin {
        require(_timelock != address(0), "Governor: zero address");
        emit TimelockUpdated(timelock, _timelock);
        timelock = _timelock;
    }

    function transferAdmin(address newAdmin) external onlyAdmin {
        require(newAdmin != address(0), "Governor: zero address");
        admin = newAdmin;
    }
}
