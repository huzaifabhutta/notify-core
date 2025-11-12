package main

import (
	"context"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/ses"   // Register SES adapter
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/smtp"  // Register SMTP adapter
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/sms/sns"     // Register SNS adapter
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/whatsapp/cloud" // Register WhatsApp adapter
	_ "github.com/huzaifabhutta/notify-core/internal/channels/email"       // Register email channel
	_ "github.com/huzaifabhutta/notify-core/internal/channels/sms"         // Register SMS channel
	_ "github.com/huzaifabhutta/notify-core/internal/channels/whatsapp"    // Register WhatsApp channel
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/database"
	appErrors "github.com/huzaifabhutta/notify-core/internal/errors"
	"github.com/huzaifabhutta/notify-core/internal/health"
	appLogger "github.com/huzaifabhutta/notify-core/internal/logger"
	"github.com/huzaifabhutta/notify-core/internal/middleware"
	"github.com/huzaifabhutta/notify-core/internal/repository"
	"github.com/huzaifabhutta/notify-core/internal/services"
	"github.com/huzaifabhutta/notify-core/internal/tenantctx"
)

// initializeDatabase initializes the database connection and runs migrations
func initializeDatabase(cfg *config.Config) (*database.DB, error) {
	dbCfg := &database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
		MaxConns: cfg.Database.MaxConns,
		MaxIdle:  cfg.Database.MaxIdle,
	}

	db, err := database.New(dbCfg)
	if err != nil {
		return nil, err
	}

	// Run migrations automatically on startup
	appLogger.Logger.Info().Msg("Running database migrations...")
	migrator := database.NewMigrator(db)
	if err := migrator.Up(context.Background()); err != nil {
		return nil, err
	}
	appLogger.Logger.Info().Msg("Database migrations completed successfully")

	return db, nil
}

// setupTenantRoutes sets up routes with tenant authentication
func setupTenantRoutes(app *fiber.App, cfg *config.Config) error {
	// Initialize database
	db, err := initializeDatabase(cfg)
	if err != nil {
		return err
	}

	// Initialize services
	tenantRepo := repository.NewTenantRepository(db, cfg.Security.EncryptionKey)
	tenantService := services.NewTenantService(tenantRepo, appLogger.Logger)

	credentialResolver := services.NewCredentialResolver(cfg)

	// Feature flag: Enable V3 service with channel registry
	// Set ENABLE_V3=true in environment to use the new channel registry architecture
	enableV3 := os.Getenv("ENABLE_V3") == "true"

	// Create V2 service (fallback)
	notifyServiceV2 := services.NewNotifyServiceV2(cfg, credentialResolver, appLogger.Logger)

	// Create V3 service if enabled
	var notifyServiceV3 *services.NotifyServiceV3
	if enableV3 {
		notifyServiceV3 = services.NewNotifyServiceV3(cfg, credentialResolver, appLogger.Logger)
		appLogger.Logger.Info().Msg("🚀 V3 Channel Registry enabled - using pluggable channel architecture")
	} else {
		appLogger.Logger.Info().Msg("Using V2 Service (set ENABLE_V3=true to enable channel registry)")
	}

	// Initialize health checker
	healthChecker := health.NewChecker(&health.Config{
		Timeout:        3 * time.Second,
		EnableAdapters: true,
		EnableDatabase: false, // TODO: Add database health check in future
	}, appLogger.Logger)

	// Initialize middleware
	tenantAuth := middleware.NewTenantAuth(tenantService, appLogger.Logger)

	appLogger.Logger.Info().Msg("Tenant-based authentication enabled")

	// API v2 routes (with tenant authentication from database)
	v2 := app.Group("/v2")

	// Comprehensive health check (no auth required)
	v2.Get("/health", func(c *fiber.Ctx) error {
		report := healthChecker.Check(c.UserContext())

		// Determine HTTP status based on health
		statusCode := fiber.StatusOK
		switch report.Status {
		case health.StatusUnhealthy, health.StatusUnknown:
			statusCode = fiber.StatusServiceUnavailable
		}

		return c.Status(statusCode).JSON(report)
	})

	// Tenant management routes (requires tenant auth)
	tenants := v2.Group("/tenants", tenantAuth.Authenticate())

	// Get current tenant info
	tenants.Get("/me", func(c *fiber.Ctx) error {
		tenant, ok := tenantctx.GetTenantFromFiberContext(c)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to retrieve tenant from context",
			})
		}

		return c.JSON(fiber.Map{
			"status": "success",
			"tenant": fiber.Map{
				"id":     tenant.ID,
				"name":   tenant.Name,
				"active": tenant.Active,
				"channels": fiber.Map{
					"email":    tenant.HasEmailConfig(),
					"whatsapp": tenant.HasWhatsAppConfig(),
					"sms":      tenant.HasSMSConfig(),
				},
			},
		})
	})

	// Update current tenant
	tenants.Put("/me", func(c *fiber.Ctx) error {
		tenant, _ := tenantctx.GetTenantFromFiberContext(c)

		var updateReq services.UpdateTenantRequest
		if err := c.BodyParser(&updateReq); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid request body",
			})
		}

		// Convert to models.UpdateTenantRequest
		modelUpdateReq := &models.UpdateTenantRequest{}
		// Map fields...

		updatedTenant, err := tenantService.UpdateTenant(c.Context(), tenant.ID, modelUpdateReq)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to update tenant",
			})
		}

		return c.JSON(fiber.Map{
			"status": "success",
			"tenant": updatedTenant,
		})
	})

	// Protected /send endpoint with tenant authentication
	v2.Post("/send",
		tenantAuth.Authenticate(),
		limiter.New(limiter.Config{
			Max:        50,              // 50 requests per minute per tenant
			Expiration: 1 * time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				tenant, ok := tenantctx.GetTenantFromFiberContext(c)
				if ok {
					return string(rune(tenant.ID))
				}
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.Status(429).JSON(fiber.Map{
					"status":  "error",
					"error":   appErrors.ErrRateLimitExceeded,
					"message": "Rate limit exceeded. Maximum 50 requests per minute per tenant.",
				})
			},
		}),
		func(c *fiber.Ctx) error {
			start := time.Now()

			var req services.SendRequest
			if err := c.BodyParser(&req); err != nil {
				appLogger.Logger.Warn().
					Err(err).
					Str("request_id", c.GetRespHeader("X-Request-ID")).
					Msg("Failed to parse request body")
				return c.Status(400).JSON(fiber.Map{
					"status":  "error",
					"error":   appErrors.ErrInvalidRequest,
					"message": "Invalid request body. Please check your JSON syntax.",
				})
			}

			// Send notification using V3 if enabled, otherwise V2
			// This allows safe rollout with feature flag
			var resp *services.SendResponse
			var err error
			if enableV3 && notifyServiceV3 != nil {
				resp, err = notifyServiceV3.Send(c.UserContext(), &req)
			} else {
				resp, err = notifyServiceV2.Send(c.UserContext(), &req)
			}
			if err != nil {
				appLogger.Logger.Error().
					Err(err).
					Str("request_id", c.GetRespHeader("X-Request-ID")).
					Str("channel", string(req.Channel)).
					Dur("duration", time.Since(start)).
					Msg("Failed to send notification")

				// Return safe error to client
				var appErr *appErrors.AppError
				if e, ok := err.(*appErrors.AppError); ok {
					appErr = e
				} else {
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
				Str("request_id", c.GetRespHeader("X-Request-ID")).
				Int("tenant_id", resp.TenantID).
				Str("tenant_name", resp.TenantName).
				Str("channel", resp.Channel).
				Str("message_id", resp.MessageID).
				Str("source", resp.Source).
				Dur("duration", time.Since(start)).
				Msg("Notification sent successfully")

			return c.JSON(fiber.Map{
				"status":      "success",
				"message":     "Notification sent successfully",
				"message_id":  resp.MessageID,
				"channel":     resp.Channel,
				"tenant_id":   resp.TenantID,
				"tenant_name": resp.TenantName,
				"source":      resp.Source,
				"timestamp":   time.Now().Format(time.RFC3339),
			})
		},
	)

	return nil
}

