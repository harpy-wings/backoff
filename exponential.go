package backoff

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"
)

func ExponentialDo(ctx context.Context, fn func(ctx context.Context) error, opts ...Option) error {
	cfg := &config{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var lastErr error

	for i := 0; i < cfg.MaxAttempts; i++ {
		// First guard: check if the context is done before executing the function
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := fn(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if i == cfg.MaxAttempts-1 {
			break // Last attempt, no need to sleep
		}

		// 1. Calculate the actual delay: InitialDelay * 2^i
		delay := cfg.InitialDelay * time.Duration(1<<uint(i))

		// 2. Apply the MaxDelay limit
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		// 3. Inject Jitter (maximum 10% of the delay time randomly)
		delay = applyJitter(delay)

		// 4. Context-Aware Sleep
		select {
		case <-ctx.Done():
			return errors.Join(ctx.Err(), lastErr) // Return both the context error and the last function error
		case <-time.After(delay):
		}
	}

	return errors.Join(lastErr, fmt.Errorf("failed after %d attempts", cfg.MaxAttempts))
}

func ExponentialDoWithReturn[T any](ctx context.Context, fn func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	cfg := &config{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var lastErr error
	var zero T

	for i := 0; i < cfg.MaxAttempts; i++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if i == cfg.MaxAttempts-1 {
			break
		}

		delay := cfg.InitialDelay * time.Duration(1<<uint(i))
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
		delay = applyJitter(delay)

		select {
		case <-ctx.Done():
			return zero, errors.Join(ctx.Err(), lastErr)
		case <-time.After(delay):
		}
	}

	return zero, fmt.Errorf("failed after %d attempts. last error: %w", cfg.MaxAttempts, lastErr)
}

// Helper function to create random jitter (Jitter) in the delay time to prevent concurrent worker attacks
func applyJitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return delay
	}
	maxJitter := delay / 10 // 10% of the delay
	if maxJitter <= 0 {
		return delay
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxJitter)))
	if err != nil {
		return delay // If the randomizer fails, return the original delay
	}
	return delay + time.Duration(n.Int64())
}
