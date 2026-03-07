package testrunner

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
)

type Transaction struct {
	Hash      string          `json:"hash"`
	Status    uint64          `json:"status"` // 1 = success, 0 = reverted
	GasUsed   uint64          `json:"gas_used"`
	Logs      []Log           `json:"logs"`
	RevertMsg string          `json:"revert_msg,omitempty"`
}

type Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
	Name    string   `json:"name"`
	Args    map[string]any `json:"args,omitempty"`
}

func AssertTxSuccess(t testing.TB, tx *Transaction) {
	t.Helper()
	if tx.Status != 1 {
		t.Errorf("expected transaction to succeed, but it reverted: %s", tx.RevertMsg)
	}
}

func AssertTxReverted(t testing.TB, tx *Transaction) {
	t.Helper()
	if tx.Status != 0 {
		t.Errorf("expected transaction to revert, but it succeeded")
	}
}

func AssertTxRevertedWith(t testing.TB, tx *Transaction, reason string) {
	t.Helper()
	if tx.Status != 0 {
		t.Errorf("expected transaction to revert with %q, but it succeeded", reason)
		return
	}
	if tx.RevertMsg != reason {
		t.Errorf("expected revert reason %q, got %q", reason, tx.RevertMsg)
	}
}

func AssertGasBelow(t testing.TB, tx *Transaction, maxGas uint64) {
	t.Helper()
	if tx.GasUsed > maxGas {
		t.Errorf("gas usage %d exceeds max %d", tx.GasUsed, maxGas)
	}
}

func AssertEvent(t testing.TB, tx *Transaction, eventName string, args map[string]any) {
	t.Helper()
	for _, log := range tx.Logs {
		if log.Name == eventName {
			if args != nil {
				for key, expected := range args {
					actual, ok := log.Args[key]
					if !ok {
						t.Errorf("event %q missing arg %q", eventName, key)
						continue
					}
					if fmt.Sprint(actual) != fmt.Sprint(expected) {
						t.Errorf("event %q arg %q: expected %v, got %v", eventName, key, expected, actual)
					}
				}
			}
			return
		}
	}
	t.Errorf("event %q not found in transaction logs", eventName)
}

func AssertEventCount(t testing.TB, tx *Transaction, eventName string, count int) {
	t.Helper()
	actual := 0
	for _, log := range tx.Logs {
		if log.Name == eventName {
			actual++
		}
	}
	if actual != count {
		t.Errorf("expected %d %q events, got %d", count, eventName, actual)
	}
}

func AssertNoEvent(t testing.TB, tx *Transaction, eventName string) {
	t.Helper()
	for _, log := range tx.Logs {
		if log.Name == eventName {
			t.Errorf("unexpected event %q found in transaction logs", eventName)
			return
		}
	}
}

func AssertBalance(t testing.TB, actual, expected string) {
	t.Helper()
	a, ok := new(big.Int).SetString(actual, 10)
	if !ok {
		t.Errorf("invalid actual balance: %q", actual)
		return
	}
	e, ok := new(big.Int).SetString(expected, 10)
	if !ok {
		t.Errorf("invalid expected balance: %q", expected)
		return
	}
	if a.Cmp(e) != 0 {
		t.Errorf("balance mismatch: expected %s, got %s", expected, actual)
	}
}

func AssertETHBalance(t testing.TB, actual *big.Int, expected *big.Int) {
	t.Helper()
	if actual.Cmp(expected) != 0 {
		t.Errorf("ETH balance mismatch: expected %s, got %s", expected, actual)
	}
}

func AssertJSONEqual(t testing.TB, actual, expected string) {
	t.Helper()
	var a, e any
	if err := json.Unmarshal([]byte(actual), &a); err != nil {
		t.Errorf("invalid actual JSON: %v", err)
		return
	}
	if err := json.Unmarshal([]byte(expected), &e); err != nil {
		t.Errorf("invalid expected JSON: %v", err)
		return
	}
	actualBytes, _ := json.Marshal(a)
	expectedBytes, _ := json.Marshal(e)
	if string(actualBytes) != string(expectedBytes) {
		t.Errorf("JSON mismatch:\n  expected: %s\n  got:      %s", expected, actual)
	}
}
