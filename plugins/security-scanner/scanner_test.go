package securityscanner

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s := New()
	require.NotNil(t, s)
	assert.Len(t, s.rules, 5) // SELFDESTRUCT, DELEGATECALL, TX_ORIGIN, REENTRANCY_RISK, UNCHECKED_CALL
	assert.Empty(t, s.results)
}

func TestScanOpcodes_FoundAtStart(t *testing.T) {
	bytecode := []byte{0xFF, 0x00, 0x00}
	assert.True(t, scanOpcodes(bytecode, 0xFF))
}

func TestScanOpcodes_FoundInMiddle(t *testing.T) {
	bytecode := []byte{0x00, 0x01, 0xFF, 0x00}
	assert.True(t, scanOpcodes(bytecode, 0xFF))
}

func TestScanOpcodes_NotFound(t *testing.T) {
	bytecode := []byte{0x00, 0x01, 0x02}
	assert.False(t, scanOpcodes(bytecode, 0xFF))
}

func TestScanOpcodes_EmptyBytecode(t *testing.T) {
	assert.False(t, scanOpcodes([]byte{}, 0xFF))
	assert.False(t, scanOpcodes(nil, 0xFF))
}

func TestScanOpcodes_SkipsPUSH1Operand(t *testing.T) {
	bytecode := []byte{0x60, 0xFF, 0x00}
	assert.False(t, scanOpcodes(bytecode, 0xFF), "0xFF inside PUSH1 data should not be detected")
}

func TestScanOpcodes_SkipsPUSH2Operand(t *testing.T) {
	bytecode := []byte{0x61, 0xFF, 0xFF, 0x00}
	assert.False(t, scanOpcodes(bytecode, 0xFF), "0xFF inside PUSH2 data should not be detected")
}

func TestScanOpcodes_SkipsPUSH32Operand(t *testing.T) {
	bytecode := make([]byte, 34)
	bytecode[0] = 0x7F
	for i := 1; i <= 32; i++ {
		bytecode[i] = 0xFF
	}
	bytecode[33] = 0x00 // STOP
	assert.False(t, scanOpcodes(bytecode, 0xFF), "0xFF inside PUSH32 data should not be detected")
}

func TestScanOpcodes_FindsTargetAfterPUSH(t *testing.T) {
	bytecode := []byte{0x60, 0x00, 0xFF}
	assert.True(t, scanOpcodes(bytecode, 0xFF))
}

func TestScanOpcodes_MultiplePUSHes(t *testing.T) {
	bytecode := []byte{0x60, 0xF4, 0x61, 0xF4, 0xF4, 0x00}
	assert.False(t, scanOpcodes(bytecode, 0xF4), "0xF4 only appears as PUSH data")
}

func TestScanOpcodes_FindsAfterMultiplePUSHes(t *testing.T) {
	bytecode := []byte{0x60, 0x00, 0x61, 0x00, 0x00, 0xF4}
	assert.True(t, scanOpcodes(bytecode, 0xF4))
}

func TestScanOpcodes_PUSHTruncatedAtEnd(t *testing.T) {
	bytecode := []byte{0x61, 0xFF}
	assert.False(t, scanOpcodes(bytecode, 0xFF), "truncated PUSH operand should be skipped")
}

func TestScan_SELFDESTRUCT_Detected(t *testing.T) {
	s := New()
	bytecode := []byte{0x00, 0x01, 0xFF}
	result := s.Scan(context.Background(), "Vuln", "0xABC", bytecode)

	findingTitles := findingTitleSet(result)
	assert.Contains(t, findingTitles, "SELFDESTRUCT usage detected")
}

func TestScan_SELFDESTRUCT_NotDetected_WhenInPUSH(t *testing.T) {
	s := New()
	bytecode := []byte{0x60, 0xFF, 0x00}
	result := s.Scan(context.Background(), "Safe", "0xDEF", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "SELFDESTRUCT usage detected")
}

func TestScan_DELEGATECALL_Detected(t *testing.T) {
	s := New()
	bytecode := []byte{0x00, 0xF4, 0x00}
	result := s.Scan(context.Background(), "Proxy", "0x123", bytecode)

	findingTitles := findingTitleSet(result)
	assert.Contains(t, findingTitles, "DELEGATECALL usage detected")
}

func TestScan_DELEGATECALL_NotDetected_WhenInPUSH(t *testing.T) {
	s := New()
	bytecode := []byte{0x60, 0xF4, 0x00}
	result := s.Scan(context.Background(), "Safe", "0x456", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "DELEGATECALL usage detected")
}

func TestScan_TX_ORIGIN_Detected(t *testing.T) {
	s := New()
	bytecode := []byte{0x32, 0x00}
	result := s.Scan(context.Background(), "Auth", "0x789", bytecode)

	findingTitles := findingTitleSet(result)
	assert.Contains(t, findingTitles, "tx.origin usage detected")
}

