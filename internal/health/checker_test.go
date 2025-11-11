package health_test

import (
	"context"
	"testing"
	"time"

	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/smtp"
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/ses"
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/sms/sns"
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/whatsapp/cloud"
	"github.com/huzaifabhutta/notify-core/internal/health"
	"github.com/rs/zerolog"
)

func TestChecker_Check(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name              string
		config            *health.Config
		wantMinComponents int
		wantStatus        health.Status
	}{
		{
			name: "default configuration with adapters",
			config: &health.Config{
				Timeout:        5 * time.Second,
				EnableAdapters: true,
			},
			wantMinComponents: 4, // smtp, ses, sns, whatsapp-cloud
			wantStatus:        health.StatusHealthy,
		},
		{
			name: "adapters disabled",
			config: &health.Config{
				Timeout:        5 * time.Second,
				EnableAdapters: false,
			},
			wantMinComponents: 0,
			wantStatus:        health.StatusUnknown,
		},
		{
			name: "short timeout",
			config: &health.Config{
				Timeout:        100 * time.Millisecond,
				EnableAdapters: true,
			},
			wantMinComponents: 4,
			wantStatus:        health.StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := health.NewChecker(tt.config, logger)
			ctx := context.Background()

			report := checker.Check(ctx)

			if report == nil {
				t.Fatal("Expected non-nil health report")
			}

			if len(report.Components) < tt.wantMinComponents {
				t.Errorf("Expected at least %d components, got %d", tt.wantMinComponents, len(report.Components))
			}

			if tt.config.EnableAdapters && report.Status != tt.wantStatus {
				t.Errorf("Expected status %s, got %s", tt.wantStatus, report.Status)
			}

			// Verify summary counts
			total := report.Summary["healthy"] + report.Summary["unhealthy"] + report.Summary["degraded"] + report.Summary["unknown"]
			if total != len(report.Components) {
				t.Errorf("Summary count mismatch: summary total=%d, components=%d", total, len(report.Components))
			}

			// Verify timestamp
			if report.Timestamp.IsZero() {
				t.Error("Expected non-zero timestamp")
			}

			// Verify all components have required fields
			for name, component := range report.Components {
				if component.Name == "" {
					t.Errorf("Component %s has empty name", name)
				}
				if component.Type == "" {
					t.Errorf("Component %s has empty type", name)
				}
				if component.Status == "" {
					t.Errorf("Component %s has empty status", name)
				}
				if component.Timestamp.IsZero() {
					t.Errorf("Component %s has zero timestamp", name)
				}
			}
		})
	}
}

func TestChecker_CheckWithAdapters(t *testing.T) {
	logger := zerolog.Nop()
	checker := health.NewChecker(&health.Config{
		Timeout:        5 * time.Second,
		EnableAdapters: true,
	}, logger)

	ctx := context.Background()
	report := checker.Check(ctx)

	// Verify specific adapters are checked
	expectedAdapters := []string{"smtp", "ses", "sns", "whatsapp-cloud"}
	for _, adapterName := range expectedAdapters {
		componentName := "adapter:" + adapterName
		component, exists := report.Components[componentName]
		if !exists {
			t.Errorf("Expected adapter %s to be checked", adapterName)
			continue
		}

		if component.Status == health.StatusUnknown {
			t.Errorf("Adapter %s has unknown status", adapterName)
		}

		// Verify metadata
		if component.Metadata == nil {
			t.Errorf("Adapter %s has nil metadata", adapterName)
			continue
		}

		if _, ok := component.Metadata["adapter_type"]; !ok {
			t.Errorf("Adapter %s missing adapter_type in metadata", adapterName)
		}

		if _, ok := component.Metadata["version"]; !ok {
			t.Errorf("Adapter %s missing version in metadata", adapterName)
		}
	}
}

func TestChecker_ContextCancellation(t *testing.T) {
	logger := zerolog.Nop()
	checker := health.NewChecker(&health.Config{
		Timeout:        10 * time.Second,
		EnableAdapters: true,
	}, logger)

	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Check should still complete (adapters are checked concurrently)
	report := checker.Check(ctx)

	if report == nil {
		t.Fatal("Expected non-nil health report even with cancelled context")
	}

	// Should still have some results
	if len(report.Components) == 0 {
		t.Error("Expected some components even with cancelled context")
	}
}

func TestChecker_NilConfig(t *testing.T) {
	logger := zerolog.Nop()

	// Should use default config
	checker := health.NewChecker(nil, logger)
	ctx := context.Background()

	report := checker.Check(ctx)

	if report == nil {
		t.Fatal("Expected non-nil health report with nil config")
	}

	// Default config should enable adapters
	if len(report.Components) < 4 {
		t.Errorf("Expected at least 4 adapter components with default config, got %d", len(report.Components))
	}
}

func TestChecker_CheckAdapter(t *testing.T) {
	logger := zerolog.Nop()
	checker := health.NewChecker(&health.Config{
		Timeout: 5 * time.Second,
	}, logger)

	// Create a test adapter (using SMTP as example)
	// We won't actually test the Ping() because we don't have real credentials
	// But we can test the method exists

	ctx := context.Background()

	// Test with a mock adapter
	mockAdapter := &mockAdapter{name: "test-adapter", pingErr: nil}
	result := checker.CheckAdapter(ctx, mockAdapter)

	if result.Name != "test-adapter" {
		t.Errorf("Expected name 'test-adapter', got %s", result.Name)
	}

	if result.Type != "adapter" {
		t.Errorf("Expected type 'adapter', got %s", result.Type)
	}

	if result.Status != health.StatusHealthy {
		t.Errorf("Expected status healthy for successful ping, got %s", result.Status)
	}

	if result.Latency == 0 {
		t.Error("Expected non-zero latency")
	}
}

// mockAdapter is a test adapter that implements the Adapter interface
type mockAdapter struct {
	name    string
	pingErr error
}

func (m *mockAdapter) Send(ctx context.Context, req interface{}) (string, error) {
	return "mock-message-id", nil
}

func (m *mockAdapter) Name() string {
	return m.name
}

func (m *mockAdapter) Ping(ctx context.Context) error {
	return m.pingErr
}
