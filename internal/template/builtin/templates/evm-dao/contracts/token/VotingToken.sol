// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title VotingToken
/// @notice ERC-20 token with vote delegation and checkpoint-based historical voting power (ERC20Votes-style).
contract VotingToken {
    // ---------------------------------------------------------------
    // ERC-20 storage
    // ---------------------------------------------------------------
    string public name;
    string public symbol;
    uint8 public constant decimals = 18;
    uint256 public totalSupply;

    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    // ---------------------------------------------------------------
    // Voting / delegation storage
    // ---------------------------------------------------------------
    mapping(address => address) private _delegates;

    struct Checkpoint {
        uint256 fromBlock;
        uint256 votes;
    }

    mapping(address => Checkpoint[]) private _checkpoints;

    address public owner;

    // ---------------------------------------------------------------
    // Events
    // ---------------------------------------------------------------
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event DelegateChanged(address indexed delegator, address indexed fromDelegate, address indexed toDelegate);
    event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance);

    // ---------------------------------------------------------------
    // Modifiers
    // ---------------------------------------------------------------
    modifier onlyOwner() {
        require(msg.sender == owner, "VotingToken: caller is not the owner");
        _;
    }

    // ---------------------------------------------------------------
    // Constructor
    // ---------------------------------------------------------------
    constructor(string memory _name, string memory _symbol, uint256 _initialSupply) {
        name = _name;
        symbol = _symbol;
        owner = msg.sender;
        if (_initialSupply > 0) {
            _mint(msg.sender, _initialSupply);
        }
    }

    // ---------------------------------------------------------------
    // ERC-20 functions
    // ---------------------------------------------------------------
    function approve(address spender, uint256 amount) external returns (bool) {
        allowance[msg.sender][spender] = amount;
        emit Approval(msg.sender, spender, amount);
        return true;
    }

    function transfer(address to, uint256 amount) external returns (bool) {
        _transfer(msg.sender, to, amount);
        return true;
    }

    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        uint256 currentAllowance = allowance[from][msg.sender];
        require(currentAllowance >= amount, "VotingToken: insufficient allowance");
        unchecked {
            allowance[from][msg.sender] = currentAllowance - amount;
        }
        _transfer(from, to, amount);
        return true;
    }

    function mint(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
    }

    // ---------------------------------------------------------------
    // Delegation & voting power
    // ---------------------------------------------------------------
    function delegates(address account) external view returns (address) {
        return _delegates[account];
    }

    function delegate(address delegatee) external {
        _delegate(msg.sender, delegatee);
    }

    function getVotes(address account) external view returns (uint256) {
        uint256 nCheckpoints = _checkpoints[account].length;
        return nCheckpoints == 0 ? 0 : _checkpoints[account][nCheckpoints - 1].votes;
    }

    /// @notice Returns the voting power of `account` at a specific `blockNumber`.
    function getPastVotes(address account, uint256 blockNumber) external view returns (uint256) {
        require(blockNumber < block.number, "VotingToken: block not yet mined");
        return _checkpointsLookup(_checkpoints[account], blockNumber);
    }

    function numCheckpoints(address account) external view returns (uint256) {
        return _checkpoints[account].length;
    }

    // ---------------------------------------------------------------
    // Internal helpers
    // ---------------------------------------------------------------
    function _transfer(address from, address to, uint256 amount) internal {
        require(from != address(0), "VotingToken: transfer from zero address");
        require(to != address(0), "VotingToken: transfer to zero address");
        require(balanceOf[from] >= amount, "VotingToken: insufficient balance");

        unchecked {
            balanceOf[from] -= amount;
        }
        balanceOf[to] += amount;
        emit Transfer(from, to, amount);

        _moveDelegates(_delegates[from], _delegates[to], amount);
    }

    function _mint(address to, uint256 amount) internal {
        require(to != address(0), "VotingToken: mint to zero address");
        totalSupply += amount;
        balanceOf[to] += amount;
        emit Transfer(address(0), to, amount);

        _moveDelegates(address(0), _delegates[to], amount);
    }

    function _delegate(address delegator, address delegatee) internal {
        address currentDelegate = _delegates[delegator];
        _delegates[delegator] = delegatee;
        emit DelegateChanged(delegator, currentDelegate, delegatee);
        _moveDelegates(currentDelegate, delegatee, balanceOf[delegator]);
    }

    function _moveDelegates(address from, address to, uint256 amount) internal {
        if (amount == 0) return;

        if (from != address(0)) {
            uint256 nCheckpoints = _checkpoints[from].length;
            uint256 oldVotes = nCheckpoints == 0 ? 0 : _checkpoints[from][nCheckpoints - 1].votes;
            uint256 newVotes = oldVotes - amount;
            _writeCheckpoint(from, oldVotes, newVotes);
        }

        if (to != address(0)) {
            uint256 nCheckpoints = _checkpoints[to].length;
            uint256 oldVotes = nCheckpoints == 0 ? 0 : _checkpoints[to][nCheckpoints - 1].votes;
            uint256 newVotes = oldVotes + amount;
            _writeCheckpoint(to, oldVotes, newVotes);
        }
    }

    function _writeCheckpoint(address account, uint256 oldVotes, uint256 newVotes) internal {
        uint256 nCheckpoints = _checkpoints[account].length;
        if (nCheckpoints > 0 && _checkpoints[account][nCheckpoints - 1].fromBlock == block.number) {
            _checkpoints[account][nCheckpoints - 1].votes = newVotes;
        } else {
            _checkpoints[account].push(Checkpoint(block.number, newVotes));
        }
        emit DelegateVotesChanged(account, oldVotes, newVotes);
    }

    /// @dev Binary search for the checkpoint at or before `blockNumber`.
    function _checkpointsLookup(Checkpoint[] storage ckpts, uint256 blockNumber) internal view returns (uint256) {
        uint256 len = ckpts.length;
        if (len == 0) return 0;
        if (ckpts[len - 1].fromBlock <= blockNumber) return ckpts[len - 1].votes;
        if (ckpts[0].fromBlock > blockNumber) return 0;

        uint256 lo = 0;
        uint256 hi = len;
        while (lo < hi) {
            uint256 mid = (lo + hi) / 2;
            if (ckpts[mid].fromBlock <= blockNumber) {
                lo = mid + 1;
            } else {
                hi = mid;
            }
        }
        return ckpts[lo - 1].votes;
    }
}
