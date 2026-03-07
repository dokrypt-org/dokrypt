package testrunner

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCoverageTracker_ReturnsNonNil(t *testing.T) {
	ct := NewCoverageTracker()
	require.NotNil(t, ct)
}

func TestNewCoverageTracker_StartsWithEmptyContracts(t *testing.T) {
	ct := NewCoverageTracker()
	report := ct.Report()
	require.NotNil(t, report)
	assert.Empty(t, report.Contracts)
}

func TestRegisterContract_AddsContractToTracker(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("ERC20", []string{"transfer", "approve", "balanceOf"})

	report := ct.Report()
	require.Contains(t, report.Contracts, "ERC20")
	assert.Equal(t, "ERC20", report.Contracts["ERC20"].Name)
}

func TestRegisterContract_MethodsDefaultToFalse(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("Token", []string{"mint", "burn"})

	cc := ct.Report().Contracts["Token"]
	require.Len(t, cc.Methods, 2)
	assert.False(t, cc.Methods["mint"])
	assert.False(t, cc.Methods["burn"])
}

func TestRegisterContract_EmptyMethods(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("Empty", []string{})

	cc := ct.Report().Contracts["Empty"]
	require.NotNil(t, cc)
	assert.Empty(t, cc.Methods)
}

func TestRegisterContract_NilMethods(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("NilMethods", nil)

	cc := ct.Report().Contracts["NilMethods"]
	require.NotNil(t, cc)
	assert.Empty(t, cc.Methods)
}

func TestRegisterContract_MultipleContracts(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("ERC20", []string{"transfer"})
	ct.RegisterContract("ERC721", []string{"safeTransferFrom"})

	report := ct.Report()
	assert.Len(t, report.Contracts, 2)
	assert.Contains(t, report.Contracts, "ERC20")
	assert.Contains(t, report.Contracts, "ERC721")
}

func TestRegisterContract_OverwritesPreviousRegistration(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("Token", []string{"mint", "burn"})
	ct.RegisterContract("Token", []string{"transfer"})

	cc := ct.Report().Contracts["Token"]
	require.Len(t, cc.Methods, 1)
	assert.Contains(t, cc.Methods, "transfer")
}

func TestRecordCall_MarksMethodAsCalled(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("ERC20", []string{"transfer", "approve"})
	ct.RecordCall("ERC20", "transfer")

	cc := ct.Report().Contracts["ERC20"]
	assert.True(t, cc.Methods["transfer"])
	assert.False(t, cc.Methods["approve"])
}

func TestRecordCall_MultipleMethods(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("Token", []string{"mint", "burn", "transfer"})
	ct.RecordCall("Token", "mint")
	ct.RecordCall("Token", "transfer")

	cc := ct.Report().Contracts["Token"]
	assert.True(t, cc.Methods["mint"])
	assert.False(t, cc.Methods["burn"])
	assert.True(t, cc.Methods["transfer"])
}

func TestRecordCall_DuplicateCallIdemptotent(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("C", []string{"m"})
	ct.RecordCall("C", "m")
	ct.RecordCall("C", "m")
	ct.RecordCall("C", "m")

	assert.True(t, ct.Report().Contracts["C"].Methods["m"])
}

func TestRecordCall_UnknownContract_DoesNotPanic(t *testing.T) {
	ct := NewCoverageTracker()
	assert.NotPanics(t, func() {
		ct.RecordCall("Unknown", "method")
	})
}

func TestRecordCall_UnregisteredMethod_SetsToTrue(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("ERC20", []string{"transfer"})
	ct.RecordCall("ERC20", "approve")

	cc := ct.Report().Contracts["ERC20"]
	assert.True(t, cc.Methods["approve"])
}

func TestReport_ReturnsNonNil(t *testing.T) {
	ct := NewCoverageTracker()
	report := ct.Report()
	require.NotNil(t, report)
	require.NotNil(t, report.Contracts)
}

func TestReport_SharesContractMap(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("Token", []string{"mint"})

	report := ct.Report()
	ct.RecordCall("Token", "mint")

	assert.True(t, report.Contracts["Token"].Methods["mint"])
}

func TestPrintCoverage_ContainsHeader(t *testing.T) {
	report := &CoverageReport{Contracts: map[string]*ContractCoverage{}}
	var buf bytes.Buffer
	PrintCoverage(report, &buf)

	output := buf.String()
	assert.Contains(t, output, "Coverage Report")
	assert.Contains(t, output, "Contract")
	assert.Contains(t, output, "Covered")
	assert.Contains(t, output, "Total")
	assert.Contains(t, output, "Percent")
}

