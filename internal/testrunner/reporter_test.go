package testrunner

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusIcon_Passed(t *testing.T) {
	assert.Equal(t, "PASS", statusIcon(StatusPassed))
}

func TestStatusIcon_Failed(t *testing.T) {
	assert.Equal(t, "FAIL", statusIcon(StatusFailed))
}

func TestStatusIcon_Skipped(t *testing.T) {
	assert.Equal(t, "SKIP", statusIcon(StatusSkipped))
}

func TestStatusIcon_Unknown(t *testing.T) {
	assert.Equal(t, "????", statusIcon(TestStatus("unknown")))
}

func TestStatusIcon_EmptyString(t *testing.T) {
	assert.Equal(t, "????", statusIcon(TestStatus("")))
}

func TestTableReporter_ImplementsReporterInterface(t *testing.T) {
	var r Reporter = &TableReporter{}
	assert.NotNil(t, r)
}

func TestTableReporter_EmptyResult(t *testing.T) {
	r := &TableReporter{}
	result := &Result{
		Suites:   nil,
		Total:    0,
		Passed:   0,
		Failed:   0,
		Skipped:  0,
		Duration: 100 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Results:")
	assert.Contains(t, output, "0 passed")
	assert.Contains(t, output, "0 failed")
	assert.Contains(t, output, "0 skipped")
	assert.Contains(t, output, "total: 0")
}

func TestTableReporter_SinglePassingSuite(t *testing.T) {
	r := &TableReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "ERC20 Tests",
				Duration: 50 * time.Millisecond,
				Tests: []TestResult{
					{Name: "transfer_success", Status: StatusPassed, Duration: 10 * time.Millisecond},
					{Name: "approve_success", Status: StatusPassed, Duration: 15 * time.Millisecond},
				},
			},
		},
		Total:    2,
		Passed:   2,
		Failed:   0,
		Skipped:  0,
		Duration: 50 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Suite: ERC20 Tests")
	assert.Contains(t, output, "PASS")
	assert.Contains(t, output, "transfer_success")
	assert.Contains(t, output, "approve_success")
	assert.Contains(t, output, "2 passed")
}

func TestTableReporter_FailedTestShowsError(t *testing.T) {
	r := &TableReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "suite",
				Duration: 20 * time.Millisecond,
				Tests: []TestResult{
					{
						Name:     "bad_test",
						Status:   StatusFailed,
						Duration: 5 * time.Millisecond,
						Error:    "assertion failed: expected 1, got 0",
					},
				},
			},
		},
		Total:    1,
		Passed:   0,
		Failed:   1,
		Skipped:  0,
		Duration: 20 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "FAIL")
	assert.Contains(t, output, "bad_test")
	assert.Contains(t, output, "Error: assertion failed: expected 1, got 0")
}

func TestTableReporter_SkippedTest(t *testing.T) {
	r := &TableReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "suite",
				Duration: 5 * time.Millisecond,
				Tests: []TestResult{
					{Name: "skipped_test", Status: StatusSkipped, Duration: 0},
				},
			},
		},
		Total:    1,
		Passed:   0,
		Failed:   0,
		Skipped:  1,
		Duration: 5 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "SKIP")
	assert.Contains(t, output, "skipped_test")
}

func TestTableReporter_MixedStatuses(t *testing.T) {
	r := &TableReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "mixed",
				Duration: 100 * time.Millisecond,
				Tests: []TestResult{
					{Name: "pass_1", Status: StatusPassed, Duration: 10 * time.Millisecond},
					{Name: "fail_1", Status: StatusFailed, Duration: 20 * time.Millisecond, Error: "err"},
					{Name: "skip_1", Status: StatusSkipped, Duration: 0},
				},
			},
		},
		Total:    3,
		Passed:   1,
		Failed:   1,
		Skipped:  1,
		Duration: 100 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "1 passed")
	assert.Contains(t, output, "1 failed")
	assert.Contains(t, output, "1 skipped")
	assert.Contains(t, output, "total: 3")
}

func TestTableReporter_MultipleSuites(t *testing.T) {
	r := &TableReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "Suite A",
				Duration: 10 * time.Millisecond,
				Tests: []TestResult{
					{Name: "a1", Status: StatusPassed, Duration: 5 * time.Millisecond},
				},
			},
			{
				Name:     "Suite B",
				Duration: 20 * time.Millisecond,
				Tests: []TestResult{
					{Name: "b1", Status: StatusFailed, Duration: 10 * time.Millisecond, Error: "boom"},
				},
			},
		},
		Total:    2,
		Passed:   1,
		Failed:   1,
		Skipped:  0,
		Duration: 30 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Suite: Suite A")
	assert.Contains(t, output, "Suite: Suite B")
}

func TestTableReporter_PassedTestHasNoErrorLine(t *testing.T) {
	r := &TableReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "suite",
				Duration: 10 * time.Millisecond,
				Tests: []TestResult{
					{Name: "good_test", Status: StatusPassed, Duration: 5 * time.Millisecond},
				},
			},
		},
		Total:    1,
		Passed:   1,
		Duration: 10 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	assert.NotContains(t, buf.String(), "Error:")
}

func TestTableReporter_SuiteSeparatorLine(t *testing.T) {
	r := &TableReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "suite",
				Duration: 5 * time.Millisecond,
				Tests:    nil,
			},
		},
		Duration: 5 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, strings.Repeat("-", 60))
}

func TestTableReporter_NilWriter_DoesNotPanic(t *testing.T) {
	r := &TableReporter{}
	result := &Result{Duration: time.Millisecond}
	assert.NotPanics(t, func() {
		_ = r.Report(result, nil)
	})
}

