package testrunner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Runner struct {
	suites   []*Suite
	config   Config
	reporter Reporter
	gasTracker *GasTracker
}

type Config struct {
	Filter    string        // Filter tests by name pattern
	Parallel  int           // Max parallel tests
	Timeout   time.Duration // Per-test timeout
	GasReport bool          // Generate gas report
	Coverage  bool          // Generate coverage report
	Snapshot  bool          // Snapshot isolation per test
}

type Result struct {
	Suites    []SuiteResult `json:"suites"`
	Total     int           `json:"total"`
	Passed    int           `json:"passed"`
	Failed    int           `json:"failed"`
	Skipped   int           `json:"skipped"`
	Duration  time.Duration `json:"duration"`
	GasReport *GasReport    `json:"gas_report,omitempty"`
}

type SuiteResult struct {
	Name     string       `json:"name"`
	Tests    []TestResult `json:"tests"`
	Duration time.Duration `json:"duration"`
}

type TestResult struct {
	Name     string        `json:"name"`
	Status   TestStatus    `json:"status"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
	GasUsed  []GasEntry    `json:"gas_used,omitempty"`
}

type TestStatus string

const (
	StatusPassed  TestStatus = "passed"
	StatusFailed  TestStatus = "failed"
	StatusSkipped TestStatus = "skipped"
)

func NewRunner(cfg Config) *Runner {
	return &Runner{
		config:     cfg,
		reporter:   &TableReporter{},
		gasTracker: NewGasTracker(),
	}
}

func (r *Runner) AddSuite(suite *Suite) {
	r.suites = append(r.suites, suite)
}

func (r *Runner) Run(ctx context.Context) (*Result, error) {
	start := time.Now()
	result := &Result{}

	slog.Info("running tests", "suites", len(r.suites), "parallel", r.config.Parallel)

	for _, suite := range r.suites {
		sr, err := r.runSuite(ctx, suite)
		if err != nil {
			return nil, fmt.Errorf("suite %q failed: %w", suite.Name, err)
		}
		result.Suites = append(result.Suites, *sr)
		for _, tr := range sr.Tests {
			result.Total++
			switch tr.Status {
			case StatusPassed:
				result.Passed++
			case StatusFailed:
				result.Failed++
			case StatusSkipped:
				result.Skipped++
			}
		}
	}

	result.Duration = time.Since(start)

	if r.config.GasReport {
		result.GasReport = r.gasTracker.Report()
	}

	return result, nil
}

func (r *Runner) runSuite(ctx context.Context, suite *Suite) (*SuiteResult, error) {
	start := time.Now()
	sr := &SuiteResult{Name: suite.Name}

	if suite.Setup != nil {
		if err := suite.Setup(ctx); err != nil {
			return nil, fmt.Errorf("suite setup failed: %w", err)
		}
	}

	defer func() {
		if suite.Teardown != nil {
			if err := suite.Teardown(ctx); err != nil {
				slog.Error("suite teardown failed", "suite", suite.Name, "error", err)
			}
		}
	}()

	type indexedTest struct {
		index int
		test  TestCase
	}
	var runnable []indexedTest
	results := make([]TestResult, len(suite.Tests))
	for i, test := range suite.Tests {
		if r.config.Filter != "" && !matchFilter(test.Name, r.config.Filter) {
			results[i] = TestResult{
				Name:   test.Name,
				Status: StatusSkipped,
			}
		} else {
			runnable = append(runnable, indexedTest{index: i, test: test})
		}
	}

	if r.config.Parallel > 1 && len(runnable) > 1 {
		sem := make(chan struct{}, r.config.Parallel)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, it := range runnable {
			wg.Add(1)
			go func(idx int, tc TestCase) {
				defer wg.Done()
				sem <- struct{}{}        // acquire semaphore slot
				defer func() { <-sem }() // release semaphore slot

				tr := r.runTest(ctx, tc)

				mu.Lock()
				results[idx] = tr
				mu.Unlock()
			}(it.index, it.test)
		}

		wg.Wait()
	} else {
		for _, it := range runnable {
			results[it.index] = r.runTest(ctx, it.test)
		}
	}

	sr.Tests = results
	sr.Duration = time.Since(start)
	return sr, nil
}

func (r *Runner) runTest(ctx context.Context, test TestCase) TestResult {
	start := time.Now()

	testCtx := ctx
	if r.config.Timeout > 0 {
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	err := test.Fn(testCtx)
	tr := TestResult{
		Name:     test.Name,
		Duration: time.Since(start),
	}

	if err != nil {
		tr.Status = StatusFailed
		tr.Error = err.Error()
	} else {
		tr.Status = StatusPassed
	}

	return tr
}

func matchFilter(name, filter string) bool {
	return len(filter) == 0 || contains(name, filter)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
