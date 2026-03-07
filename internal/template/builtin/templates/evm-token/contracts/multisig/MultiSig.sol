// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title MultiSig
/// @notice M-of-N multisignature wallet that can hold ETH and execute
///         arbitrary transactions once the required number of owners confirm.
contract MultiSig {
    // ----------------------------------------------------------------
    // Types
    // ----------------------------------------------------------------
    struct Transaction {
        address to;
        uint256 value;
        bytes   data;
        bool    executed;
        uint256 confirmations;
    }

    // ----------------------------------------------------------------
    // State
    // ----------------------------------------------------------------
    address[] public owners;
    mapping(address => bool) public isOwner;
    uint256 public required;  // minimum confirmations to execute

    Transaction[] public transactions;

    // txId => owner => confirmed
    mapping(uint256 => mapping(address => bool)) public isConfirmed;

    // ----------------------------------------------------------------
    // Events
    // ----------------------------------------------------------------
    event Deposit(address indexed sender, uint256 value, uint256 balance);
    event Submission(uint256 indexed txId);
    event Confirmation(address indexed owner, uint256 indexed txId);
    event Revocation(address indexed owner, uint256 indexed txId);
    event Execution(uint256 indexed txId);
    event ExecutionFailure(uint256 indexed txId);

    // ----------------------------------------------------------------
    // Modifiers
    // ----------------------------------------------------------------
    modifier onlyOwner() {
        require(isOwner[msg.sender], "MultiSig: caller is not an owner");
        _;
    }

    modifier txExists(uint256 txId) {
        require(txId < transactions.length, "MultiSig: transaction does not exist");
        _;
    }

    modifier notExecuted(uint256 txId) {
        require(!transactions[txId].executed, "MultiSig: transaction already executed");
        _;
    }

    modifier notConfirmed(uint256 txId) {
        require(!isConfirmed[txId][msg.sender], "MultiSig: transaction already confirmed by caller");
        _;
    }

    // ----------------------------------------------------------------
    // Constructor
    // ----------------------------------------------------------------
    constructor(address[] memory _owners, uint256 _required) {
        require(_owners.length > 0, "MultiSig: owners required");
        require(
            _required > 0 && _required <= _owners.length,
            "MultiSig: invalid required count"
        );

        for (uint256 i = 0; i < _owners.length; i++) {
            address o = _owners[i];
            require(o != address(0), "MultiSig: zero address owner");
            require(!isOwner[o],     "MultiSig: duplicate owner");

            isOwner[o] = true;
            owners.push(o);
        }
        required = _required;
    }

    // ----------------------------------------------------------------
    // Receive ETH
    // ----------------------------------------------------------------
    receive() external payable {
        emit Deposit(msg.sender, msg.value, address(this).balance);
    }

    fallback() external payable {
        emit Deposit(msg.sender, msg.value, address(this).balance);
    }

    // ----------------------------------------------------------------
    // Transaction Lifecycle
    // ----------------------------------------------------------------

    /// @notice Submit a new transaction for confirmation.
    /// @return txId The index of the newly created transaction.
    function submitTransaction(
        address _to,
        uint256 _value,
        bytes calldata _data
    ) external onlyOwner returns (uint256 txId) {
        txId = transactions.length;

        transactions.push(Transaction({
            to:            _to,
            value:         _value,
            data:          _data,
            executed:      false,
            confirmations: 0
        }));

        emit Submission(txId);
    }

    /// @notice Confirm a pending transaction.
    function confirmTransaction(uint256 txId)
        external
        onlyOwner
        txExists(txId)
        notExecuted(txId)
        notConfirmed(txId)
    {
        Transaction storage t = transactions[txId];
        isConfirmed[txId][msg.sender] = true;
        t.confirmations += 1;

        emit Confirmation(msg.sender, txId);
    }

    /// @notice Execute a confirmed transaction.
    function executeTransaction(uint256 txId)
        external
        onlyOwner
        txExists(txId)
        notExecuted(txId)
    {
        Transaction storage t = transactions[txId];
        require(
            t.confirmations >= required,
            "MultiSig: not enough confirmations"
        );

        t.executed = true;

        (bool success, ) = t.to.call{value: t.value}(t.data);
        if (success) {
            emit Execution(txId);
        } else {
            emit ExecutionFailure(txId);
            t.executed = false;
        }
    }

    /// @notice Revoke a previously given confirmation.
    function revokeConfirmation(uint256 txId)
        external
        onlyOwner
        txExists(txId)
        notExecuted(txId)
    {
        require(isConfirmed[txId][msg.sender], "MultiSig: transaction not confirmed by caller");

        Transaction storage t = transactions[txId];
        isConfirmed[txId][msg.sender] = false;
        t.confirmations -= 1;

        emit Revocation(msg.sender, txId);
    }

    // ----------------------------------------------------------------
    // View Functions
    // ----------------------------------------------------------------

    /// @notice Return the total number of submitted transactions.
    function getTransactionCount() external view returns (uint256) {
        return transactions.length;
    }

    /// @notice Return full details of a transaction.
    function getTransaction(uint256 txId)
        external
        view
        txExists(txId)
        returns (
            address to,
            uint256 value,
            bytes memory data,
            bool executed,
            uint256 confirmations
        )
    {
        Transaction storage t = transactions[txId];
        return (t.to, t.value, t.data, t.executed, t.confirmations);
    }

    /// @notice Return the list of owner addresses.
    function getOwners() external view returns (address[] memory) {
        return owners;
    }

    /// @notice Check whether a transaction has reached the required confirmations.
    function isTransactionConfirmed(uint256 txId)
        external
        view
        txExists(txId)
        returns (bool)
    {
        return transactions[txId].confirmations >= required;
    }

    /// @notice Return the number of confirmations for a transaction.
    function getConfirmationCount(uint256 txId)
        external
        view
        txExists(txId)
        returns (uint256)
    {
        return transactions[txId].confirmations;
    }
}