func TestTableReporter_ReturnsNilError(t *testing.T) {
	r := &TableReporter{}
	result := &Result{Duration: time.Millisecond}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	assert.NoError(t, err)
}

func TestJSONReporter_ImplementsReporterInterface(t *testing.T) {
	var r Reporter = &JSONReporter{}
	assert.NotNil(t, r)
}

func TestJSONReporter_EmptyResult_ProducesValidJSON(t *testing.T) {
	r := &JSONReporter{}
	result := &Result{
		Total:    0,
		Passed:   0,
		Failed:   0,
		Skipped:  0,
		Duration: 100 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err, "output should be valid JSON")
}

func TestJSONReporter_ContainsAllFields(t *testing.T) {
	r := &JSONReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "test-suite",
				Duration: 50 * time.Millisecond,
				Tests: []TestResult{
					{Name: "test1", Status: StatusPassed, Duration: 10 * time.Millisecond},
				},
			},
		},
		Total:    1,
		Passed:   1,
		Failed:   0,
		Skipped:  0,
		Duration: 50 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)

	assert.Contains(t, decoded, "suites")
	assert.Contains(t, decoded, "total")
	assert.Contains(t, decoded, "passed")
	assert.Contains(t, decoded, "failed")
	assert.Contains(t, decoded, "skipped")
	assert.Contains(t, decoded, "duration")
}

func TestJSONReporter_SuiteAndTestDetails(t *testing.T) {
	r := &JSONReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "my-suite",
				Duration: 30 * time.Millisecond,
				Tests: []TestResult{
					{Name: "passing-test", Status: StatusPassed, Duration: 10 * time.Millisecond},
					{Name: "failing-test", Status: StatusFailed, Duration: 20 * time.Millisecond, Error: "boom"},
				},
			},
		},
		Total:    2,
		Passed:   1,
		Failed:   1,
		Duration: 30 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "my-suite")
	assert.Contains(t, output, "passing-test")
	assert.Contains(t, output, "failing-test")
	assert.Contains(t, output, "boom")
	assert.Contains(t, output, "passed")
	assert.Contains(t, output, "failed")
}

func TestJSONReporter_OutputIsIndented(t *testing.T) {
	r := &JSONReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "suite",
				Duration: 10 * time.Millisecond,
				Tests: []TestResult{
					{Name: "t1", Status: StatusPassed, Duration: 5 * time.Millisecond},
				},
			},
		},
		Total:    1,
		Passed:   1,
		Duration: 10 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "\n")
	assert.Contains(t, output, "  ")
}

func TestJSONReporter_GasReportOmittedWhenNil(t *testing.T) {
	r := &JSONReporter{}
	result := &Result{
		Total:     0,
		GasReport: nil,
		Duration:  time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	assert.NotContains(t, buf.String(), "gas_report")
}

func TestJSONReporter_GasReportIncludedWhenPresent(t *testing.T) {
	r := &JSONReporter{}
	result := &Result{
		Total: 0,
		GasReport: &GasReport{
			Entries: []GasMethodSummary{
				{Contract: "C", Method: "m", MinGas: 100, AvgGas: 200, MaxGas: 300, Calls: 3},
			},
		},
		Duration: time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "gas_report")
}

func TestJSONReporter_NilWriter_DoesNotPanic(t *testing.T) {
	r := &JSONReporter{}
	result := &Result{Duration: time.Millisecond}
	assert.NotPanics(t, func() {
		_ = r.Report(result, nil)
	})
}

func TestJSONReporter_ErrorFieldOmittedForPassingTest(t *testing.T) {
	r := &JSONReporter{}
	result := &Result{
		Suites: []SuiteResult{
			{
				Name:     "suite",
				Duration: 5 * time.Millisecond,
				Tests: []TestResult{
					{Name: "pass", Status: StatusPassed, Duration: 2 * time.Millisecond},
				},
			},
		},
		Total:    1,
		Passed:   1,
		Duration: 5 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(result, &buf)
	require.NoError(t, err)

	var decoded Result
	err = json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded.Suites[0].Tests[0].Error)
}

func TestJSONReporter_RoundTrip_DeserializesCorrectly(t *testing.T) {
	r := &JSONReporter{}
	original := &Result{
		Suites: []SuiteResult{
			{
				Name:     "roundtrip-suite",
				Duration: 42 * time.Millisecond,
				Tests: []TestResult{
					{Name: "test-a", Status: StatusPassed, Duration: 10 * time.Millisecond},
					{Name: "test-b", Status: StatusFailed, Duration: 20 * time.Millisecond, Error: "err msg"},
					{Name: "test-c", Status: StatusSkipped, Duration: 0},
				},
			},
		},
		Total:    3,
		Passed:   1,
		Failed:   1,
		Skipped:  1,
		Duration: 42 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := r.Report(original, &buf)
	require.NoError(t, err)

	var decoded Result
	err = json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Total, decoded.Total)
	assert.Equal(t, original.Passed, decoded.Passed)
	assert.Equal(t, original.Failed, decoded.Failed)
	assert.Equal(t, original.Skipped, decoded.Skipped)
	require.Len(t, decoded.Suites, 1)
	assert.Equal(t, "roundtrip-suite", decoded.Suites[0].Name)
	require.Len(t, decoded.Suites[0].Tests, 3)
	assert.Equal(t, StatusPassed, decoded.Suites[0].Tests[0].Status)
	assert.Equal(t, StatusFailed, decoded.Suites[0].Tests[1].Status)
	assert.Equal(t, "err msg", decoded.Suites[0].Tests[1].Error)
	assert.Equal(t, StatusSkipped, decoded.Suites[0].Tests[2].Status)
}
