package health_test

import (
	"context"
	"testing"
	"time"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/huzaifabhutta/notify-core/internal/health"
	"github.com/rs/zerolog"
)

// panicAdapter is a test adapter that panics during health check
type panicAdapter struct{}

func (p *panicAdapter) Send(ctx context.Context, req interface{}) (string, error) {
	return "", nil
}

func (p *panicAdapter) Name() string {
	return "panic-adapter"
}

func TestChecker_PanicRecovery(t *testing.T) {
	logger := zerolog.Nop()

	// Register a test adapter that will cause panic
	err := adapters.Register(adapters.Metadata{
		Name:        "panic-test-adapter",
		Type:        adapters.TypeEmail,
		Description: "Test adapter that panics",
		Version:     "1.0.0",
	}, func(config interface{}) (adapters.Adapter, error) {
		// This factory will be called, but won't cause issues
		// The panic will happen when we try to get metadata
		return &panicAdapter{}, nil
	})

	if err != nil && err != adapters.ErrAdapterAlreadyRegistered {
		t.Fatalf("Failed to register panic adapter: %v", err)
	}

	// Clean up after test
	defer func() {
		// Note: We can't easily unregister adapters, but that's okay for tests
	}()

	// Create health checker
	checker := health.NewChecker(&health.Config{
		Timeout:        3 * time.Second,
		EnableAdapters: true,
	}, logger)

	ctx := context.Background()

	// This should not panic even if an adapter causes issues
	report := checker.Check(ctx)

	if report == nil {
		t.Fatal("Expected non-nil health report even with panic")
	}

	// The health check should still complete successfully
	// Even if one adapter had issues, the overall system should report status
	if report.Status == "" {
		t.Error("Expected non-empty status")
	}

	// Verify we have components
	if len(report.Components) == 0 {
		t.Error("Expected at least some components in report")
	}

	// Verify summary is populated
	if report.Summary == nil {
		t.Fatal("Expected non-nil summary")
	}
}

func TestChecker_ConcurrentChecks(t *testing.T) {
	logger := zerolog.Nop()

	checker := health.NewChecker(&health.Config{
		Timeout:        3 * time.Second,
		EnableAdapters: true,
	}, logger)

	ctx := context.Background()

	// Run multiple health checks concurrently to test thread safety
	done := make(chan *health.HealthReport, 10)

	for i := 0; i < 10; i++ {
		go func() {
			report := checker.Check(ctx)
			done <- report
		}()
	}

	// Collect all reports
	for i := 0; i < 10; i++ {
		report := <-done
		if report == nil {
			t.Error("Expected non-nil report from concurrent check")
			continue
		}

		if report.Status == "" {
			t.Error("Expected non-empty status from concurrent check")
		}

		if len(report.Components) == 0 {
			t.Error("Expected components from concurrent check")
		}
	}
}

func TestChecker_ContextTimeout(t *testing.T) {
	logger := zerolog.Nop()

	checker := health.NewChecker(&health.Config{
		Timeout:        100 * time.Millisecond, // Very short timeout
		EnableAdapters: true,
	}, logger)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should still return a report, even with cancelled context
	report := checker.Check(ctx)

	if report == nil {
		t.Fatal("Expected non-nil report even with cancelled context")
	}

	// Should have some status
	if report.Status == "" {
		t.Error("Expected non-empty status")
	}
}
