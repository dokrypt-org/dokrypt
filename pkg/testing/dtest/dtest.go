package dtest

import (
	"math/big"
	"testing"

	"github.com/dokrypt/dokrypt/internal/testrunner"
)

func AssertTxSuccess(t testing.TB, tx *testrunner.Transaction) {
	t.Helper()
	testrunner.AssertTxSuccess(t, tx)
}

func AssertTxReverted(t testing.TB, tx *testrunner.Transaction) {
	t.Helper()
	testrunner.AssertTxReverted(t, tx)
}

func AssertTxRevertedWith(t testing.TB, tx *testrunner.Transaction, reason string) {
	t.Helper()
	testrunner.AssertTxRevertedWith(t, tx, reason)
}

func AssertGasBelow(t testing.TB, tx *testrunner.Transaction, maxGas uint64) {
	t.Helper()
	testrunner.AssertGasBelow(t, tx, maxGas)
}

func AssertEvent(t testing.TB, tx *testrunner.Transaction, eventName string, args map[string]any) {
	t.Helper()
	testrunner.AssertEvent(t, tx, eventName, args)
}

func AssertEventCount(t testing.TB, tx *testrunner.Transaction, eventName string, count int) {
	t.Helper()
	testrunner.AssertEventCount(t, tx, eventName, count)
}

func AssertNoEvent(t testing.TB, tx *testrunner.Transaction, eventName string) {
	t.Helper()
	testrunner.AssertNoEvent(t, tx, eventName)
}

func AssertBalance(t testing.TB, actual, expected string) {
	t.Helper()
	testrunner.AssertBalance(t, actual, expected)
}

func AssertETHBalance(t testing.TB, actual *big.Int, expected *big.Int) {
	t.Helper()
	testrunner.AssertETHBalance(t, actual, expected)
}

func AssertJSONEqual(t testing.TB, actual, expected string) {
	t.Helper()
	testrunner.AssertJSONEqual(t, actual, expected)
}
