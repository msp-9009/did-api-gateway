package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Status represents health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Component represents a health check component
type Component struct {
	Name    string        `json:"name"`
	Status  Status        `json:"status"`
	Error   string        `json:"error,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
}

// HealthStatus represents overall health status
type HealthStatus struct {
	Status     Status       `json:"status"`
	Components []*Component `json:"components"`
	Timestamp  time.Time    `json:"timestamp"`
}

// Checker defines health check interface
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// HealthChecker aggregates multiple health checks
type HealthChecker struct {
	checkers []Checker
	mu       sync.RWMutex
}

// New creates a new health checker
func New() *HealthChecker {
	return &HealthChecker{
		checkers: make([]Checker, 0),
	}
}

// Register adds a health checker
func (h *HealthChecker) Register(checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers = append(h.checkers, checker)
}

// Check runs all health checks
func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
	h.mu.RLock()
	checkers := h.checkers
	h.mu.RUnlock()

	components := make([]*Component, len(checkers))
	var wg sync.WaitGroup

	// Run checks in parallel
	for i, checker := range checkers {
		wg.Add(1)
		go func(idx int, chk Checker) {
			defer wg.Done()

			start := time.Now()
			err := chk.Check(ctx)
			latency := time.Since(start)

			component := &Component{
				Name:    chk.Name(),
				Status:  statusFromError(err),
				Latency: latency,
			}
			if err != nil {
				component.Error = err.Error()
			}

			components[idx] = component
		}(i, checker)
	}

	wg.Wait()

	// Calculate overall status
	overallStatus := calculateOverallStatus(components)

	return &HealthStatus{
		Status:     overallStatus,
		Components: components,
		Timestamp:  time.Now(),
	}
}

// Handler returns an HTTP handler for health checks
func (h *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		status := h.Check(ctx)

		w.Header().Set("Content-Type", "application/json")
		
		// Set HTTP status code based on health
		switch status.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK) // Still accept traffic
		case StatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(status)
	}
}

// statusFromError converts an error to a health status
func statusFromError(err error) Status {
	if err == nil {
		return StatusHealthy
	}
	return StatusUnhealthy
}

// calculateOverallStatus determines overall health from components
func calculateOverallStatus(components []*Component) Status {
	unhealthy := 0
	degraded := 0

	for _, c := range components {
		switch c.Status {
		case StatusUnhealthy:
			unhealthy++
		case StatusDegraded:
			degraded++
		}
	}

	// If any critical component is unhealthy, system is unhealthy
	if unhealthy > 0 {
		return StatusUnhealthy
	}

	// If any component is degraded, system is degraded
	if degraded > 0 {
		return StatusDegraded
	}

	return StatusHealthy
}

// DatabaseChecker checks database health
type DatabaseChecker struct {
	name string
	ping func(context.Context) error
}

// NewDatabaseChecker creates a database health checker
func NewDatabaseChecker(name string, ping func(context.Context) error) *DatabaseChecker {
	return &DatabaseChecker{
		name: name,
		ping: ping,
	}
}

// Name returns the checker name
func (d *DatabaseChecker) Name() string {
	return d.name
}

// Check performs the health check
func (d *DatabaseChecker) Check(ctx context.Context) error {
	return d.ping(ctx)
}

// RedisChecker checks Redis health
type RedisChecker struct {
	name string
	ping func(context.Context) error
}

// NewRedisChecker creates a Redis health checker
func NewRedisChecker(name string, ping func(context.Context) error) *RedisChecker {
	return &RedisChecker{
		name: name,
		ping: ping,
	}
}

// Name returns the checker name
func (r *RedisChecker) Name() string {
	return r.name
}

// Check performs the health check
func (r *RedisChecker) Check(ctx context.Context) error {
	return r.ping(ctx)
}

// ReadinessHandler returns a simple readiness check
func ReadinessHandler(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		status := checker.Check(ctx)

		// Readiness: only healthy instances should receive traffic
		if status.Status == StatusHealthy {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "ready")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "not ready")
		}
	}
}

// LivenessHandler returns a simple liveness check
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness: just check if the process is alive
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "alive")
	}
}
