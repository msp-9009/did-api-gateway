package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

var (
	ErrMaxAttemptsReached = errors.New("max retry attempts reached")
)

// Config holds retry configuration
type Config struct {
	MaxAttempts  int           // Maximum number of attempts
	InitialDelay time.Duration // Initial delay before first retry
	MaxDelay     time.Duration // Maximum delay between retries
	Multiplier   float64       // Backoff multiplier
	Jitter       bool          // Add randomness to prevent thundering herd
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// WithExponentialBackoff retries a function with exponential backoff
func WithExponentialBackoff(ctx context.Context, cfg Config, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Don't sleep before first attempt
		if attempt > 0 {
			delay := calculateBackoff(attempt-1, cfg)

			select {
			case <-time.After(delay):
				// Continue to retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return err
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return ErrMaxAttemptsReached
}

// WithExponentialBackoffContext is like WithExponentialBackoff but accepts context-aware function
func WithExponentialBackoffContext(ctx context.Context, cfg Config, fn func(context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Don't sleep before first attempt
		if attempt > 0 {
			delay := calculateBackoff(attempt-1, cfg)

			select {
			case <-time.After(delay):
				// Continue to retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Execute function with context
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return err
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return ErrMaxAttemptsReached
}

// calculateBackoff calculates the backoff delay for a given attempt
func calculateBackoff(attempt int, cfg Config) time.Duration {
	// Exponential backoff: delay = initial * (multiplier ^ attempt)
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	if cfg.Jitter {
		// Add random jitter (±25%)
		jitter := delay * 0.25 * (2*rand.Float64() - 1)
		delay += jitter
	}

	return time.Duration(delay)
}

// RetryableError is an error that can be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NonRetryableError is an error that should not be retried
type NonRetryableError struct {
	Err error
}

func (e *NonRetryableError) Error() string {
	return e.Err.Error()
}

func (e *NonRetryableError) Unwrap() error {
	return e.Err
}

// isRetryable checks if an error should be retried
func isRetryable(err error) bool {
	// Check for explicit non-retryable error
	var nonRetryable *NonRetryableError
	if errors.As(err, &nonRetryable) {
		return false
	}

	// Check for explicit retryable error
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}

	// Default: retry on timeout and temporary errors
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check for temporary interface (e.g., net.Error)
	type temporary interface {
		Temporary() bool
	}
	if te, ok := err.(temporary); ok {
		return te.Temporary()
	}

	// Default to retrying
	return true
}

// Retryable wraps an error as retryable
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// NonRetryable wraps an error as non-retryable
func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &NonRetryableError{Err: err}
}
