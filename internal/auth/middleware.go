package auth

import (
	"github.com/gofiber/fiber/v2"
)

// Middleware creates an API key authentication middleware
func Middleware(validAPIKeys map[string]string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get API key from header
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "AUTH_REQUIRED",
				"message": "API key is required. Please provide X-API-Key header.",
			})
		}

		// Validate API key and get tenant ID
		tenantID, ok := validAPIKeys[apiKey]
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "INVALID_API_KEY",
				"message": "Invalid API key provided.",
			})
		}

		// Store tenant info in context for multi-tenancy
		c.Locals("api_key", apiKey)
		c.Locals("tenant_id", tenantID)

		return c.Next()
	}
}

// LoadAPIKeysFromEnv loads API keys from environment variables
// Format: API_KEYS=key1:tenant1,key2:tenant2
func LoadAPIKeysFromEnv(keysStr string) map[string]string {
	keys := make(map[string]string)

	if keysStr == "" {
		return keys
	}

	// Split by comma to get individual key:tenant pairs
	pairs := splitAndTrim(keysStr, ",")
	for _, pair := range pairs {
		parts := splitAndTrim(pair, ":")
		if len(parts) == 2 {
			apiKey := parts[0]
			tenantID := parts[1]
			keys[apiKey] = tenantID
		}
	}

	return keys
}

// splitAndTrim splits a string and trims whitespace from each part
func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for i := 0; i < len(s); {
		// Find next separator
		idx := -1
		for j := i; j < len(s); j++ {
			if s[j] == sep[0] {
				idx = j
				break
			}
		}

		if idx == -1 {
			// No more separators, add rest of string
			part := trim(s[i:])
			if part != "" {
				parts = append(parts, part)
			}
			break
		}

		// Add part before separator
		part := trim(s[i:idx])
		if part != "" {
			parts = append(parts, part)
		}
		i = idx + 1
	}

	return parts
}

// trim removes leading and trailing whitespace
func trim(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing whitespace
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