func TestScan_TX_ORIGIN_NotDetected_WhenInPUSH(t *testing.T) {
	s := New()
	bytecode := []byte{0x60, 0x32, 0x00}
	result := s.Scan(context.Background(), "Safe", "0xaaa", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "tx.origin usage detected")
}

func TestScan_Reentrancy_Detected(t *testing.T) {
	s := New()
	bytecode := []byte{0x00, 0xF1, 0x00, 0x55, 0x00}
	result := s.Scan(context.Background(), "Reentr", "0xbbb", bytecode)

	findingTitles := findingTitleSet(result)
	assert.Contains(t, findingTitles, "Potential reentrancy risk")
}

func TestScan_Reentrancy_CALL_WithoutSSTORE(t *testing.T) {
	s := New()
	bytecode := []byte{0xF1, 0x00, 0x00}
	result := s.Scan(context.Background(), "OK", "0xccc", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "Potential reentrancy risk")
}

func TestScan_Reentrancy_SSTORE_BeforeCALL(t *testing.T) {
	s := New()
	bytecode := []byte{0x55, 0xF1, 0x00}
	result := s.Scan(context.Background(), "CEI", "0xddd", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "Potential reentrancy risk")
}

func TestScan_Reentrancy_CALL_InPUSH_NoFalsePositive(t *testing.T) {
	s := New()
	bytecode := []byte{0x60, 0xF1, 0x55, 0x00}
	result := s.Scan(context.Background(), "NOFP", "0xeee", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "Potential reentrancy risk",
		"CALL in PUSH data should not trigger reentrancy")
}

func TestScan_Reentrancy_SSTORE_InPUSH_NoFalsePositive(t *testing.T) {
	s := New()
	bytecode := []byte{0xF1, 0x60, 0x55, 0x00}
	result := s.Scan(context.Background(), "NOFP2", "0xfff", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "Potential reentrancy risk",
		"SSTORE in PUSH data after CALL should not trigger reentrancy")
}

func TestScan_UncheckedCall_Detected(t *testing.T) {
	s := New()
	bytecode := []byte{0x00, 0xF1, 0x50, 0x00}
	result := s.Scan(context.Background(), "Unchecked", "0x111", bytecode)

	findingTitles := findingTitleSet(result)
	assert.Contains(t, findingTitles, "Unchecked low-level call")
}

func TestScan_UncheckedCall_NotDetected_WhenChecked(t *testing.T) {
	s := New()
	bytecode := []byte{0xF1, 0x15, 0x00}
	result := s.Scan(context.Background(), "Checked", "0x222", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "Unchecked low-level call")
}

func TestScan_UncheckedCall_CALL_InPUSH_NoFalsePositive(t *testing.T) {
	s := New()
	bytecode := []byte{0x60, 0xF1, 0x50, 0x00}
	result := s.Scan(context.Background(), "NOFP", "0x333", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "Unchecked low-level call",
		"CALL byte inside PUSH data should not trigger unchecked call")
}

func TestScan_UncheckedCall_CALL_AtEndOfBytecode(t *testing.T) {
	s := New()
	bytecode := []byte{0x00, 0xF1}
	result := s.Scan(context.Background(), "Edge", "0x444", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "Unchecked low-level call")
}

func TestScan_CleanBytecode(t *testing.T) {
	s := New()
	bytecode := []byte{0x60, 0x00, 0x60, 0x00, 0xF3}
	result := s.Scan(context.Background(), "Clean", "0x555", bytecode)

	assert.Empty(t, result.Findings)
	assert.Equal(t, "Clean", result.Contract)
	assert.Equal(t, "0x555", result.Address)
}

func TestScan_EmptyBytecode(t *testing.T) {
	s := New()
	result := s.Scan(context.Background(), "Empty", "0x000", []byte{})
	assert.Empty(t, result.Findings)
}

func TestScan_NilBytecode(t *testing.T) {
	s := New()
	result := s.Scan(context.Background(), "Nil", "0x000", nil)
	assert.Empty(t, result.Findings)
}

func TestScan_MultipleFindings(t *testing.T) {
	s := New()
	bytecode := []byte{
		0x32,       // ORIGIN
		0xF1, 0x50, // CALL + POP
		0x55, // SSTORE (after CALL -> reentrancy)
		0xF4, // DELEGATECALL
		0xFF, // SELFDESTRUCT
	}
	result := s.Scan(context.Background(), "AllVuln", "0x666", bytecode)

	titles := findingTitleSet(result)
	assert.Contains(t, titles, "tx.origin usage detected")
	assert.Contains(t, titles, "Unchecked low-level call")
	assert.Contains(t, titles, "Potential reentrancy risk")
	assert.Contains(t, titles, "DELEGATECALL usage detected")
	assert.Contains(t, titles, "SELFDESTRUCT usage detected")
	assert.Len(t, result.Findings, 5)
}