// RunTenantMode starts the server in multi-tenant mode (database-backed authentication)
func RunTenantMode() {
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

	// Validate configuration (includes database and encryption key validation)
	if err := cfg.Validate(); err != nil {
		appLogger.Logger.Fatal().Err(err).Msg("Invalid configuration")
	}

	appLogger.Logger.Info().Msg("Configuration loaded and validated successfully")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Notify-Core v2.0 (Multi-Tenant)",
		ServerHeader: "",
		ErrorHandler: customErrorHandler,
	})

	// Global middleware
	app.Use(requestid.New(requestid.Config{
		Generator: func() string {
			return uuid.New().String()
		},
	}))

	// Custom logging middleware
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		requestID := c.GetRespHeader("X-Request-ID")

		err := c.Next()

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

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("CORS_ORIGINS"),
		AllowMethods: "GET,POST,PUT,DELETE",
		AllowHeaders: "Origin,Content-Type,Accept,X-API-Key,Authorization",
	}))

	// Global rate limiting
	app.Use(limiter.New(limiter.Config{
		Max:        200,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error":   appErrors.ErrRateLimitExceeded,
				"message": "Too many requests. Please try again later.",
			})
		},
	}))

	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "notify-core",
			"version": "2.0.0",
			"mode":    "multi-tenant",
			"status":  "running",
		})
	})

	// Setup tenant routes
	if err := setupTenantRoutes(app, cfg); err != nil {
		appLogger.Logger.Fatal().Err(err).Msg("Failed to setup tenant routes")
	}

	// Start server
	port := cfg.Server.Port
	appLogger.Logger.Info().
		Str("port", port).
		Str("mode", "multi-tenant").
		Msg("Starting notify-core server")

	if err := app.Listen(":" + port); err != nil {
		appLogger.Logger.Fatal().Err(err).Msg("Server failed to start")
	}
}
