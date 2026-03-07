package testrunner

import (
	"fmt"
	"io"
	"os"
	"sort"
)

type CoverageReport struct {
	Contracts map[string]*ContractCoverage `json:"contracts"`
}

type ContractCoverage struct {
	Name    string            `json:"name"`
	Methods map[string]bool   `json:"methods"` // method -> called?
}

type CoverageTracker struct {
	contracts map[string]*ContractCoverage
}

func NewCoverageTracker() *CoverageTracker {
	return &CoverageTracker{contracts: make(map[string]*ContractCoverage)}
}

func (c *CoverageTracker) RegisterContract(name string, methods []string) {
	cc := &ContractCoverage{
		Name:    name,
		Methods: make(map[string]bool),
	}
	for _, m := range methods {
		cc.Methods[m] = false
	}
	c.contracts[name] = cc
}

func (c *CoverageTracker) RecordCall(contract, method string) {
	if cc, ok := c.contracts[contract]; ok {
		cc.Methods[method] = true
	}
}

func (c *CoverageTracker) Report() *CoverageReport {
	return &CoverageReport{Contracts: c.contracts}
}

func PrintCoverage(report *CoverageReport, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintf(w, "\n  Coverage Report\n")
	fmt.Fprintf(w, "  %-30s %10s %10s %10s\n", "Contract", "Covered", "Total", "Percent")
	fmt.Fprintf(w, "  %s\n", "------------------------------------------------------")

	names := make([]string, 0, len(report.Contracts))
	for name := range report.Contracts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cc := report.Contracts[name]
		total := len(cc.Methods)
		covered := 0
		for _, called := range cc.Methods {
			if called {
				covered++
			}
		}
		pct := 0.0
		if total > 0 {
			pct = float64(covered) / float64(total) * 100
		}
		fmt.Fprintf(w, "  %-30s %10d %10d %9.1f%%\n", name, covered, total, pct)
	}
}
