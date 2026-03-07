package common

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func zeroDelayConfig(maxAttempts int) RetryConfig {
	return RetryConfig{
		MaxAttempts:  maxAttempts,
		InitialDelay: 0,
		MaxDelay:     0,
		Multiplier:   1.0,
		Jitter:       0,
	}
}

func TestRetry_SucceedsOnFirstTry(t *testing.T) {
	var calls int32

	err := Retry(context.Background(), zeroDelayConfig(3), func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	if err != nil {
		t.Fatalf("Retry() error: %v", err)
	}
	if calls != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
}

func TestRetry_RetriesOnFailureThenSucceeds(t *testing.T) {
	var calls int32
	sentinelErr := errors.New("temporary failure")

	err := Retry(context.Background(), zeroDelayConfig(5), func(_ context.Context) error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return sentinelErr
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Retry() error: %v", err)
	}
	if calls != 3 {
		t.Errorf("fn called %d times, want 3", calls)
	}
}

func TestRetry_FailsAfterMaxAttempts(t *testing.T) {
	var calls int32
	sentinelErr := errors.New("always fails")

	err := Retry(context.Background(), zeroDelayConfig(3), func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		return sentinelErr
	})

	if !errors.Is(err, sentinelErr) {
		t.Errorf("Retry() error = %v, want %v", err, sentinelErr)
	}
	if calls != 3 {
		t.Errorf("fn called %d times, want 3 (MaxAttempts)", calls)
	}
}

func TestRetry_MaxAttempts1_NeverRetries(t *testing.T) {
	var calls int32
	someErr := errors.New("fail")

	err := Retry(context.Background(), zeroDelayConfig(1), func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		return someErr
	})

	if err == nil {
		t.Error("Retry() should return error")
	}
	if calls != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
}

func TestRetry_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   1.0,
		Jitter:       0,
	}

	var calls int32
	cancel() // Cancel immediately before calling Retry.

	err := Retry(ctx, cfg, func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("always fails")
	})

	if err == nil {
		t.Error("Retry() should return error when context is cancelled")
	}
	if calls > 10 {
		t.Errorf("fn called %d times, should be well under MaxAttempts", calls)
	}
}

func TestRetry_CancelledMidway(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   1.0,
		Jitter:       0,
	}

	var calls int32

	err := Retry(ctx, cfg, func(_ context.Context) error {
		n := atomic.AddInt32(&calls, 1)
		if n == 2 {
			cancel() // Cancel after second call so the sleep select triggers.
		}
		return errors.New("fail")
	})

	if err == nil {
		t.Error("Retry() should return error")
	}
	if calls > 5 {
		t.Errorf("fn called %d times, should have stopped early due to cancellation", calls)
	}
}

func TestRetry_ReturnsLastError(t *testing.T) {
	var attempt int32
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	err := Retry(context.Background(), zeroDelayConfig(2), func(_ context.Context) error {
		n := atomic.AddInt32(&attempt, 1)
		if n == 1 {
			return err1
		}
		return err2
	})

	if !errors.Is(err, err2) {
		t.Errorf("Retry() returned %v, want %v (last error)", err, err2)
	}
}

func TestRetry_PassesContextToFn(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "marker")

	err := Retry(ctx, zeroDelayConfig(1), func(fnCtx context.Context) error {
		val, _ := fnCtx.Value(key{}).(string)
		if val != "marker" {
			t.Errorf("context value = %q, want %q", val, "marker")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Retry() error: %v", err)
	}
}

func TestRetryWithResult_SucceedsOnFirstTry(t *testing.T) {
	result, err := RetryWithResult(context.Background(), zeroDelayConfig(3), func(_ context.Context) (int, error) {
		return 42, nil
	})

	if err != nil {
		t.Fatalf("RetryWithResult() error: %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
}

func TestRetryWithResult_RetriesOnFailureThenSucceeds(t *testing.T) {
	var calls int32

	result, err := RetryWithResult(context.Background(), zeroDelayConfig(5), func(_ context.Context) (string, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return "", errors.New("not ready")
		}
		return "done", nil
	})

	if err != nil {
		t.Fatalf("RetryWithResult() error: %v", err)
	}
	if result != "done" {
		t.Errorf("result = %q, want %q", result, "done")
	}
	if calls != 3 {
		t.Errorf("fn called %d times, want 3", calls)
	}
}

func TestRetryWithResult_FailsAfterMaxAttempts(t *testing.T) {
	sentinelErr := errors.New("always fails")

	result, err := RetryWithResult(context.Background(), zeroDelayConfig(3), func(_ context.Context) (int, error) {
		return 0, sentinelErr
	})

	if !errors.Is(err, sentinelErr) {
		t.Errorf("RetryWithResult() error = %v, want %v", err, sentinelErr)
	}
	if result != 0 {
		t.Errorf("result = %d, want 0", result)
	}
}

func TestRetryWithResult_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   1.0,
		Jitter:       0,
	}

	_, err := RetryWithResult(ctx, cfg, func(_ context.Context) (string, error) {
		return "", errors.New("fail")
	})

	if err == nil {
		t.Error("RetryWithResult() should return error when context is cancelled")
	}
}

func TestDefaultRetryConfig_HasSaneValues(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != 500*time.Millisecond {
		t.Errorf("InitialDelay = %v, want 500ms", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", cfg.MaxDelay)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", cfg.Multiplier)
	}
	if cfg.Jitter != 0.1 {
		t.Errorf("Jitter = %f, want 0.1", cfg.Jitter)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       0, // No jitter for predictable timing.
	}

	var timestamps []time.Time

	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		timestamps = append(timestamps, time.Now())
		return errors.New("fail")
	})

	if err == nil {
		t.Fatal("Retry() should have returned error")
	}

	if len(timestamps) != 4 {
		t.Fatalf("expected 4 timestamps, got %d", len(timestamps))
	}

	for i := 1; i < len(timestamps); i++ {
		gap := timestamps[i].Sub(timestamps[i-1])
		if gap < 5*time.Millisecond {
			t.Errorf("gap between attempt %d and %d = %v, expected at least 5ms", i-1, i, gap)
		}
	}
}

func TestRetry_MaxDelayIsCapped(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond, // Cap at 100ms
		Multiplier:   10.0,                   // Would grow to 1s without cap
		Jitter:       0,
	}

	start := time.Now()

	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		return errors.New("fail")
	})

	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Retry() should have returned error")
	}

	if elapsed > 500*time.Millisecond {
		t.Errorf("elapsed = %v, max delay cap may not be working (expected < 500ms)", elapsed)
	}
}
