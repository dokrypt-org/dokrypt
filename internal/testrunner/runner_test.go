package testrunner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner_CreatesWithDefaultReporter(t *testing.T) {
	cfg := Config{Parallel: 4, Timeout: 30 * time.Second}
	r := NewRunner(cfg)

	require.NotNil(t, r)
	assert.NotNil(t, r.reporter)
	assert.NotNil(t, r.gasTracker)
}

func TestNewRunner_StoresConfig(t *testing.T) {
	cfg := Config{
		Filter:    "Transfer",
		Parallel:  8,
		Timeout:   10 * time.Second,
		GasReport: true,
		Coverage:  true,
		Snapshot:  true,
	}
	r := NewRunner(cfg)

	assert.Equal(t, cfg.Filter, r.config.Filter)
	assert.Equal(t, cfg.Parallel, r.config.Parallel)
	assert.Equal(t, cfg.Timeout, r.config.Timeout)
	assert.True(t, r.config.GasReport)
	assert.True(t, r.config.Coverage)
	assert.True(t, r.config.Snapshot)
}

func TestNewRunner_StartsWithNoSuites(t *testing.T) {
	r := NewRunner(Config{})

	assert.Empty(t, r.suites)
}

func TestAddSuite_AppendsSuite(t *testing.T) {
	r := NewRunner(Config{})
	s1 := NewSuite("suite-one")
	s2 := NewSuite("suite-two")

	r.AddSuite(s1)
	r.AddSuite(s2)

	require.Len(t, r.suites, 2)
	assert.Equal(t, "suite-one", r.suites[0].Name)
	assert.Equal(t, "suite-two", r.suites[1].Name)
}

func TestRun_PassingTests_AllPass(t *testing.T) {
	r := NewRunner(Config{})

	suite := NewSuite("happy-path")
	suite.AddTest("test-one", func(_ context.Context) error { return nil })
	suite.AddTest("test-two", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, result.Passed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 0, result.Skipped)
}

func TestRun_PassingTests_SuiteResultHasCorrectName(t *testing.T) {
	r := NewRunner(Config{})
	suite := NewSuite("my-suite")
	suite.AddTest("t1", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Suites, 1)
	assert.Equal(t, "my-suite", result.Suites[0].Name)
}

func TestRun_PassingTests_TestResultStatusIsPassed(t *testing.T) {
	r := NewRunner(Config{})
	suite := NewSuite("s")
	suite.AddTest("passing", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)

	tr := result.Suites[0].Tests[0]
	assert.Equal(t, StatusPassed, tr.Status)
	assert.Empty(t, tr.Error)
}

func TestRun_DurationIsPositive(t *testing.T) {
	r := NewRunner(Config{})
	suite := NewSuite("s")
	suite.AddTest("t", func(_ context.Context) error {
		time.Sleep(time.Millisecond)
		return nil
	})
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	assert.Positive(t, result.Duration)
}

func TestRun_FailingTests_CountedCorrectly(t *testing.T) {
	r := NewRunner(Config{})

	suite := NewSuite("mixed")
	suite.AddTest("passes", func(_ context.Context) error { return nil })
	suite.AddTest("fails", func(_ context.Context) error { return errors.New("assertion failed") })
	suite.AddTest("also-passes", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 3, result.Total)
	assert.Equal(t, 2, result.Passed)
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, 0, result.Skipped)
}

func TestRun_FailingTest_StatusIsFailed(t *testing.T) {
	r := NewRunner(Config{})
	suite := NewSuite("s")
	suite.AddTest("bad", func(_ context.Context) error { return errors.New("boom") })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)

	tr := result.Suites[0].Tests[0]
	assert.Equal(t, StatusFailed, tr.Status)
	assert.Equal(t, "boom", tr.Error)
}

func TestRun_MultipleFailingTests_AllCounted(t *testing.T) {
	r := NewRunner(Config{})
	suite := NewSuite("all-fail")
	for i := 0; i < 5; i++ {
		suite.AddTest("t", func(_ context.Context) error { return errors.New("err") })
	}
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 5, result.Failed)
}

func TestRun_EmptyRunner_ZeroCounts(t *testing.T) {
	r := NewRunner(Config{})

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Zero(t, result.Total)
	assert.Zero(t, result.Passed)
	assert.Zero(t, result.Failed)
}

func TestRun_EmptySuite_ZeroCounts(t *testing.T) {
	r := NewRunner(Config{})
	r.AddSuite(NewSuite("empty"))

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	assert.Zero(t, result.Total)
}

