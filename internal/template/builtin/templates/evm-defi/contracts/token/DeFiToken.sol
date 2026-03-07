// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title DeFiToken
/// @notice Full ERC-20 token with mint, burn, and governance delegation.
/// Standalone implementation with no external imports.
contract DeFiToken {
    string public name;
    string public symbol;
    uint8 public constant decimals = 18;
    uint256 public totalSupply;

    address public owner;

    mapping(address => uint256) private _balances;
    mapping(address => mapping(address => uint256)) private _allowances;

    // Governance delegation
    mapping(address => address) public delegates;
    mapping(address => uint256) public votingPower;

    // --- Events ---
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event DelegateChanged(address indexed delegator, address indexed fromDelegate, address indexed toDelegate);

    // --- Modifiers ---
    modifier onlyOwner() {
        require(msg.sender == owner, "DeFiToken: caller is not the owner");
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

    // --- ERC-20 Views ---

    function balanceOf(address account) external view returns (uint256) {
        return _balances[account];
    }

    function allowance(address _owner, address spender) external view returns (uint256) {
        return _allowances[_owner][spender];
    }

    // --- ERC-20 Mutative ---

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
            require(currentAllowance >= amount, "DeFiToken: insufficient allowance");
            unchecked {
                _approve(from, msg.sender, currentAllowance - amount);
            }
        }
        _transfer(from, to, amount);
        return true;
    }

    // --- Mint / Burn ---

    function mint(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
    }

    function burn(uint256 amount) external {
        _burn(msg.sender, amount);
    }

    // --- Governance ---

    function delegate(address delegatee) external {
        address currentDelegate = delegates[msg.sender];
        delegates[msg.sender] = delegatee;

        if (currentDelegate != address(0)) {
            votingPower[currentDelegate] -= _balances[msg.sender];
        }
        if (delegatee != address(0)) {
            votingPower[delegatee] += _balances[msg.sender];
        }

        emit DelegateChanged(msg.sender, currentDelegate, delegatee);
    }

    // --- Ownership ---

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "DeFiToken: new owner is zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    // --- Internal ---

    function _transfer(address from, address to, uint256 amount) internal {
        require(from != address(0), "DeFiToken: transfer from zero address");
        require(to != address(0), "DeFiToken: transfer to zero address");
        require(_balances[from] >= amount, "DeFiToken: insufficient balance");

        unchecked {
            _balances[from] -= amount;
            _balances[to] += amount;
        }

        // Update voting power if delegated
        if (delegates[from] != address(0)) {
            votingPower[delegates[from]] -= amount;
        }
        if (delegates[to] != address(0)) {
            votingPower[delegates[to]] += amount;
        }

        emit Transfer(from, to, amount);
    }

    function _mint(address to, uint256 amount) internal {
        require(to != address(0), "DeFiToken: mint to zero address");
        totalSupply += amount;
        _balances[to] += amount;

        if (delegates[to] != address(0)) {
            votingPower[delegates[to]] += amount;
        }

        emit Transfer(address(0), to, amount);
    }

    function _burn(address from, uint256 amount) internal {
        require(_balances[from] >= amount, "DeFiToken: burn exceeds balance");
        unchecked {
            _balances[from] -= amount;
        }
        totalSupply -= amount;

        if (delegates[from] != address(0)) {
            votingPower[delegates[from]] -= amount;
        }

        emit Transfer(from, address(0), amount);
    }

    function _approve(address _owner, address spender, uint256 amount) internal {
        require(_owner != address(0), "DeFiToken: approve from zero address");
        require(spender != address(0), "DeFiToken: approve to zero address");
        _allowances[_owner][spender] = amount;
        emit Approval(_owner, spender, amount);
    }
}
