// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title ArbitrumToken
/// @notice Gas-efficient ERC-20 token optimized for Arbitrum L2 deployment.
/// Features batch transfers, owner-controlled minting, and permissionless burning.
/// Standalone implementation with no external imports.
contract ArbitrumToken {
    // ──────────────────── Storage ────────────────────
    // Packed storage layout for L2 gas efficiency
    string public name;
    string public symbol;
    uint8 public constant decimals = 18;
    uint256 public totalSupply;

    address public owner;

    // Gateway address authorized to mint/burn for bridging
    address public gateway;

    mapping(address => uint256) private _balances;
    mapping(address => mapping(address => uint256)) private _allowances;

    // ──────────────────── Events ────────────────────
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event GatewayUpdated(address indexed previousGateway, address indexed newGateway);
    event BatchTransfer(address indexed from, uint256 recipientCount, uint256 totalAmount);

    // ──────────────────── Modifiers ────────────────────
    modifier onlyOwner() {
        require(msg.sender == owner, "ArbitrumToken: caller is not the owner");
        _;
    }

    modifier onlyOwnerOrGateway() {
        require(
            msg.sender == owner || msg.sender == gateway,
            "ArbitrumToken: caller is not the owner or gateway"
        );
        _;
    }

    constructor(string memory _name, string memory _symbol, uint256 _initialSupply) {
        name = _name;
        symbol = _symbol;
        owner = msg.sender;

        if (_initialSupply > 0) {
            _mint(msg.sender, _initialSupply);
        }
    }

    // ──────────────────── ERC-20 Views ────────────────────

    function balanceOf(address account) external view returns (uint256) {
        return _balances[account];
    }

    function allowance(address _owner, address spender) external view returns (uint256) {
        return _allowances[_owner][spender];
    }

    // ──────────────────── ERC-20 Mutative ────────────────────

    function transfer(address to, uint256 amount) external returns (bool) {
        _transfer(msg.sender, to, amount);
        return true;
    }

    function approve(address spender, uint256 amount) external returns (bool) {
        _approve(msg.sender, spender, amount);
        return true;
    }

    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        uint256 currentAllowance = _allowances[from][msg.sender];
        if (currentAllowance != type(uint256).max) {
            require(currentAllowance >= amount, "ArbitrumToken: insufficient allowance");
            unchecked {
                _approve(from, msg.sender, currentAllowance - amount);
            }
        }
        _transfer(from, to, amount);
        return true;
    }

    // ──────────────────── Batch Transfer ────────────────────

    /// @notice Transfer tokens to multiple recipients in a single transaction.
    /// Gas-efficient for Arbitrum L2 by reducing the number of separate transactions.
    /// @param recipients Array of recipient addresses.
    /// @param amounts Array of amounts to transfer to each recipient.
    function batchTransfer(address[] calldata recipients, uint256[] calldata amounts) external {
        require(recipients.length == amounts.length, "ArbitrumToken: length mismatch");
        require(recipients.length > 0, "ArbitrumToken: empty batch");

        uint256 totalAmount = 0;
        for (uint256 i = 0; i < recipients.length; i++) {
            totalAmount += amounts[i];
        }
        require(_balances[msg.sender] >= totalAmount, "ArbitrumToken: insufficient balance for batch");

        unchecked {
            _balances[msg.sender] -= totalAmount;
        }

        for (uint256 i = 0; i < recipients.length; i++) {
            require(recipients[i] != address(0), "ArbitrumToken: transfer to zero address");
            _balances[recipients[i]] += amounts[i];
            emit Transfer(msg.sender, recipients[i], amounts[i]);
        }

        emit BatchTransfer(msg.sender, recipients.length, totalAmount);
    }

    // ──────────────────── Mint / Burn ────────────────────

    /// @notice Mint tokens. Callable by owner or authorized gateway for bridging.
    /// @param to Recipient address.
    /// @param amount Amount to mint.
    function mint(address to, uint256 amount) external onlyOwnerOrGateway {
        _mint(to, amount);
    }

    /// @notice Burn tokens from the caller's balance.
    /// @param amount Amount to burn.
    function burn(uint256 amount) external {
        _burn(msg.sender, amount);
    }

    /// @notice Burn tokens from a specific address (requires allowance).
    /// Used by the gateway during L2-to-L1 withdrawals.
    /// @param from Address to burn from.
    /// @param amount Amount to burn.
    function burnFrom(address from, uint256 amount) external {
        uint256 currentAllowance = _allowances[from][msg.sender];
        if (currentAllowance != type(uint256).max) {
            require(currentAllowance >= amount, "ArbitrumToken: insufficient allowance");
            unchecked {
                _approve(from, msg.sender, currentAllowance - amount);
            }
        }
        _burn(from, amount);
    }

    // ──────────────────── Gateway Management ────────────────────

    /// @notice Set the authorized gateway address for bridge minting/burning.
    /// @param _gateway New gateway address.
    function setGateway(address _gateway) external onlyOwner {
        emit GatewayUpdated(gateway, _gateway);
        gateway = _gateway;
    }

    // ──────────────────── Ownership ────────────────────

    /// @notice Transfer ownership of the token contract.
    /// @param newOwner New owner address.
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "ArbitrumToken: new owner is zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    // ──────────────────── Internal ────────────────────

    function _transfer(address from, address to, uint256 amount) internal {
        require(from != address(0), "ArbitrumToken: transfer from zero address");
        require(to != address(0), "ArbitrumToken: transfer to zero address");
        require(_balances[from] >= amount, "ArbitrumToken: insufficient balance");

        unchecked {
            _balances[from] -= amount;
            _balances[to] += amount;
        }

        emit Transfer(from, to, amount);
    }

    function _mint(address to, uint256 amount) internal {
        require(to != address(0), "ArbitrumToken: mint to zero address");
        totalSupply += amount;
        _balances[to] += amount;

        emit Transfer(address(0), to, amount);
    }

    function _burn(address from, uint256 amount) internal {
        require(_balances[from] >= amount, "ArbitrumToken: burn exceeds balance");
        unchecked {
            _balances[from] -= amount;
        }
        totalSupply -= amount;

        emit Transfer(from, address(0), amount);
    }

    function _approve(address _owner, address spender, uint256 amount) internal {
        require(_owner != address(0), "ArbitrumToken: approve from zero address");
        require(spender != address(0), "ArbitrumToken: approve to zero address");
        _allowances[_owner][spender] = amount;
        emit Approval(_owner, spender, amount);
    }
}
