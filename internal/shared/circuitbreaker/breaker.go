package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
	ErrTimeout     = errors.New("operation timed out")
)

// CircuitBreaker prevents cascading failures by failing fast when a service is down
type CircuitBreaker struct {
	maxFailures  int
	timeout      time.Duration
	resetTimeout time.Duration

	mu           sync.RWMutex
	state        State
	failures     int
	successes    int
	lastFailTime time.Time
	lastStateChange time.Time
	
	// Metrics
	totalCalls   int64
	totalSuccess int64
	totalFailure int64
}

// Config holds circuit breaker configuration
type Config struct {
	MaxFailures  int           // Number of failures before opening
	Timeout      time.Duration // Max duration for a single call
	ResetTimeout time.Duration // Time to wait before trying again
}

// New creates a new circuit breaker
func New(cfg Config) *CircuitBreaker {
	if cfg.MaxFailures == 0 {
		cfg.MaxFailures = 5
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.ResetTimeout == 0 {
		cfg.ResetTimeout = 60 * time.Second
	}

	return &CircuitBreaker{
		maxFailures:  cfg.MaxFailures,
		timeout:      cfg.Timeout,
		resetTimeout: cfg.ResetTimeout,
		state:        StateClosed,
		lastStateChange: time.Now(),
	}
}

// Call executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	if !cb.canAttempt() {
		return ErrCircuitOpen
	}

	// Create timeout context
	callCtx, cancel := context.WithTimeout(ctx, cb.timeout)
	defer cancel()

	// Execute with timeout
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn(callCtx)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			cb.recordFailure()
			return err
		}
		cb.recordSuccess()
		return nil
	case <-callCtx.Done():
		cb.recordFailure()
		return ErrTimeout
	}
}

// canAttempt checks if a request can be attempted
func (cb *CircuitBreaker) canAttempt() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalCalls++

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.lastStateChange = time.Now()
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		return true
	}

	return false
}

// recordFailure records a failed call
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalFailure++
	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.state == StateHalfOpen {
		// If fails in half-open, go back to open
		cb.state = StateOpen
		cb.failures = 0
		cb.lastStateChange = time.Now()
	} else if cb.failures >= cb.maxFailures {
		// Open the circuit
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
	}
}

// recordSuccess records a successful call
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalSuccess++

	if cb.state == StateHalfOpen {
		cb.successes++
		// After a few successes in half-open, close the circuit
		if cb.successes >= 3 {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
			cb.lastStateChange = time.Now()
		}
	} else {
		cb.failures = 0
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:        cb.state,
		Failures:     cb.failures,
		TotalCalls:   cb.totalCalls,
		TotalSuccess: cb.totalSuccess,
		TotalFailure: cb.totalFailure,
		LastFailTime: cb.lastFailTime,
	}
}

// Stats holds circuit breaker statistics
type Stats struct {
	State        State
	Failures     int
	TotalCalls   int64
	TotalSuccess int64
	TotalFailure int64
	LastFailTime time.Time
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastStateChange = time.Now()
}
