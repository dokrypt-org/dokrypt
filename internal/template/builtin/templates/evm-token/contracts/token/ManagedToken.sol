// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title ManagedToken
/// @notice Full-featured ERC-20 token with mint, burn, pause, and ownership.
contract ManagedToken {
    // ----------------------------------------------------------------
    // State
    // ----------------------------------------------------------------
    string public name;
    string public symbol;
    uint8  public constant decimals = 18;
    uint256 public totalSupply;

    address public owner;
    bool    public paused;

    mapping(address => uint256)                     private _balances;
    mapping(address => mapping(address => uint256)) private _allowances;

    // ----------------------------------------------------------------
    // Events
    // ----------------------------------------------------------------
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event Paused(address account);
    event Unpaused(address account);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // ----------------------------------------------------------------
    // Modifiers
    // ----------------------------------------------------------------
    modifier onlyOwner() {
        require(msg.sender == owner, "ManagedToken: caller is not the owner");
        _;
    }

    modifier whenNotPaused() {
        require(!paused, "ManagedToken: token transfers are paused");
        _;
    }

    modifier whenPaused() {
        require(paused, "ManagedToken: token transfers are not paused");
        _;
    }

    // ----------------------------------------------------------------
    // Constructor
    // ----------------------------------------------------------------
    constructor(
        string memory _name,
        string memory _symbol,
        uint256 _initialSupply
    ) {
        name   = _name;
        symbol = _symbol;
        owner  = msg.sender;

        if (_initialSupply > 0) {
            _mint(msg.sender, _initialSupply);
        }
    }

    // ----------------------------------------------------------------
    // ERC-20 View Functions
    // ----------------------------------------------------------------
    function balanceOf(address account) external view returns (uint256) {
        return _balances[account];
    }

    function allowance(address _owner, address spender) external view returns (uint256) {
        return _allowances[_owner][spender];
    }

    // ----------------------------------------------------------------
    // ERC-20 State-Changing Functions
    // ----------------------------------------------------------------
    function transfer(address to, uint256 amount) external whenNotPaused returns (bool) {
        _transfer(msg.sender, to, amount);
        return true;
    }

    function approve(address spender, uint256 amount) external returns (bool) {
        _approve(msg.sender, spender, amount);
        return true;
    }

    function transferFrom(
        address from,
        address to,
        uint256 amount
    ) external whenNotPaused returns (bool) {
        uint256 currentAllowance = _allowances[from][msg.sender];
        require(currentAllowance >= amount, "ManagedToken: transfer amount exceeds allowance");
        unchecked {
            _approve(from, msg.sender, currentAllowance - amount);
        }
        _transfer(from, to, amount);
        return true;
    }

    // ----------------------------------------------------------------
    // Mint / Burn
    // ----------------------------------------------------------------
    function mint(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
    }

    function burn(uint256 amount) external {
        require(_balances[msg.sender] >= amount, "ManagedToken: burn amount exceeds balance");
        unchecked {
            _balances[msg.sender] -= amount;
        }
        totalSupply -= amount;
        emit Transfer(msg.sender, address(0), amount);
    }

    // ----------------------------------------------------------------
    // Pause / Unpause
    // ----------------------------------------------------------------
    function pause() external onlyOwner whenNotPaused {
        paused = true;
        emit Paused(msg.sender);
    }

    function unpause() external onlyOwner whenPaused {
        paused = false;
        emit Unpaused(msg.sender);
    }

    // ----------------------------------------------------------------
    // Ownership
    // ----------------------------------------------------------------
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "ManagedToken: new owner is the zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    // ----------------------------------------------------------------
    // Internal helpers
    // ----------------------------------------------------------------
    function _transfer(address from, address to, uint256 amount) internal {
        require(from != address(0), "ManagedToken: transfer from the zero address");
        require(to   != address(0), "ManagedToken: transfer to the zero address");
        require(_balances[from] >= amount, "ManagedToken: transfer amount exceeds balance");
        unchecked {
            _balances[from] -= amount;
        }
        _balances[to] += amount;
        emit Transfer(from, to, amount);
    }

    function _mint(address to, uint256 amount) internal {
        require(to != address(0), "ManagedToken: mint to the zero address");
        totalSupply      += amount;
        _balances[to]    += amount;
        emit Transfer(address(0), to, amount);
    }

    function _approve(address _owner, address spender, uint256 amount) internal {
        require(_owner  != address(0), "ManagedToken: approve from the zero address");
        require(spender != address(0), "ManagedToken: approve to the zero address");
        _allowances[_owner][spender] = amount;
        emit Approval(_owner, spender, amount);
    }
}