func TestScan_FindingContractName(t *testing.T) {
	s := New()
	bytecode := []byte{0xFF}
	result := s.Scan(context.Background(), "MyContract", "0x777", bytecode)

	require.NotEmpty(t, result.Findings)
	for _, f := range result.Findings {
		assert.Equal(t, "MyContract", f.Contract)
	}
}

func TestScan_FindingSeverityAndSuggestion(t *testing.T) {
	s := New()
	bytecode := []byte{0xFF} // SELFDESTRUCT
	result := s.Scan(context.Background(), "C", "0x888", bytecode)

	require.NotEmpty(t, result.Findings)
	f := result.Findings[0]
	assert.Equal(t, SeverityHigh, f.Severity)
	assert.NotEmpty(t, f.Description)
	assert.NotEmpty(t, f.Suggestion)
}

func TestResults_Empty(t *testing.T) {
	s := New()
	assert.Empty(t, s.Results())
}

func TestResults_AccumulatesScans(t *testing.T) {
	s := New()
	s.Scan(context.Background(), "A", "0x1", []byte{0xFF})
	s.Scan(context.Background(), "B", "0x2", []byte{0xF4})
	s.Scan(context.Background(), "C", "0x3", []byte{0x00})

	results := s.Results()
	assert.Len(t, results, 3)
	assert.Equal(t, "A", results[0].Contract)
	assert.Equal(t, "B", results[1].Contract)
	assert.Equal(t, "C", results[2].Contract)
}

func TestResults_ReturnsCopy(t *testing.T) {
	s := New()
	s.Scan(context.Background(), "A", "0x1", []byte{0xFF})

	r1 := s.Results()
	r2 := s.Results()
	r1[0].Contract = "MODIFIED"
	assert.Equal(t, "A", r2[0].Contract)
}

func TestReset(t *testing.T) {
	s := New()
	s.Scan(context.Background(), "A", "0x1", []byte{0xFF})
	require.NotEmpty(t, s.Results())

	s.Reset()
	assert.Empty(t, s.Results())
}

func TestReset_CanScanAfterReset(t *testing.T) {
	s := New()
	s.Scan(context.Background(), "A", "0x1", []byte{0xFF})
	s.Reset()
	s.Scan(context.Background(), "B", "0x2", []byte{0xF4})

	results := s.Results()
	require.Len(t, results, 1)
	assert.Equal(t, "B", results[0].Contract)
}

func TestConcurrentScan(t *testing.T) {
	s := New()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bytecode := []byte{0xFF, 0xF4, 0x32}
			s.Scan(context.Background(), "Contract", "0xaddr", bytecode)
		}(i)
	}
	wg.Wait()

	results := s.Results()
	assert.Len(t, results, 50)
}

func TestConcurrentScanAndReset(t *testing.T) {
	s := New()
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			s.Scan(context.Background(), "C", "0x1", []byte{0xFF})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			s.Reset()
		}
	}()

	wg.Wait()
}

func TestConcurrentScanAndResults(t *testing.T) {
	s := New()
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			s.Scan(context.Background(), "C", "0x1", []byte{0xFF})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = s.Results()
		}
	}()

	wg.Wait()
}

func TestDefaultRules_IDs(t *testing.T) {
	rules := defaultRules()
	ids := make(map[string]bool)
	for _, r := range rules {
		ids[r.ID] = true
	}
	assert.True(t, ids["SELFDESTRUCT"])
	assert.True(t, ids["DELEGATECALL"])
	assert.True(t, ids["TX_ORIGIN"])
	assert.True(t, ids["REENTRANCY_RISK"])
	assert.True(t, ids["UNCHECKED_CALL"])
}

func TestDefaultRules_Severities(t *testing.T) {
	rules := defaultRules()
	severities := make(map[string]Severity)
	for _, r := range rules {
		severities[r.ID] = r.Severity
	}
	assert.Equal(t, SeverityHigh, severities["SELFDESTRUCT"])
	assert.Equal(t, SeverityMedium, severities["DELEGATECALL"])
	assert.Equal(t, SeverityMedium, severities["TX_ORIGIN"])
	assert.Equal(t, SeverityHigh, severities["REENTRANCY_RISK"])
	assert.Equal(t, SeverityMedium, severities["UNCHECKED_CALL"])
}

func TestScan_PUSHTruncatedAtEnd_NoFalsePositive(t *testing.T) {
	s := New()
	bytecode := []byte{0x62, 0x00, 0xFF}
	result := s.Scan(context.Background(), "Trunc", "0x999", bytecode)

	findingTitles := findingTitleSet(result)
	assert.NotContains(t, findingTitles, "SELFDESTRUCT usage detected",
		"0xFF in truncated PUSH data should not trigger SELFDESTRUCT")
}

func findingTitleSet(result *ScanResult) map[string]bool {
	titles := make(map[string]bool)
	for _, f := range result.Findings {
		titles[f.Title] = true
	}
	return titles
}
