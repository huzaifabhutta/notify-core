package main

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	"github.com/huzaifabhutta/notify-core/internal/auth"
	"github.com/huzaifabhutta/notify-core/internal/config"
	appErrors "github.com/huzaifabhutta/notify-core/internal/errors"
	appLogger "github.com/huzaifabhutta/notify-core/internal/logger"
	"github.com/huzaifabhutta/notify-core/internal/notify"
)

func main() {
	// Initialize structured logging
	appLogger.Init(appLogger.Config{
		Level:      getEnvOrDefault("LOG_LEVEL", "info"),
		JSONFormat: getEnvOrDefault("LOG_FORMAT", "console") == "json",
	})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		appLogger.Logger.Fatal().Err(err).Msg("Failed to load config")
	}

	// CRITICAL: Validate configuration before starting
	if err := cfg.Validate(); err != nil {
		appLogger.Logger.Fatal().Err(err).Msg("Invalid configuration")
	}

	appLogger.Logger.Info().Msg("Configuration loaded and validated successfully")

	// Load API keys from environment
	apiKeysStr := os.Getenv("API_KEYS")
	if apiKeysStr == "" {
		appLogger.Logger.Fatal().Msg("API_KEYS environment variable is required. Format: key1:tenant1,key2:tenant2")
	}

	validAPIKeys := auth.LoadAPIKeysFromEnv(apiKeysStr)
	if len(validAPIKeys) == 0 {
		appLogger.Logger.Fatal().Msg("No valid API keys configured. Please set API_KEYS environment variable.")
	}

	appLogger.Logger.Info().Int("api_keys_count", len(validAPIKeys)).Msg("API keys loaded")

	// Create Fiber app with security settings
	app := fiber.New(fiber.Config{
		AppName:      "Notify-Core v1.0",
		ServerHeader: "", // Hide server header for security
		ErrorHandler: customErrorHandler,
	})

	// Global middleware
	// Request ID middleware
	app.Use(requestid.New(requestid.Config{
		Generator: func() string {
			return uuid.New().String()
		},
	}))

	// Custom logging middleware
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		requestID := c.GetRespHeader("X-Request-ID")

		// Process request
		err := c.Next()

		// Log request
		logger := appLogger.Logger.With().
			Str("request_id", requestID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("latency", time.Since(start)).
			Str("ip", c.IP()).
			Logger()

		if err != nil {
			logger.Error().Err(err).Msg("Request failed")
		} else {
			logger.Info().Msg("Request completed")
		}

		return err
	})

	// CORS configuration - restrict to specific origins in production
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("CORS_ORIGINS"), // e.g., "https://yourdomain.com"
		AllowMethods: "GET,POST",
		AllowHeaders: "Origin,Content-Type,Accept,X-API-Key",
	}))

	// Rate limiting - global rate limit
	app.Use(limiter.New(limiter.Config{
		Max:        100,                   // 100 requests
		Expiration: 1 * time.Minute,       // per minute
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by IP
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error":   appErrors.ErrRateLimitExceeded,
				"message": "Too many requests. Please try again later.",
			})
		},
	}))

	// Initialize notification service
	notifyService := notify.NewService(cfg)

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "notify-core",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})

	// Protected /send endpoint with authentication and stricter rate limiting
	app.Post("/send",
		// Authentication middleware
		auth.Middleware(validAPIKeys),
		// Stricter rate limit for send endpoint
		limiter.New(limiter.Config{
			Max:        20,              // 20 requests
			Expiration: 1 * time.Minute, // per minute per API key
			KeyGenerator: func(c *fiber.Ctx) string {
				// Rate limit by API key
				apiKey := c.Locals("api_key")
				if apiKey != nil {
					return apiKey.(string)
				}
				return c.IP() // Fallback to IP
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.Status(429).JSON(fiber.Map{
					"error":   appErrors.ErrRateLimitExceeded,
					"message": "Rate limit exceeded. Maximum 20 requests per minute.",
				})
			},
		}),
		// Handler
		func(c *fiber.Ctx) error {
			start := time.Now()

			var req notify.SendRequest
			if err := c.BodyParser(&req); err != nil {
				appLogger.Logger.Warn().
					Err(err).
					Str("request_id", c.GetRespHeader("X-Request-ID")).
					Msg("Failed to parse request body")
				return c.Status(400).JSON(fiber.Map{
					"error":   appErrors.ErrInvalidRequest,
					"message": "Invalid request body. Please check your JSON syntax.",
				})
			}

			// Get tenant ID from auth middleware
			tenantID := c.Locals("tenant_id").(string)
			requestID := c.GetRespHeader("X-Request-ID")

			// Create context with tenant and request IDs
			ctx := appLogger.WithTenantID(c.Context(), tenantID)
			ctx = appLogger.WithRequestID(ctx, requestID)

			// Mask sensitive data for logging
			maskedTo := req.To
			if req.Channel == notify.ChannelEmail {
				maskedTo = appLogger.MaskEmail(req.To)
			} else {
				maskedTo = appLogger.MaskPhone(req.To)
			}

			appLogger.Logger.Info().
				Str("request_id", requestID).
				Str("tenant_id", tenantID).
				Str("channel", string(req.Channel)).
				Str("template", req.Template).
				Str("to", maskedTo).
				Msg("Processing notification request")

			// Send notification
			resp, err := notifyService.Send(ctx, &req)
			if err != nil {
				// Log the full error server-side
				appLogger.Logger.Error().
					Err(err).
					Str("request_id", requestID).
					Str("tenant_id", tenantID).
					Str("channel", string(req.Channel)).
					Str("template", req.Template).
					Dur("duration", time.Since(start)).
					Msg("Failed to send notification")

				// Return safe error to client
				var appErr *appErrors.AppError
				if e, ok := err.(*appErrors.AppError); ok {
					appErr = e
				} else {
					// Wrap unexpected errors
					appErr = appErrors.SendFailed(string(req.Channel), err)
				}

				return c.Status(appErr.StatusCode).JSON(fiber.Map{
					"status":    "error",
					"error":     appErr.Code,
					"message":   appErr.Message,
					"timestamp": time.Now().Format(time.RFC3339),
				})
			}

			appLogger.Logger.Info().
				Str("request_id", requestID).
				Str("tenant_id", tenantID).
				Str("channel", string(req.Channel)).
				Str("template", req.Template).
				Str("message_id", resp.MessageID).
				Dur("duration", time.Since(start)).
				Msg("Notification sent successfully")

			// Return standardized success response
			return c.JSON(fiber.Map{
				"status":     "success",
				"message":    "Notification sent successfully",
				"message_id": resp.MessageID,
				"channel":    resp.Channel,
				"timestamp":  time.Now().Format(time.RFC3339),
			})
		},
	)

	// Start server
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}

	appLogger.Logger.Info().
		Str("port", port).
		Str("smtp_host", cfg.SMTP.Host).
		Int("smtp_port", cfg.SMTP.Port).
		Int("api_keys_count", len(validAPIKeys)).
		Msg("Server starting")

	appLogger.Logger.Info().Msg("📧 Email channel ready")
	appLogger.Logger.Info().Msg("🔒 Authentication enabled")
	appLogger.Logger.Info().Msg("⏱️  Rate limiting: 20 req/min per key, 100 req/min per IP")

	if err := app.Listen(":" + port); err != nil {
		appLogger.Logger.Fatal().Err(err).Msg("Failed to start server")
	}
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// customErrorHandler handles errors globally with safe error messages
func customErrorHandler(c *fiber.Ctx, err error) error {
	// Default to 500 Internal Server Error
	code := fiber.StatusInternalServerError
	errorCode := appErrors.ErrInternalError
	message := "An internal error occurred"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		switch code {
		case 404:
			errorCode = "NOT_FOUND"
			message = "Resource not found"
		case 400:
			errorCode = appErrors.ErrInvalidRequest
			message = "Bad request"
		case 401:
			errorCode = appErrors.ErrAuthRequired
			message = "Authentication required"
		case 403:
			errorCode = appErrors.ErrInvalidAPIKey
			message = "Forbidden"
		case 429:
			errorCode = appErrors.ErrRateLimitExceeded
			message = "Rate limit exceeded"
		default:
			message = e.Message
		}
	}

	// Check if it's an AppError
	if appErr, ok := err.(*appErrors.AppError); ok {
		code = appErr.StatusCode
		errorCode = appErr.Code
		message = appErr.Message
	}

	// Log the error server-side
	appLogger.Logger.Error().
		Err(err).
		Str("request_id", c.GetRespHeader("X-Request-ID")).
		Str("path", c.Path()).
		Int("status", code).
		Msg("Request error")

	// Return safe error to client
	return c.Status(code).JSON(fiber.Map{
		"error":   errorCode,
		"message": message,
	})
}
