package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/rs/zerolog"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
	StatusUnknown   Status = "unknown"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Latency   time.Duration          `json:"latency_ms"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// HealthReport represents the overall system health
type HealthReport struct {
	Status     Status                      `json:"status"`
	Timestamp  time.Time                   `json:"timestamp"`
	Components map[string]ComponentHealth  `json:"components"`
	Summary    map[string]int              `json:"summary"` // Count of healthy/unhealthy/degraded
}

// Checker performs health checks on adapters and system components
type Checker struct {
	config *Config
	logger zerolog.Logger
}

// Config holds health checker configuration
type Config struct {
	Timeout        time.Duration // Timeout for each health check
	EnableAdapters bool          // Check adapter health
	EnableDatabase bool          // Check database health (future)
}

// NewChecker creates a new health checker
func NewChecker(cfg *Config, logger zerolog.Logger) *Checker {
	if cfg == nil {
		cfg = &Config{
			Timeout:        5 * time.Second,
			EnableAdapters: true,
			EnableDatabase: false,
		}
	}

	return &Checker{
		config: cfg,
		logger: logger.With().Str("component", "health-checker").Logger(),
	}
}

// Check performs a comprehensive health check
func (c *Checker) Check(ctx context.Context) *HealthReport {
	startTime := time.Now()

	report := &HealthReport{
		Status:     StatusHealthy,
		Timestamp:  startTime,
		Components: make(map[string]ComponentHealth),
		Summary: map[string]int{
			"healthy":   0,
			"unhealthy": 0,
			"degraded":  0,
			"unknown":   0,
		},
	}

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	var wg sync.WaitGroup

	// Check adapters if enabled
	if c.config.EnableAdapters {
		c.checkAdapters(checkCtx, report, &wg)
	}

	// Wait for all checks to complete
	wg.Wait()

	// Calculate summary and overall status
	c.calculateOverallStatus(report)

	c.logger.Info().
		Str("overall_status", string(report.Status)).
		Int("healthy", report.Summary["healthy"]).
		Int("unhealthy", report.Summary["unhealthy"]).
		Dur("duration_ms", time.Since(startTime)).
		Msg("Health check completed")

	return report
}

// checkAdapters checks the health of all registered adapters
func (c *Checker) checkAdapters(ctx context.Context, report *HealthReport, wg *sync.WaitGroup) {
	allAdapters := adapters.List()

	c.logger.Debug().
		Int("adapter_count", len(allAdapters)).
		Msg("Checking adapter health")

	var mu sync.Mutex

	for _, adapterName := range allAdapters {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			health := c.checkAdapter(ctx, name)

			mu.Lock()
			report.Components[fmt.Sprintf("adapter:%s", name)] = health
			report.Summary[string(health.Status)]++
			mu.Unlock()
		}(adapterName)
	}
}

// checkAdapter checks the health of a single adapter
func (c *Checker) checkAdapter(ctx context.Context, adapterName string) ComponentHealth {
	startTime := time.Now()

	health := ComponentHealth{
		Name:      adapterName,
		Type:      "adapter",
		Status:    StatusUnknown,
		Timestamp: startTime,
		Metadata:  make(map[string]interface{}),
	}

	// Get adapter metadata
	meta := adapters.GetMetadata(adapterName)
	if meta == nil {
		health.Status = StatusUnhealthy
		health.Message = "adapter not found in registry"
		health.Latency = time.Since(startTime)
		return health
	}

	health.Metadata["adapter_type"] = string(meta.Type)
	health.Metadata["version"] = meta.Version

	// For now, we just check if the adapter is registered
	// In the future, we could instantiate with test config and call Ping()
	// However, this would require knowing the configuration for each adapter

	// Since adapter is registered, mark as healthy
	health.Status = StatusHealthy
	health.Message = "adapter registered and available"
	health.Latency = time.Since(startTime)

	c.logger.Debug().
		Str("adapter", adapterName).
		Str("status", string(health.Status)).
		Dur("latency_ms", health.Latency).
		Msg("Adapter health check completed")

	return health
}

// calculateOverallStatus determines the overall system status
func (c *Checker) calculateOverallStatus(report *HealthReport) {
	if report.Summary["unhealthy"] > 0 {
		report.Status = StatusUnhealthy
	} else if report.Summary["degraded"] > 0 {
		report.Status = StatusDegraded
	} else if report.Summary["healthy"] > 0 {
		report.Status = StatusHealthy
	} else {
		report.Status = StatusUnknown
	}
}

// CheckAdapter performs a health check on a specific adapter with actual configuration
// This is useful for checking if an adapter can actually connect to its service
func (c *Checker) CheckAdapter(ctx context.Context, adapter adapters.Adapter) ComponentHealth {
	startTime := time.Now()

	health := ComponentHealth{
		Name:      adapter.Name(),
		Type:      "adapter",
		Status:    StatusUnknown,
		Timestamp: startTime,
		Metadata:  make(map[string]interface{}),
	}

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Try to ping the adapter
	// Note: Not all adapters implement Ping(), so we use type assertion
	type pinger interface {
		Ping(context.Context) error
	}

	if p, ok := adapter.(pinger); ok {
		err := p.Ping(checkCtx)
		if err != nil {
			health.Status = StatusUnhealthy
			health.Message = fmt.Sprintf("ping failed: %v", err)
		} else {
			health.Status = StatusHealthy
			health.Message = "ping successful"
		}
	} else {
		// Adapter doesn't support Ping(), assume healthy
		health.Status = StatusHealthy
		health.Message = "adapter available (ping not supported)"
	}

	health.Latency = time.Since(startTime)

	return health
}
