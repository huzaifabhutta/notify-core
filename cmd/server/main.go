package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/huzaifabhutta/notify-core/internal/auth"
	"github.com/huzaifabhutta/notify-core/internal/config"
	appErrors "github.com/huzaifabhutta/notify-core/internal/errors"
	"github.com/huzaifabhutta/notify-core/internal/notify"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// CRITICAL: Validate configuration before starting
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Configuration loaded and validated successfully")

	// Load API keys from environment
	apiKeysStr := os.Getenv("API_KEYS")
	if apiKeysStr == "" {
		log.Fatalf("API_KEYS environment variable is required. Format: key1:tenant1,key2:tenant2")
	}

	validAPIKeys := auth.LoadAPIKeysFromEnv(apiKeysStr)
	if len(validAPIKeys) == 0 {
		log.Fatalf("No valid API keys configured. Please set API_KEYS environment variable.")
	}

	log.Printf("Loaded %d API key(s)", len(validAPIKeys))

	// Create Fiber app with security settings
	app := fiber.New(fiber.Config{
		AppName:      "Notify-Core v1.0",
		ServerHeader: "", // Hide server header for security
		ErrorHandler: customErrorHandler,
	})

	// Global middleware
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))

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
			var req notify.SendRequest
			if err := c.BodyParser(&req); err != nil {
				log.Printf("Failed to parse request body: %v", err)
				return c.Status(400).JSON(fiber.Map{
					"error":   appErrors.ErrInvalidRequest,
					"message": "Invalid request body. Please check your JSON syntax.",
				})
			}

			// Get tenant ID from auth middleware
			tenantID := c.Locals("tenant_id")
			log.Printf("Sending notification for tenant: %s, channel: %s, template: %s", tenantID, req.Channel, req.Template)

			// Send notification
			if err := notifyService.Send(c.Context(), &req); err != nil {
				// Log the full error server-side
				log.Printf("Failed to send notification: %v", err)

				// Return safe error to client
				var appErr *appErrors.AppError
				if e, ok := err.(*appErrors.AppError); ok {
					appErr = e
				} else {
					// Wrap unexpected errors
					appErr = appErrors.SendFailed(string(req.Channel), err)
				}

				return c.Status(appErr.StatusCode).JSON(fiber.Map{
					"error":   appErr.Code,
					"message": appErr.Message,
				})
			}

			log.Printf("Notification sent successfully for tenant: %s", tenantID)

			return c.JSON(fiber.Map{
				"success": true,
				"message": "Notification sent successfully",
			})
		},
	)

	// Start server
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📧 Email channel ready (SMTP: %s:%d)", cfg.SMTP.Host, cfg.SMTP.Port)
	log.Printf("🔒 Authentication enabled (%d API key(s))", len(validAPIKeys))
	log.Printf("⏱️  Rate limiting: 20 req/min per key, 100 req/min per IP")

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
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
	log.Printf("Error: %v", err)

	// Return safe error to client
	return c.Status(code).JSON(fiber.Map{
		"error":   errorCode,
		"message": message,
	})
}
