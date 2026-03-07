package testrunner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGasTracker_NotNil(t *testing.T) {
	gt := NewGasTracker()
	require.NotNil(t, gt)
}

func TestNewGasTracker_StartsEmpty(t *testing.T) {
	gt := NewGasTracker()
	report := gt.Report()

	require.NotNil(t, report)
	assert.Empty(t, report.Entries)
}

func TestRecord_SingleEntry(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("ERC20", "transfer", 21000)

	report := gt.Report()
	require.Len(t, report.Entries, 1)
	assert.Equal(t, "ERC20", report.Entries[0].Contract)
	assert.Equal(t, "transfer", report.Entries[0].Method)
}

func TestRecord_MultipleCallsSameMethod(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("Token", "mint", 50_000)
	gt.Record("Token", "mint", 60_000)
	gt.Record("Token", "mint", 70_000)

	report := gt.Report()
	require.Len(t, report.Entries, 1)
	assert.Equal(t, 3, report.Entries[0].Calls)
}

func TestRecord_DifferentContractsSameMethod_AreDistinct(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("TokenA", "transfer", 21_000)
	gt.Record("TokenB", "transfer", 22_000)

	report := gt.Report()
	assert.Len(t, report.Entries, 2)
}

func TestRecord_DifferentMethodsSameContract_AreDistinct(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("ERC20", "transfer", 21_000)
	gt.Record("ERC20", "approve", 46_000)
	gt.Record("ERC20", "transferFrom", 55_000)

	report := gt.Report()
	assert.Len(t, report.Entries, 3)
}

func TestReport_SingleCall_MinAvgMaxAreEqual(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("MyContract", "doSomething", 100_000)

	report := gt.Report()
	require.Len(t, report.Entries, 1)
	e := report.Entries[0]

	assert.Equal(t, uint64(100_000), e.MinGas)
	assert.Equal(t, uint64(100_000), e.AvgGas)
	assert.Equal(t, uint64(100_000), e.MaxGas)
	assert.Equal(t, 1, e.Calls)
}

func TestReport_TwoCalls_MinAndMax(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("C", "m", 10_000)
	gt.Record("C", "m", 20_000)

	report := gt.Report()
	require.Len(t, report.Entries, 1)
	e := report.Entries[0]

	assert.Equal(t, uint64(10_000), e.MinGas)
	assert.Equal(t, uint64(20_000), e.MaxGas)
	assert.Equal(t, uint64(15_000), e.AvgGas)
	assert.Equal(t, 2, e.Calls)
}

func TestReport_ThreeCalls_CorrectMinAvgMax(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("Vault", "deposit", 100)
	gt.Record("Vault", "deposit", 200)
	gt.Record("Vault", "deposit", 300)

	report := gt.Report()
	require.Len(t, report.Entries, 1)
	e := report.Entries[0]

	assert.Equal(t, uint64(100), e.MinGas)
	assert.Equal(t, uint64(200), e.AvgGas)
	assert.Equal(t, uint64(300), e.MaxGas)
}

func TestReport_LargeValues_NoOverflow(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("ComplexContract", "complexOp", 5_000_000)
	gt.Record("ComplexContract", "complexOp", 3_000_000)

	report := gt.Report()
	require.Len(t, report.Entries, 1)
	e := report.Entries[0]

	assert.Equal(t, uint64(3_000_000), e.MinGas)
	assert.Equal(t, uint64(4_000_000), e.AvgGas)
	assert.Equal(t, uint64(5_000_000), e.MaxGas)
}

func TestReport_FirstEntryIsAlwaysMin(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("C", "m", 500)
	gt.Record("C", "m", 100) // smaller than the first entry
	gt.Record("C", "m", 300)

	report := gt.Report()
	assert.Equal(t, uint64(100), report.Entries[0].MinGas)
	assert.Equal(t, uint64(500), report.Entries[0].MaxGas)
}

func TestReport_EntriesSortedAlphabetically(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("Token", "transfer", 21_000)
	gt.Record("Token", "approve", 46_000)
	gt.Record("Vault", "deposit", 80_000)
	gt.Record("Aave", "borrow", 120_000)

	report := gt.Report()
	require.Len(t, report.Entries, 4)

	assert.Equal(t, "Aave", report.Entries[0].Contract)
	assert.Equal(t, "borrow", report.Entries[0].Method)

	assert.Equal(t, "Token", report.Entries[1].Contract)
	assert.Equal(t, "approve", report.Entries[1].Method)

	assert.Equal(t, "Token", report.Entries[2].Contract)
	assert.Equal(t, "transfer", report.Entries[2].Method)

	assert.Equal(t, "Vault", report.Entries[3].Contract)
	assert.Equal(t, "deposit", report.Entries[3].Method)
}

func TestReport_EachEntryHasCorrectCallCount(t *testing.T) {
	gt := NewGasTracker()
	gt.Record("C", "a", 1)
	gt.Record("C", "a", 2)
	gt.Record("C", "b", 3)

	report := gt.Report()
	require.Len(t, report.Entries, 2)

	var aEntry, bEntry GasMethodSummary
	for _, e := range report.Entries {
		switch e.Method {
		case "a":
			aEntry = e
		case "b":
			bEntry = e
		}
	}

	assert.Equal(t, 2, aEntry.Calls)
	assert.Equal(t, 1, bEntry.Calls)
}

func TestSplitKey_NormalKey(t *testing.T) {
	contract, method := splitKey("ERC20.transfer")
	assert.Equal(t, "ERC20", contract)
	assert.Equal(t, "transfer", method)
}

func TestSplitKey_NoSeparator_ReturnsFullKeyAndEmpty(t *testing.T) {
	contract, method := splitKey("nodot")
	assert.Equal(t, "nodot", contract)
	assert.Equal(t, "", method)
}

func TestSplitKey_MultipleDots_SplitsOnLastDot(t *testing.T) {
	contract, method := splitKey("My.Contract.method")
	assert.Equal(t, "My.Contract", contract)
	assert.Equal(t, "method", method)
}
