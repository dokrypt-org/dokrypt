package common

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64 // 0.0 to 1.0, fraction of delay to randomize
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

func Retry(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := range cfg.MaxAttempts {
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		if attempt == cfg.MaxAttempts-1 {
			break
		}

		jitterRange := float64(delay) * cfg.Jitter
		jitter := time.Duration(rand.Float64()*2*jitterRange - jitterRange)
		sleepDuration := delay + jitter

		if sleepDuration < 0 {
			sleepDuration = 0
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDuration):
		}

		delay = time.Duration(math.Min(
			float64(delay)*cfg.Multiplier,
			float64(cfg.MaxDelay),
		))
	}

	return lastErr
}

func RetryWithResult[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	var lastResult T
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := range cfg.MaxAttempts {
		lastResult, lastErr = fn(ctx)
		if lastErr == nil {
			return lastResult, nil
		}

		if attempt == cfg.MaxAttempts-1 {
			break
		}

		jitterRange := float64(delay) * cfg.Jitter
		jitter := time.Duration(rand.Float64()*2*jitterRange - jitterRange)
		sleepDuration := delay + jitter

		if sleepDuration < 0 {
			sleepDuration = 0
		}

		select {
		case <-ctx.Done():
			return lastResult, ctx.Err()
		case <-time.After(sleepDuration):
		}

		delay = time.Duration(math.Min(
			float64(delay)*cfg.Multiplier,
			float64(cfg.MaxDelay),
		))
	}

	return lastResult, lastErr
}
