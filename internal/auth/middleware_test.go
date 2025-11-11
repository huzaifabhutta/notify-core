package auth

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestMiddleware_Success(t *testing.T) {
	app := fiber.New()

	validKeys := map[string]string{
		"test-key-123": "tenant-1",
		"test-key-456": "tenant-2",
	}

	app.Use(Middleware(validKeys))
	app.Get("/test", func(c *fiber.Ctx) error {
		tenantID := c.Locals("tenant_id").(string)
		return c.JSON(fiber.Map{"tenant": tenantID})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-key-123")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestMiddleware_MissingKey(t *testing.T) {
	app := fiber.New()

	validKeys := map[string]string{
		"test-key-123": "tenant-1",
	}

	app.Use(Middleware(validKeys))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No X-API-Key header

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("Expected error message in response body")
	}
}

func TestMiddleware_InvalidKey(t *testing.T) {
	app := fiber.New()

	validKeys := map[string]string{
		"test-key-123": "tenant-1",
	}

	app.Use(Middleware(validKeys))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestLoadAPIKeysFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "single key",
			input: "key1:tenant1",
			expected: map[string]string{
				"key1": "tenant1",
			},
		},
		{
			name:  "multiple keys",
			input: "key1:tenant1,key2:tenant2",
			expected: map[string]string{
				"key1": "tenant1",
				"key2": "tenant2",
			},
		},
		{
			name:  "keys with spaces",
			input: " key1:tenant1 , key2:tenant2 ",
			expected: map[string]string{
				"key1": "tenant1",
				"key2": "tenant2",
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:  "malformed entry ignored",
			input: "key1:tenant1,invalid,key2:tenant2",
			expected: map[string]string{
				"key1": "tenant1",
				"key2": "tenant2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LoadAPIKeysFromEnv(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d keys, got %d", len(tt.expected), len(result))
			}

			for key, expectedTenant := range tt.expected {
				if tenant, ok := result[key]; !ok {
					t.Errorf("Expected key %s not found", key)
				} else if tenant != expectedTenant {
					t.Errorf("For key %s, expected tenant %s, got %s", key, expectedTenant, tenant)
				}
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "simple split",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			input:    " a , b , c ",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty parts ignored",
			input:    "a,,b",
			sep:      ",",
			expected: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parts, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("Missing part at index %d", i)
				} else if result[i] != expected {
					t.Errorf("At index %d, expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}
