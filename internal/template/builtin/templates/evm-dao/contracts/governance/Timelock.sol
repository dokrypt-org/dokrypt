// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title Timelock
/// @notice A timelock controller that queues operations and enforces a minimum
///         delay before execution. Used by the Governor to add a safety buffer
///         between proposal passage and execution.
contract Timelock {
    // ---------------------------------------------------------------
    // Types
    // ---------------------------------------------------------------
    enum OperationState {
        Unset,     // 0 - does not exist
        Scheduled, // 1 - waiting for delay
        Ready,     // 2 - delay elapsed, can execute
        Executed   // 3 - already executed
    }

    struct Operation {
        bool executed;
        uint256 readyTimestamp; // timestamp after which the operation can be executed
    }

    // ---------------------------------------------------------------
    // State
    // ---------------------------------------------------------------
    uint256 public minDelay;
    mapping(bytes32 => Operation) public operations;

    mapping(address => bool) public isProposer;
    mapping(address => bool) public isExecutor;
    address public admin;

    // ---------------------------------------------------------------
    // Events
    // ---------------------------------------------------------------
    event CallScheduled(
        bytes32 indexed id,
        address indexed target,
        uint256 value,
        bytes data,
        bytes32 salt,
        uint256 delay,
        uint256 readyTimestamp
    );
    event CallExecuted(bytes32 indexed id, address indexed target, uint256 value, bytes data);
    event Cancelled(bytes32 indexed id);
    event MinDelayChanged(uint256 oldDelay, uint256 newDelay);
    event ProposerAdded(address indexed account);
    event ProposerRemoved(address indexed account);
    event ExecutorAdded(address indexed account);
    event ExecutorRemoved(address indexed account);

    // ---------------------------------------------------------------
    // Modifiers
    // ---------------------------------------------------------------
    modifier onlyProposer() {
        require(isProposer[msg.sender] || msg.sender == admin, "Timelock: caller is not proposer");
        _;
    }

    modifier onlyExecutor() {
        require(isExecutor[msg.sender] || msg.sender == admin, "Timelock: caller is not executor");
        _;
    }

    modifier onlyAdmin() {
        require(msg.sender == admin, "Timelock: caller is not admin");
        _;
    }

    // ---------------------------------------------------------------
    // Constructor
    // ---------------------------------------------------------------
    constructor(uint256 _minDelay, address[] memory proposers, address[] memory executors) {
        minDelay = _minDelay;
        admin = msg.sender;

        for (uint256 i = 0; i < proposers.length; i++) {
            isProposer[proposers[i]] = true;
            emit ProposerAdded(proposers[i]);
        }
        for (uint256 i = 0; i < executors.length; i++) {
            isExecutor[executors[i]] = true;
            emit ExecutorAdded(executors[i]);
        }
    }

    // ---------------------------------------------------------------
    // Core functions
    // ---------------------------------------------------------------

    /// @notice Hash an operation for use as its unique identifier.
    function hashOperation(
        address target,
        uint256 value,
        bytes calldata data,
        bytes32 salt
    ) public pure returns (bytes32) {
        return keccak256(abi.encode(target, value, data, salt));
    }

    /// @notice Schedule an operation after `delay` seconds (must be >= minDelay).
    function schedule(
        address target,
        uint256 value,
        bytes calldata data,
        bytes32 salt,
        uint256 delay
    ) external onlyProposer returns (bytes32 id) {
        require(delay >= minDelay, "Timelock: delay below minimum");

        id = hashOperation(target, value, data, salt);
        require(operations[id].readyTimestamp == 0, "Timelock: operation already scheduled");

        uint256 readyTimestamp = block.timestamp + delay;
        operations[id] = Operation({executed: false, readyTimestamp: readyTimestamp});

        emit CallScheduled(id, target, value, data, salt, delay, readyTimestamp);
    }

    /// @notice Execute a scheduled operation once the delay has passed.
    function execute(
        address target,
        uint256 value,
        bytes calldata data,
        bytes32 salt
    ) external payable onlyExecutor {
        bytes32 id = hashOperation(target, value, data, salt);
        Operation storage op = operations[id];

        require(op.readyTimestamp > 0, "Timelock: operation not scheduled");
        require(!op.executed, "Timelock: operation already executed");
        require(block.timestamp >= op.readyTimestamp, "Timelock: operation not ready");

        op.executed = true;

        (bool success, bytes memory result) = target.call{value: value}(data);
        require(success, string(abi.encodePacked("Timelock: call failed: ", result)));

        emit CallExecuted(id, target, value, data);
    }

    /// @notice Cancel a scheduled (non-executed) operation.
    function cancel(bytes32 id) external onlyProposer {
        require(operations[id].readyTimestamp > 0, "Timelock: operation not scheduled");
        require(!operations[id].executed, "Timelock: operation already executed");

        delete operations[id];
        emit Cancelled(id);
    }

    /// @notice Returns the state of an operation.
    function getOperationState(bytes32 id) external view returns (OperationState) {
        Operation storage op = operations[id];
        if (op.readyTimestamp == 0) return OperationState.Unset;
        if (op.executed) return OperationState.Executed;
        if (block.timestamp >= op.readyTimestamp) return OperationState.Ready;
        return OperationState.Scheduled;
    }

    /// @notice Returns whether an operation is pending (scheduled but not executed).
    function isOperationPending(bytes32 id) external view returns (bool) {
        return operations[id].readyTimestamp > 0 && !operations[id].executed;
    }

    /// @notice Returns whether an operation is ready for execution.
    function isOperationReady(bytes32 id) external view returns (bool) {
        Operation storage op = operations[id];
        return op.readyTimestamp > 0 && !op.executed && block.timestamp >= op.readyTimestamp;
    }

    // ---------------------------------------------------------------
    // Admin functions
    // ---------------------------------------------------------------

    function setMinDelay(uint256 _minDelay) external onlyAdmin {
        emit MinDelayChanged(minDelay, _minDelay);
        minDelay = _minDelay;
    }

    function addProposer(address account) external onlyAdmin {
        isProposer[account] = true;
        emit ProposerAdded(account);
    }

    function removeProposer(address account) external onlyAdmin {
        isProposer[account] = false;
        emit ProposerRemoved(account);
    }

    function addExecutor(address account) external onlyAdmin {
        isExecutor[account] = true;
        emit ExecutorAdded(account);
    }

    function removeExecutor(address account) external onlyAdmin {
        isExecutor[account] = false;
        emit ExecutorRemoved(account);
    }

    function transferAdmin(address newAdmin) external onlyAdmin {
        require(newAdmin != address(0), "Timelock: zero address");
        admin = newAdmin;
    }

    // Allow the contract to receive ETH so it can forward value-bearing calls.
    receive() external payable {}
}
