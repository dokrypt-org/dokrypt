// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title LendingVault
/// @notice A lending and borrowing vault supporting multiple collateral types.
/// Users deposit collateral, borrow against it, repay loans, and can be liquidated
/// if their collateral ratio falls below the threshold.
contract LendingVault {
    // ──────────────────── State ────────────────────
    address public owner;
    address public interestModel;
    address public priceOracle;
    address public borrowToken; // the token users borrow

    uint256 public constant COLLATERAL_FACTOR = 7500; // 75% in basis points
    uint256 public constant LIQUIDATION_THRESHOLD = 8000; // 80% in basis points
    uint256 public constant LIQUIDATION_BONUS = 10500; // 105% in basis points (5% bonus)
    uint256 public constant BASIS_POINTS = 10000;

    uint256 public totalDeposited;
    uint256 public totalBorrowed;
    uint256 public lastAccrualTimestamp;
    uint256 public accruedInterestIndex; // scaled by 1e18

    // Supported collateral tokens
    address[] public supportedCollaterals;
    mapping(address => bool) public isCollateralSupported;

    // User state per collateral type
    // user => collateral token => amount deposited
    mapping(address => mapping(address => uint256)) public userCollateral;
    // user => amount borrowed (in borrowToken terms)
    mapping(address => uint256) public userBorrowShares;
    uint256 public totalBorrowShares;

    // Reentrancy guard
    uint256 private _locked = 1;

    // ──────────────────── Events ────────────────────
    event CollateralDeposited(address indexed user, address indexed token, uint256 amount);
    event CollateralWithdrawn(address indexed user, address indexed token, uint256 amount);
    event Borrowed(address indexed user, uint256 amount, uint256 shares);
    event Repaid(address indexed user, uint256 amount, uint256 shares);
    event Liquidated(
        address indexed liquidator,
        address indexed borrower,
        address indexed collateralToken,
        uint256 debtRepaid,
        uint256 collateralSeized
    );
    event CollateralAdded(address indexed token);
    event InterestAccrued(uint256 interestAmount, uint256 newIndex);

    // ──────────────────── Modifiers ────────────────────
    modifier onlyOwner() {
        require(msg.sender == owner, "LendingVault: not owner");
        _;
    }

    modifier nonReentrant() {
        require(_locked == 1, "LendingVault: reentrancy");
        _locked = 2;
        _;
        _locked = 1;
    }

    constructor(address _borrowToken, address _interestModel, address _priceOracle) {
        require(_borrowToken != address(0), "LendingVault: zero borrow token");
        require(_interestModel != address(0), "LendingVault: zero interest model");
        require(_priceOracle != address(0), "LendingVault: zero oracle");

        owner = msg.sender;
        borrowToken = _borrowToken;
        interestModel = _interestModel;
        priceOracle = _priceOracle;
        lastAccrualTimestamp = block.timestamp;
        accruedInterestIndex = 1e18;
    }

    // ──────────────────── Admin ────────────────────

    function addCollateral(address token) external onlyOwner {
        require(token != address(0), "LendingVault: zero address");
        require(!isCollateralSupported[token], "LendingVault: already supported");
        isCollateralSupported[token] = true;
        supportedCollaterals.push(token);
        emit CollateralAdded(token);
    }

    /// @notice Supply borrow tokens into the vault for others to borrow.
    function supply(uint256 amount) external nonReentrant {
        _safeTransferFrom(borrowToken, msg.sender, address(this), amount);
        totalDeposited += amount;
    }

    // ──────────────────── User Actions ────────────────────

    /// @notice Deposit collateral into the vault.
    function depositCollateral(address token, uint256 amount) external nonReentrant {
        require(isCollateralSupported[token], "LendingVault: unsupported collateral");
        require(amount > 0, "LendingVault: zero amount");

        _safeTransferFrom(token, msg.sender, address(this), amount);
        userCollateral[msg.sender][token] += amount;

        emit CollateralDeposited(msg.sender, token, amount);
    }

    /// @notice Withdraw collateral from the vault.
    function withdrawCollateral(address token, uint256 amount) external nonReentrant {
        require(amount > 0, "LendingVault: zero amount");
        require(userCollateral[msg.sender][token] >= amount, "LendingVault: insufficient collateral");

        userCollateral[msg.sender][token] -= amount;

        // Ensure the user remains solvent after withdrawal
        require(_isSolvent(msg.sender), "LendingVault: would become insolvent");

        _safeTransfer(token, msg.sender, amount);

        emit CollateralWithdrawn(msg.sender, token, amount);
    }

    /// @notice Borrow tokens against deposited collateral.
    function borrow(uint256 amount) external nonReentrant {
        require(amount > 0, "LendingVault: zero amount");
        accrueInterest();

        uint256 shares = totalBorrowed == 0 ? amount : (amount * totalBorrowShares) / totalBorrowed;
        require(shares > 0, "LendingVault: zero shares");

        userBorrowShares[msg.sender] += shares;
        totalBorrowShares += shares;
        totalBorrowed += amount;

        require(_isSolvent(msg.sender), "LendingVault: insufficient collateral");

        _safeTransfer(borrowToken, msg.sender, amount);

        emit Borrowed(msg.sender, amount, shares);
    }

    /// @notice Repay borrowed tokens.
    function repay(uint256 amount) external nonReentrant {
        require(amount > 0, "LendingVault: zero amount");
        accrueInterest();

        uint256 userDebt = _borrowBalance(msg.sender);
        if (amount > userDebt) {
            amount = userDebt;
        }

        uint256 shares = (amount * totalBorrowShares) / totalBorrowed;
        if (shares == 0) shares = 1;
        if (shares > userBorrowShares[msg.sender]) shares = userBorrowShares[msg.sender];

        _safeTransferFrom(borrowToken, msg.sender, address(this), amount);

        userBorrowShares[msg.sender] -= shares;
        totalBorrowShares -= shares;
        totalBorrowed -= amount;

        emit Repaid(msg.sender, amount, shares);
    }

    /// @notice Liquidate an underwater borrower.
    /// @param borrower Address of the borrower to liquidate.
    /// @param collateralToken Collateral token to seize.
    /// @param debtAmount Amount of debt to repay on behalf of the borrower.
    function liquidate(
        address borrower,
        address collateralToken,
        uint256 debtAmount
    ) external nonReentrant {
        accrueInterest();

        require(!_isSolvent(borrower), "LendingVault: borrower is solvent");
        require(debtAmount > 0, "LendingVault: zero debt amount");

        uint256 borrowerDebt = _borrowBalance(borrower);
        require(debtAmount <= borrowerDebt, "LendingVault: excess debt repay");

        // Calculate collateral to seize (debt value + liquidation bonus)
        uint256 debtValueUSD = _getValueUSD(borrowToken, debtAmount);
        uint256 seizeValueUSD = (debtValueUSD * LIQUIDATION_BONUS) / BASIS_POINTS;
        uint256 collateralPrice = _getPrice(collateralToken);
        uint256 seizeAmount = (seizeValueUSD * 1e18) / collateralPrice;

        require(
            userCollateral[borrower][collateralToken] >= seizeAmount,
            "LendingVault: insufficient collateral to seize"
        );

        // Repay debt
        _safeTransferFrom(borrowToken, msg.sender, address(this), debtAmount);

        uint256 shares = (debtAmount * totalBorrowShares) / totalBorrowed;
        if (shares > userBorrowShares[borrower]) shares = userBorrowShares[borrower];

        userBorrowShares[borrower] -= shares;
        totalBorrowShares -= shares;
        totalBorrowed -= debtAmount;

        // Seize collateral
        userCollateral[borrower][collateralToken] -= seizeAmount;
        _safeTransfer(collateralToken, msg.sender, seizeAmount);

        emit Liquidated(msg.sender, borrower, collateralToken, debtAmount, seizeAmount);
    }

    // ──────────────────── Interest ────────────────────

    /// @notice Accrue interest based on the interest rate model.
    function accrueInterest() public {
        uint256 elapsed = block.timestamp - lastAccrualTimestamp;
        if (elapsed == 0) return;

        lastAccrualTimestamp = block.timestamp;

        if (totalBorrowed == 0) return;

        uint256 rate = _getInterestRate();
        uint256 interestAmount = (totalBorrowed * rate * elapsed) / (365 days * 1e18);

        totalBorrowed += interestAmount;
        accruedInterestIndex = (accruedInterestIndex * (1e18 + (rate * elapsed) / 365 days)) / 1e18;

        emit InterestAccrued(interestAmount, accruedInterestIndex);
    }

    // ──────────────────── View Functions ────────────────────

    /// @notice Get the total collateral value in USD for a user.
    function getUserCollateralValueUSD(address user) external view returns (uint256 totalValue) {
        for (uint256 i = 0; i < supportedCollaterals.length; i++) {
            address token = supportedCollaterals[i];
            uint256 amount = userCollateral[user][token];
            if (amount > 0) {
                totalValue += _getValueUSD(token, amount);
            }
        }
    }

    /// @notice Get the current borrow balance for a user (including interest).
    function getBorrowBalance(address user) external view returns (uint256) {
        return _borrowBalance(user);
    }

    /// @notice Check if a user is currently solvent.
    function isSolvent(address user) external view returns (bool) {
        return _isSolvent(user);
    }

    function getSupportedCollaterals() external view returns (address[] memory) {
        return supportedCollaterals;
    }

    // ──────────────────── Internal ────────────────────

    function _borrowBalance(address user) internal view returns (uint256) {
        if (totalBorrowShares == 0) return 0;
        return (userBorrowShares[user] * totalBorrowed) / totalBorrowShares;
    }

    function _isSolvent(address user) internal view returns (bool) {
        uint256 debtValue = _getValueUSD(borrowToken, _borrowBalance(user));
        if (debtValue == 0) return true;

        uint256 collateralValue = 0;
        for (uint256 i = 0; i < supportedCollaterals.length; i++) {
            address token = supportedCollaterals[i];
            uint256 amount = userCollateral[user][token];
            if (amount > 0) {
                collateralValue += _getValueUSD(token, amount);
            }
        }

        // collateralValue * collateralFactor >= debtValue
        return (collateralValue * COLLATERAL_FACTOR) / BASIS_POINTS >= debtValue;
    }

    function _getValueUSD(address token, uint256 amount) internal view returns (uint256) {
        uint256 price = _getPrice(token);
        return (amount * price) / 1e18;
    }

    function _getPrice(address token) internal view returns (uint256) {
        (bool success, bytes memory data) = priceOracle.staticcall(
            abi.encodeWithSignature("getPrice(address)", token)
        );
        require(success && data.length >= 32, "LendingVault: oracle query failed");
        uint256 price = abi.decode(data, (uint256));
        require(price > 0, "LendingVault: zero price");
        return price;
    }

    function _getInterestRate() internal view returns (uint256) {
        (bool success, bytes memory data) = interestModel.staticcall(
            abi.encodeWithSignature("getInterestRate(uint256,uint256)", totalBorrowed, totalDeposited)
        );
        require(success && data.length >= 32, "LendingVault: interest query failed");
        return abi.decode(data, (uint256));
    }

    function _safeTransfer(address token, address to, uint256 amount) internal {
        (bool success, bytes memory data) = token.call(
            abi.encodeWithSignature("transfer(address,uint256)", to, amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "LendingVault: transfer failed");
    }

    function _safeTransferFrom(address token, address from, address to, uint256 amount) internal {
        (bool success, bytes memory data) = token.call(
            abi.encodeWithSignature("transferFrom(address,address,uint256)", from, to, amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "LendingVault: transferFrom failed");
    }
}
