// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title TokenGateway
/// @notice Bidirectional token gateway for bridging ERC-20 tokens between L1 and L2
/// on Arbitrum. Implements a lock-and-mint / burn-and-release pattern:
/// - L1 -> L2 (deposit): Tokens are locked in the L1 gateway, and equivalent tokens
///   are minted on L2.
/// - L2 -> L1 (withdraw): Tokens are burned on L2, and the locked tokens are released
///   on L1.
contract TokenGateway {
    // ──────────────────── State ────────────────────
    address public owner;
    address public counterpartGateway; // The paired gateway on the other layer

    // Token mappings: L1 token address => L2 token address (and vice versa)
    mapping(address => address) public tokenMapping;

    // Locked balances per token (for L1 gateway: tokens locked when depositing to L2)
    mapping(address => uint256) public lockedBalance;

    // Deposit/withdrawal tracking
    uint256 public nextDepositId;
    uint256 public nextWithdrawalId;

    struct Deposit {
        address sender;
        address token;
        address recipient;
        uint256 amount;
        uint256 timestamp;
        bool finalized;
    }

    struct Withdrawal {
        address sender;
        address token;
        address recipient;
        uint256 amount;
        uint256 timestamp;
        bool finalized;
    }

    mapping(uint256 => Deposit) public deposits;
    mapping(uint256 => Withdrawal) public withdrawals;

    // Reentrancy guard
    uint256 private _locked = 1;

    // ──────────────────── Events ────────────────────
    event TokenDeposited(
        uint256 indexed depositId,
        address indexed token,
        address indexed sender,
        address recipient,
        uint256 amount
    );
    event TokenWithdrawn(
        uint256 indexed withdrawalId,
        address indexed token,
        address indexed sender,
        address recipient,
        uint256 amount
    );
    event DepositFinalized(uint256 indexed depositId);
    event WithdrawalFinalized(uint256 indexed withdrawalId);
    event TokenMappingSet(address indexed l1Token, address indexed l2Token);
    event CounterpartGatewayUpdated(address indexed oldGateway, address indexed newGateway);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // ──────────────────── Modifiers ────────────────────
    modifier onlyOwner() {
        require(msg.sender == owner, "TokenGateway: caller is not the owner");
        _;
    }

    modifier onlyCounterpart() {
        require(
            msg.sender == counterpartGateway || msg.sender == owner,
            "TokenGateway: caller is not the counterpart gateway"
        );
        _;
    }

    modifier nonReentrant() {
        require(_locked == 1, "TokenGateway: reentrancy");
        _locked = 2;
        _;
        _locked = 1;
    }

    constructor(address _counterpartGateway) {
        owner = msg.sender;
        counterpartGateway = _counterpartGateway;
    }

    // ──────────────────── Deposit (L1 -> L2) ────────────────────

    /// @notice Deposit tokens to bridge from the current layer to the other.
    /// On L1: locks tokens and initiates minting on L2.
    /// On L2: burns tokens and initiates release on L1.
    /// @param token Address of the token to deposit.
    /// @param recipient Address to receive tokens on the destination layer.
    /// @param amount Amount of tokens to deposit.
    /// @return depositId The unique identifier for this deposit.
    function deposit(
        address token,
        address recipient,
        uint256 amount
    ) external nonReentrant returns (uint256 depositId) {
        require(token != address(0), "TokenGateway: zero token address");
        require(recipient != address(0), "TokenGateway: zero recipient");
        require(amount > 0, "TokenGateway: zero amount");
        require(tokenMapping[token] != address(0), "TokenGateway: token not mapped");

        // Transfer tokens from sender to this gateway (lock them)
        _safeTransferFrom(token, msg.sender, address(this), amount);
        lockedBalance[token] += amount;

        depositId = nextDepositId++;

        deposits[depositId] = Deposit({
            sender: msg.sender,
            token: token,
            recipient: recipient,
            amount: amount,
            timestamp: block.timestamp,
            finalized: false
        });

        emit TokenDeposited(depositId, token, msg.sender, recipient, amount);
    }

    /// @notice Finalize a deposit on the destination layer.
    /// Called by the counterpart gateway to mint/release tokens.
    /// @param depositId The deposit ID to finalize.
    /// @param token Address of the mapped token on this layer.
    /// @param recipient Address to receive the tokens.
    /// @param amount Amount of tokens to mint/release.
    function finalizeDeposit(
        uint256 depositId,
        address token,
        address recipient,
        uint256 amount
    ) external onlyCounterpart nonReentrant {
        require(token != address(0), "TokenGateway: zero token address");
        require(recipient != address(0), "TokenGateway: zero recipient");
        require(amount > 0, "TokenGateway: zero amount");

        // Mint tokens on the destination layer
        (bool success, ) = token.call(
            abi.encodeWithSignature("mint(address,uint256)", recipient, amount)
        );
        require(success, "TokenGateway: mint failed");

        emit DepositFinalized(depositId);
    }

    // ──────────────────── Withdrawal (L2 -> L1) ────────────────────

    /// @notice Withdraw tokens to bridge back from L2 to L1.
    /// Burns L2 tokens and creates a withdrawal record for L1 release.
    /// @param token Address of the L2 token to withdraw.
    /// @param recipient Address to receive tokens on L1.
    /// @param amount Amount of tokens to withdraw.
    /// @return withdrawalId The unique identifier for this withdrawal.
    function withdraw(
        address token,
        address recipient,
        uint256 amount
    ) external nonReentrant returns (uint256 withdrawalId) {
        require(token != address(0), "TokenGateway: zero token address");
        require(recipient != address(0), "TokenGateway: zero recipient");
        require(amount > 0, "TokenGateway: zero amount");

        // Burn tokens on the source layer (L2)
        // The token must support burnFrom or the sender must have approved this gateway
        (bool success, ) = token.call(
            abi.encodeWithSignature("burnFrom(address,uint256)", msg.sender, amount)
        );
        require(success, "TokenGateway: burn failed");

        withdrawalId = nextWithdrawalId++;

        withdrawals[withdrawalId] = Withdrawal({
            sender: msg.sender,
            token: token,
            recipient: recipient,
            amount: amount,
            timestamp: block.timestamp,
            finalized: false
        });

        emit TokenWithdrawn(withdrawalId, token, msg.sender, recipient, amount);
    }

    /// @notice Finalize a withdrawal on L1, releasing locked tokens.
    /// Called by the counterpart gateway after the challenge period.
    /// @param withdrawalId The withdrawal ID to finalize.
    /// @param token Address of the L1 token to release.
    /// @param recipient Address to receive the tokens.
    /// @param amount Amount of tokens to release.
    function finalizeWithdrawal(
        uint256 withdrawalId,
        address token,
        address recipient,
        uint256 amount
    ) external onlyCounterpart nonReentrant {
        require(token != address(0), "TokenGateway: zero token address");
        require(recipient != address(0), "TokenGateway: zero recipient");
        require(amount > 0, "TokenGateway: zero amount");
        require(lockedBalance[token] >= amount, "TokenGateway: insufficient locked balance");

        lockedBalance[token] -= amount;

        _safeTransfer(token, recipient, amount);

        emit WithdrawalFinalized(withdrawalId);
    }

    // ──────────────────── View Functions ────────────────────

    /// @notice Get the mapped token address for a given source token.
    /// @param sourceToken The token address on the source layer.
    /// @return The mapped token address on the destination layer.
    function getMappedToken(address sourceToken) external view returns (address) {
        return tokenMapping[sourceToken];
    }

    /// @notice Get the locked balance for a token.
    /// @param token The token address.
    /// @return The amount of tokens locked in this gateway.
    function getLockedBalance(address token) external view returns (uint256) {
        return lockedBalance[token];
    }

    /// @notice Get the total number of deposits.
    /// @return The deposit count.
    function depositCount() external view returns (uint256) {
        return nextDepositId;
    }

    /// @notice Get the total number of withdrawals.
    /// @return The withdrawal count.
    function withdrawalCount() external view returns (uint256) {
        return nextWithdrawalId;
    }

    // ──────────────────── Admin ────────────────────

    /// @notice Set the token mapping between L1 and L2 tokens.
    /// @param l1Token Address of the L1 token.
    /// @param l2Token Address of the L2 token.
    function setTokenMapping(address l1Token, address l2Token) external onlyOwner {
        require(l1Token != address(0), "TokenGateway: zero L1 token address");
        require(l2Token != address(0), "TokenGateway: zero L2 token address");
        tokenMapping[l1Token] = l2Token;
        tokenMapping[l2Token] = l1Token;
        emit TokenMappingSet(l1Token, l2Token);
    }

    /// @notice Update the counterpart gateway address.
    /// @param _counterpartGateway New counterpart gateway address.
    function setCounterpartGateway(address _counterpartGateway) external onlyOwner {
        emit CounterpartGatewayUpdated(counterpartGateway, _counterpartGateway);
        counterpartGateway = _counterpartGateway;
    }

    /// @notice Transfer ownership.
    /// @param newOwner New owner address.
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "TokenGateway: zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    // ──────────────────── Internal ────────────────────

    function _safeTransfer(address token, address to, uint256 amount) internal {
        (bool success, bytes memory data) = token.call(
            abi.encodeWithSignature("transfer(address,uint256)", to, amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "TokenGateway: transfer failed");
    }

    function _safeTransferFrom(address token, address from, address to, uint256 amount) internal {
        (bool success, bytes memory data) = token.call(
            abi.encodeWithSignature("transferFrom(address,address,uint256)", from, to, amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "TokenGateway: transferFrom failed");
    }
}
