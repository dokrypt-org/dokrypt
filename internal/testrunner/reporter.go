package testrunner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type Reporter interface {
	Report(result *Result, w io.Writer) error
}

type TableReporter struct{}

func (r *TableReporter) Report(result *Result, w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}

	for _, suite := range result.Suites {
		fmt.Fprintf(w, "\n  Suite: %s (%s)\n", suite.Name, suite.Duration)
		fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 60))

		for _, test := range suite.Tests {
			icon := statusIcon(test.Status)
			fmt.Fprintf(w, "  %s %s (%s)\n", icon, test.Name, test.Duration)
			if test.Error != "" {
				fmt.Fprintf(w, "      Error: %s\n", test.Error)
			}
		}
	}

	fmt.Fprintf(w, "\n  Results: %d passed, %d failed, %d skipped (total: %d, %s)\n",
		result.Passed, result.Failed, result.Skipped, result.Total, result.Duration)

	return nil
}

func statusIcon(s TestStatus) string {
	switch s {
	case StatusPassed:
		return "PASS"
	case StatusFailed:
		return "FAIL"
	case StatusSkipped:
		return "SKIP"
	default:
		return "????"
	}
}

type JSONReporter struct{}

func (r *JSONReporter) Report(result *Result, w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