func TestPrintCoverage_EmptyReport_NoContractLines(t *testing.T) {
	report := &CoverageReport{Contracts: map[string]*ContractCoverage{}}
	var buf bytes.Buffer
	PrintCoverage(report, &buf)

	lines := strings.Split(buf.String(), "\n")
	contractLines := 0
	for _, line := range lines {
		if strings.Contains(line, "%") {
			contractLines++
		}
	}
	assert.Zero(t, contractLines)
}

func TestPrintCoverage_SingleContract_ShowsCoverage(t *testing.T) {
	report := &CoverageReport{
		Contracts: map[string]*ContractCoverage{
			"ERC20": {
				Name: "ERC20",
				Methods: map[string]bool{
					"transfer": true,
					"approve":  false,
				},
			},
		},
	}
	var buf bytes.Buffer
	PrintCoverage(report, &buf)

	output := buf.String()
	assert.Contains(t, output, "ERC20")
	assert.Contains(t, output, "50.0%")
}

func TestPrintCoverage_AllMethodsCovered_100Percent(t *testing.T) {
	report := &CoverageReport{
		Contracts: map[string]*ContractCoverage{
			"Token": {
				Name: "Token",
				Methods: map[string]bool{
					"mint": true,
					"burn": true,
				},
			},
		},
	}
	var buf bytes.Buffer
	PrintCoverage(report, &buf)

	assert.Contains(t, buf.String(), "100.0%")
}

func TestPrintCoverage_NoMethodsCovered_0Percent(t *testing.T) {
	report := &CoverageReport{
		Contracts: map[string]*ContractCoverage{
			"Token": {
				Name: "Token",
				Methods: map[string]bool{
					"mint": false,
					"burn": false,
				},
			},
		},
	}
	var buf bytes.Buffer
	PrintCoverage(report, &buf)

	assert.Contains(t, buf.String(), "0.0%")
}

func TestPrintCoverage_ZeroMethods_ShowsZeroPercent(t *testing.T) {
	report := &CoverageReport{
		Contracts: map[string]*ContractCoverage{
			"Empty": {
				Name:    "Empty",
				Methods: map[string]bool{},
			},
		},
	}
	var buf bytes.Buffer
	PrintCoverage(report, &buf)

	assert.Contains(t, buf.String(), "0.0%")
}

func TestPrintCoverage_MultipleContracts_SortedAlphabetically(t *testing.T) {
	report := &CoverageReport{
		Contracts: map[string]*ContractCoverage{
			"Zebra": {Name: "Zebra", Methods: map[string]bool{"run": true}},
			"Alpha": {Name: "Alpha", Methods: map[string]bool{"start": false}},
			"Mango": {Name: "Mango", Methods: map[string]bool{"eat": true}},
		},
	}
	var buf bytes.Buffer
	PrintCoverage(report, &buf)

	output := buf.String()
	alphaIdx := strings.Index(output, "Alpha")
	mangoIdx := strings.Index(output, "Mango")
	zebraIdx := strings.Index(output, "Zebra")

	assert.Greater(t, mangoIdx, alphaIdx, "Alpha should appear before Mango")
	assert.Greater(t, zebraIdx, mangoIdx, "Mango should appear before Zebra")
}

func TestPrintCoverage_NilWriter_DoesNotPanic(t *testing.T) {
	report := &CoverageReport{Contracts: map[string]*ContractCoverage{}}
	assert.NotPanics(t, func() {
		PrintCoverage(report, nil)
	})
}

func TestCoverage_FullWorkflow(t *testing.T) {
	ct := NewCoverageTracker()
	ct.RegisterContract("ERC20", []string{"transfer", "approve", "balanceOf"})
	ct.RegisterContract("Vault", []string{"deposit", "withdraw"})

	ct.RecordCall("ERC20", "transfer")
	ct.RecordCall("ERC20", "balanceOf")
	ct.RecordCall("Vault", "deposit")

	report := ct.Report()
	require.Len(t, report.Contracts, 2)

	erc20 := report.Contracts["ERC20"]
	covered := 0
	for _, v := range erc20.Methods {
		if v {
			covered++
		}
	}
	assert.Equal(t, 2, covered)

	vault := report.Contracts["Vault"]
	covered = 0
	for _, v := range vault.Methods {
		if v {
			covered++
		}
	}
	assert.Equal(t, 1, covered)

	var buf bytes.Buffer
	PrintCoverage(report, &buf)
	output := buf.String()
	assert.Contains(t, output, "ERC20")
	assert.Contains(t, output, "Vault")
}
