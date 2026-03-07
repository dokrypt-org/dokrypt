package testrunner

import (
	"fmt"
	"io"
	"os"
	"sort"
)

type GasEntry struct {
	Contract string `json:"contract"`
	Method   string `json:"method"`
	GasUsed  uint64 `json:"gas_used"`
}

type GasReport struct {
	Entries []GasMethodSummary `json:"entries"`
}

type GasMethodSummary struct {
	Contract string `json:"contract"`
	Method   string `json:"method"`
	MinGas   uint64 `json:"min_gas"`
	AvgGas   uint64 `json:"avg_gas"`
	MaxGas   uint64 `json:"max_gas"`
	Calls    int    `json:"calls"`
}

type GasTracker struct {
	entries map[string][]uint64 // "Contract.Method" -> gas values
}

func NewGasTracker() *GasTracker {
	return &GasTracker{entries: make(map[string][]uint64)}
}

func (g *GasTracker) Record(contract, method string, gasUsed uint64) {
	key := contract + "." + method
	g.entries[key] = append(g.entries[key], gasUsed)
}

func (g *GasTracker) Report() *GasReport {
	report := &GasReport{}

	keys := make([]string, 0, len(g.entries))
	for k := range g.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := g.entries[key]
		contract, method := splitKey(key)

		var minGas, maxGas, totalGas uint64
		minGas = values[0]
		for _, v := range values {
			totalGas += v
			if v < minGas {
				minGas = v
			}
			if v > maxGas {
				maxGas = v
			}
		}

		report.Entries = append(report.Entries, GasMethodSummary{
			Contract: contract,
			Method:   method,
			MinGas:   minGas,
			AvgGas:   totalGas / uint64(len(values)),
			MaxGas:   maxGas,
			Calls:    len(values),
		})
	}

	return report
}

func PrintReport(report *GasReport, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintf(w, "\n  Gas Report\n")
	fmt.Fprintf(w, "  %-30s %10s %10s %10s %8s\n", "Contract.Method", "Min Gas", "Avg Gas", "Max Gas", "Calls")
	fmt.Fprintf(w, "  %s\n", "---------------------------------------------------------------------")

	for _, entry := range report.Entries {
		fmt.Fprintf(w, "  %-30s %10d %10d %10d %8d\n",
			entry.Contract+"."+entry.Method,
			entry.MinGas, entry.AvgGas, entry.MaxGas, entry.Calls)
	}
}

func splitKey(key string) (string, string) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
