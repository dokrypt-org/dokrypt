package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/common"
)

type AccountManager struct {
	rpcClient *RPCClient
	accounts  []chain.Account
	seed      []byte
	engine    string // "anvil", "hardhat", or "geth"
}

func NewAccountManager(rpcClient *RPCClient, count int, seed []byte, engine string) (*AccountManager, error) {
	if seed == nil {
		seed = common.DefaultSeed
	}
	if count <= 0 {
		count = 10
	}

	infos, err := common.GenerateAccounts(seed, count)
	if err != nil {
		return nil, fmt.Errorf("failed to generate accounts: %w", err)
	}

	accounts := make([]chain.Account, len(infos))
	for i, info := range infos {
		accounts[i] = chain.Account{
			Address:    info.Address,
			PrivateKey: info.PrivateKey,
			Label:      info.Label,
		}
	}

	return &AccountManager{
		rpcClient: rpcClient,
		accounts:  accounts,
		seed:      seed,
		engine:    engine,
	}, nil
}

func (a *AccountManager) FundAccounts(ctx context.Context, balanceWei *big.Int) error {
	method := "anvil_setBalance"
	if a.engine == "hardhat" {
		method = "hardhat_setBalance"
	} else if a.engine == "geth" {
		return fmt.Errorf("direct balance setting not supported in geth dev mode")
	}

	hexBalance := fmt.Sprintf("0x%x", balanceWei)
	for i, acct := range a.accounts {
		slog.Debug("funding account", "address", acct.Address, "label", acct.Label)
		_, err := a.rpcClient.Call(ctx, method, acct.Address, hexBalance)
		if err != nil {
			return fmt.Errorf("failed to fund account %s: %w", acct.Address, err)
		}
		a.accounts[i].Balance = new(big.Int).Set(balanceWei)
	}
	return nil
}

func (a *AccountManager) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	result, err := a.rpcClient.Call(ctx, "eth_getBalance", address, "latest")
	if err != nil {
		return nil, err
	}
	var hexBalance string
	if err := json.Unmarshal(result, &hexBalance); err != nil {
		return nil, fmt.Errorf("failed to parse balance: %w", err)
	}
	balance := new(big.Int)
	if len(hexBalance) > 2 && hexBalance[:2] == "0x" {
		balance.SetString(hexBalance[2:], 16)
	} else {
		balance.SetString(hexBalance, 16)
	}
	return balance, nil
}

func (a *AccountManager) List(ctx context.Context) ([]chain.Account, error) {
	for i := range a.accounts {
		bal, err := a.GetBalance(ctx, a.accounts[i].Address)
		if err != nil {
			slog.Warn("failed to get balance", "address", a.accounts[i].Address, "error", err)
			continue
		}
		a.accounts[i].Balance = bal
	}
	return a.accounts, nil
}

func (a *AccountManager) Import(privateKeyHex string, label string) (*chain.Account, error) {
	key, err := common.PrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	address := common.AddressFromPrivateKey(key)

	acct := chain.Account{
		Address:    address,
		PrivateKey: common.PrivateKeyToHex(key),
		Label:      label,
	}
	a.accounts = append(a.accounts, acct)
	return &acct, nil
}

func (a *AccountManager) SetLabel(address string, label string) error {
	for i := range a.accounts {
		if a.accounts[i].Address == address {
			a.accounts[i].Label = label
			return nil
		}
	}
	return fmt.Errorf("account %s not found", address)
}

func (a *AccountManager) Accounts() []chain.Account {
	return a.accounts
}
