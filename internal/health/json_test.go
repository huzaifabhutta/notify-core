package health_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/huzaifabhutta/notify-core/internal/health"
)

func TestDurationMillis_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantJSON string
	}{
		{
			name:     "1 second",
			duration: 1 * time.Second,
			wantJSON: "1000",
		},
		{
			name:     "500 milliseconds",
			duration: 500 * time.Millisecond,
			wantJSON: "500",
		},
		{
			name:     "1.5 seconds",
			duration: 1500 * time.Millisecond,
			wantJSON: "1500",
		},
		{
			name:     "100 microseconds",
			duration: 100 * time.Microsecond,
			wantJSON: "0.1",
		},
		{
			name:     "zero",
			duration: 0,
			wantJSON: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := health.DurationMillis(tt.duration)
			got, err := json.Marshal(d)
			if err != nil {
				t.Fatalf("Failed to marshal duration: %v", err)
			}

			if string(got) != tt.wantJSON {
				t.Errorf("MarshalJSON() = %s, want %s", string(got), tt.wantJSON)
			}
		})
	}
}

func TestComponentHealth_MarshalJSON(t *testing.T) {
	component := health.ComponentHealth{
		Name:      "test-adapter",
		Type:      "adapter",
		Status:    health.StatusHealthy,
		Message:   "all good",
		Latency:   health.DurationMillis(1500 * time.Millisecond),
		Timestamp: time.Date(2025, 11, 11, 10, 30, 0, 0, time.UTC),
		Metadata: map[string]interface{}{
			"adapter_type": "email",
			"version":      "1.0.0",
		},
	}

	got, err := json.Marshal(component)
	if err != nil {
		t.Fatalf("Failed to marshal ComponentHealth: %v", err)
	}

	// Verify the JSON contains latency_ms as a number (not nanoseconds)
	var result map[string]interface{}
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Check latency_ms field
	latencyMS, ok := result["latency_ms"].(float64)
	if !ok {
		t.Fatalf("latency_ms field is not a number, got type %T", result["latency_ms"])
	}

	// Should be 1500 milliseconds, not 1500000000 nanoseconds
	if latencyMS != 1500 {
		t.Errorf("latency_ms = %v, want 1500", latencyMS)
	}

	// Verify other fields
	if result["name"] != "test-adapter" {
		t.Errorf("name = %v, want test-adapter", result["name"])
	}
	if result["status"] != "healthy" {
		t.Errorf("status = %v, want healthy", result["status"])
	}
}

func TestHealthReport_MarshalJSON(t *testing.T) {
	report := health.HealthReport{
		Status:    health.StatusHealthy,
		Timestamp: time.Date(2025, 11, 11, 10, 30, 0, 0, time.UTC),
		Components: map[string]health.ComponentHealth{
			"adapter:smtp": {
				Name:      "smtp",
				Type:      "adapter",
				Status:    health.StatusHealthy,
				Message:   "registered",
				Latency:   health.DurationMillis(250 * time.Millisecond),
				Timestamp: time.Date(2025, 11, 11, 10, 30, 0, 0, time.UTC),
			},
			"adapter:ses": {
				Name:      "ses",
				Type:      "adapter",
				Status:    health.StatusHealthy,
				Message:   "registered",
				Latency:   health.DurationMillis(100 * time.Millisecond),
				Timestamp: time.Date(2025, 11, 11, 10, 30, 0, 0, time.UTC),
			},
		},
		Summary: map[string]int{
			"healthy":   2,
			"unhealthy": 0,
			"degraded":  0,
			"unknown":   0,
		},
	}

	got, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal HealthReport: %v", err)
	}

	// Verify JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Check components
	components, ok := result["components"].(map[string]interface{})
	if !ok {
		t.Fatalf("components is not a map")
	}

	// Check SMTP adapter latency
	smtpComponent, ok := components["adapter:smtp"].(map[string]interface{})
	if !ok {
		t.Fatalf("adapter:smtp is not a map")
	}

	smtpLatency, ok := smtpComponent["latency_ms"].(float64)
	if !ok {
		t.Fatalf("smtp latency_ms is not a number")
	}

	if smtpLatency != 250 {
		t.Errorf("SMTP latency_ms = %v, want 250", smtpLatency)
	}

	// Check SES adapter latency
	sesComponent, ok := components["adapter:ses"].(map[string]interface{})
	if !ok {
		t.Fatalf("adapter:ses is not a map")
	}

	sesLatency, ok := sesComponent["latency_ms"].(float64)
	if !ok {
		t.Fatalf("ses latency_ms is not a number")
	}

	if sesLatency != 100 {
		t.Errorf("SES latency_ms = %v, want 100", sesLatency)
	}

	// Verify summary
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary is not a map")
	}

	if summary["healthy"].(float64) != 2 {
		t.Errorf("summary.healthy = %v, want 2", summary["healthy"])
	}
}