func TestRun_Filter_MatchingTestsExecuted(t *testing.T) {
	r := NewRunner(Config{Filter: "Transfer"})

	suite := NewSuite("s")
	suite.AddTest("Transfer_Success", func(_ context.Context) error { return nil })
	suite.AddTest("Transfer_InsufficientFunds", func(_ context.Context) error { return nil })
	suite.AddTest("Approval_Basic", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, result.Passed)
	assert.Equal(t, 1, result.Skipped)
}

func TestRun_Filter_NoMatchSkipsAll(t *testing.T) {
	r := NewRunner(Config{Filter: "XYZ_NOMATCH"})

	suite := NewSuite("s")
	suite.AddTest("Transfer", func(_ context.Context) error { return nil })
	suite.AddTest("Approval", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 0, result.Passed)
	assert.Equal(t, 2, result.Skipped)
}

func TestRun_Filter_EmptyFilterRunsAll(t *testing.T) {
	r := NewRunner(Config{Filter: ""})

	suite := NewSuite("s")
	suite.AddTest("A", func(_ context.Context) error { return nil })
	suite.AddTest("B", func(_ context.Context) error { return nil })
	suite.AddTest("C", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, result.Passed)
	assert.Equal(t, 0, result.Skipped)
}

func TestRun_Filter_SkippedTestsHaveSkippedStatus(t *testing.T) {
	r := NewRunner(Config{Filter: "Transfer"})

	suite := NewSuite("s")
	suite.AddTest("Approval_Basic", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)

	require.Len(t, result.Suites[0].Tests, 1)
	assert.Equal(t, StatusSkipped, result.Suites[0].Tests[0].Status)
}

func TestRun_MultipleSuites_AggregatesTotals(t *testing.T) {
	r := NewRunner(Config{})

	s1 := NewSuite("suite-one")
	s1.AddTest("pass", func(_ context.Context) error { return nil })
	s1.AddTest("fail", func(_ context.Context) error { return errors.New("err") })

	s2 := NewSuite("suite-two")
	s2.AddTest("pass1", func(_ context.Context) error { return nil })
	s2.AddTest("pass2", func(_ context.Context) error { return nil })

	r.AddSuite(s1)
	r.AddSuite(s2)

	result, err := r.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 4, result.Total)
	assert.Equal(t, 3, result.Passed)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Suites, 2)
}

func TestRun_MultipleSuites_EachSuiteResultHasCorrectTests(t *testing.T) {
	r := NewRunner(Config{})

	s1 := NewSuite("alpha")
	s1.AddTest("a1", func(_ context.Context) error { return nil })

	s2 := NewSuite("beta")
	s2.AddTest("b1", func(_ context.Context) error { return errors.New("fail") })
	s2.AddTest("b2", func(_ context.Context) error { return nil })

	r.AddSuite(s1)
	r.AddSuite(s2)

	result, err := r.Run(context.Background())
	require.NoError(t, err)

	require.Len(t, result.Suites[0].Tests, 1)
	require.Len(t, result.Suites[1].Tests, 2)

	assert.Equal(t, StatusPassed, result.Suites[0].Tests[0].Status)
	assert.Equal(t, StatusFailed, result.Suites[1].Tests[0].Status)
	assert.Equal(t, StatusPassed, result.Suites[1].Tests[1].Status)
}

func TestRun_GasReportIncludedWhenEnabled(t *testing.T) {
	r := NewRunner(Config{GasReport: true})
	suite := NewSuite("s")
	suite.AddTest("t", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result.GasReport)
}

func TestRun_GasReportNilWhenDisabled(t *testing.T) {
	r := NewRunner(Config{GasReport: false})
	suite := NewSuite("s")
	suite.AddTest("t", func(_ context.Context) error { return nil })
	r.AddSuite(suite)

	result, err := r.Run(context.Background())
	require.NoError(t, err)
	assert.Nil(t, result.GasReport)
}

func TestMatchFilter_EmptyFilterAlwaysMatches(t *testing.T) {
	assert.True(t, matchFilter("AnyTestName", ""))
}

func TestMatchFilter_SubstringMatch(t *testing.T) {
	assert.True(t, matchFilter("Transfer_Success", "Transfer"))
	assert.True(t, matchFilter("Test_Transfer_Fail", "Transfer"))
}

func TestMatchFilter_NoMatch(t *testing.T) {
	assert.False(t, matchFilter("Approval_Basic", "Transfer"))
}

func TestMatchFilter_ExactMatch(t *testing.T) {
	assert.True(t, matchFilter("Exact", "Exact"))
}

func TestMatchFilter_FilterLongerThanName(t *testing.T) {
	assert.False(t, matchFilter("Hi", "LongerFilterString"))
}
