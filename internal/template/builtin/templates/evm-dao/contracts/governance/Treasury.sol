// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title Treasury
/// @notice DAO treasury that holds ETH and ERC-20 tokens. Only the designated
///         controller (typically the Timelock) can authorize withdrawals.

/// @dev Minimal ERC-20 interface used for token interactions.
interface IERC20 {
    function balanceOf(address account) external view returns (uint256);
    function transfer(address to, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
}

contract Treasury {
    // ---------------------------------------------------------------
    // State
    // ---------------------------------------------------------------
    address public controller; // timelock / governor address
    address public admin;      // can update controller

    // ---------------------------------------------------------------
    // Events
    // ---------------------------------------------------------------
    event Deposited(address indexed token, address indexed from, uint256 amount);
    event ETHDeposited(address indexed from, uint256 amount);
    event Withdrawn(address indexed token, address indexed to, uint256 amount);
    event ETHWithdrawn(address indexed to, uint256 amount);
    event ControllerUpdated(address indexed oldController, address indexed newController);

    // ---------------------------------------------------------------
    // Modifiers
    // ---------------------------------------------------------------
    modifier onlyController() {
        require(
            msg.sender == controller || msg.sender == admin,
            "Treasury: caller is not controller"
        );
        _;
    }

    modifier onlyAdmin() {
        require(msg.sender == admin, "Treasury: caller is not admin");
        _;
    }

    // ---------------------------------------------------------------
    // Constructor
    // ---------------------------------------------------------------
    constructor(address _controller) {
        require(_controller != address(0), "Treasury: zero controller address");
        controller = _controller;
        admin = msg.sender;
    }

    // ---------------------------------------------------------------
    // Receive ETH
    // ---------------------------------------------------------------

    /// @notice Accept plain ETH transfers.
    receive() external payable {
        emit ETHDeposited(msg.sender, msg.value);
    }

    /// @notice Fallback also accepts ETH.
    fallback() external payable {
        emit ETHDeposited(msg.sender, msg.value);
    }

    // ---------------------------------------------------------------
    // Deposit
    // ---------------------------------------------------------------

    /// @notice Deposit ERC-20 tokens into the treasury. Caller must have
    ///         approved the treasury to spend `amount` of `token` first.
    function deposit(address token, uint256 amount) external {
        require(token != address(0), "Treasury: zero token address");
        require(amount > 0, "Treasury: zero amount");

        bool success = IERC20(token).transferFrom(msg.sender, address(this), amount);
        require(success, "Treasury: transferFrom failed");

        emit Deposited(token, msg.sender, amount);
    }

    // ---------------------------------------------------------------
    // Withdraw (controller only)
    // ---------------------------------------------------------------

    /// @notice Withdraw ERC-20 tokens from the treasury.
    function withdraw(address token, address to, uint256 amount) external onlyController {
        require(token != address(0), "Treasury: zero token address");
        require(to != address(0), "Treasury: zero recipient");
        require(amount > 0, "Treasury: zero amount");

        bool success = IERC20(token).transfer(to, amount);
        require(success, "Treasury: transfer failed");

        emit Withdrawn(token, to, amount);
    }

    /// @notice Withdraw ETH from the treasury.
    function withdrawETH(address payable to, uint256 amount) external onlyController {
        require(to != address(0), "Treasury: zero recipient");
        require(amount > 0, "Treasury: zero amount");
        require(address(this).balance >= amount, "Treasury: insufficient ETH balance");

        (bool success, ) = to.call{value: amount}("");
        require(success, "Treasury: ETH transfer failed");

        emit ETHWithdrawn(to, amount);
    }

    // ---------------------------------------------------------------
    // View functions
    // ---------------------------------------------------------------

    /// @notice Returns the treasury balance of an ERC-20 token.
    function getBalance(address token) external view returns (uint256) {
        return IERC20(token).balanceOf(address(this));
    }

    /// @notice Returns the ETH balance held by the treasury.
    function getETHBalance() external view returns (uint256) {
        return address(this).balance;
    }

    // ---------------------------------------------------------------
    // Admin
    // ---------------------------------------------------------------

    /// @notice Update the controller address (e.g. when migrating the timelock).
    function setController(address _controller) external onlyAdmin {
        require(_controller != address(0), "Treasury: zero address");
        emit ControllerUpdated(controller, _controller);
        controller = _controller;
    }

    /// @notice Transfer admin role.
    function transferAdmin(address newAdmin) external onlyAdmin {
        require(newAdmin != address(0), "Treasury: zero address");
        admin = newAdmin;
    }
}
