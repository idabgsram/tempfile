package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/pandeptwidyaop/tempfile/internal/config"
	"github.com/pandeptwidyaop/tempfile/internal/handlers"
	"github.com/pandeptwidyaop/tempfile/internal/middleware"
	"github.com/pandeptwidyaop/tempfile/internal/ratelimit"
	"github.com/pandeptwidyaop/tempfile/internal/services"
	"github.com/pandeptwidyaop/tempfile/internal/utils"
	"github.com/pandeptwidyaop/tempfile/web"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal("Invalid configuration:", err)
	}

	// Validate trusted proxies if rate limiting is enabled
	if cfg.EnableRateLimit {
		if err := cfg.ValidateTrustedProxies(); err != nil {
			log.Fatal("Invalid trusted proxy configuration:", err)
		}
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(cfg.UploadDir, 0750); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	// Initialize services
	uploadService := services.NewUploadService(cfg)
	pasteService := services.NewPasteService(cfg)
	cleanupService := services.NewCleanupService(cfg)

	var templateService *services.TemplateService
	var staticService *services.StaticService

	if cfg.EnableWebUI {
		// Initialize template service with embedded templates
		templateService = services.NewTemplateService(cfg, web.TemplateFiles)
		if err := templateService.Initialize(); err != nil {
			log.Fatal("Failed to initialize embedded templates:", err)
		}

		// Initialize static service with hybrid loading
		staticService, err = services.NewStaticService(cfg, web.StaticFiles)
		if err != nil {
			log.Fatal("Failed to initialize static service:", err)
		}
	}

	// Initialize handlers
	apiHandler := handlers.NewAPIHandler(cfg, uploadService)
	fileHandler := handlers.NewFileHandler(cfg)

	var webHandler *handlers.WebHandler
	var pasteHandler *handlers.PasteHandler
	if cfg.EnableWebUI {
		webHandler = handlers.NewWebHandler(cfg, uploadService, templateService)
		pasteHandler = handlers.NewPasteHandler(cfg, pasteService, templateService)
	}

	// Initialize rate limiter if enabled
	var rateLimiter ratelimit.RateLimiter
	if cfg.EnableRateLimit {
		rateLimiterConfig := &ratelimit.Config{
			Store:            cfg.RateLimitStore,
			UploadsPerMinute: cfg.RateLimitUploadsPerMinute,
			BytesPerHour:     cfg.RateLimitBytesPerHour,
			WindowMinutes:    cfg.RateLimitWindowMinutes,
			TrustedProxies:   cfg.RateLimitTrustedProxies,
			IPHeaders:        cfg.RateLimitIPHeaders,
			WhitelistIPs:     cfg.RateLimitWhitelistIPs,
			CustomLimits:     convertCustomLimits(cfg.RateLimitCustomLimits),
			RedisURL:         cfg.RedisURL,
			RedisPassword:    cfg.RedisPassword,
			RedisDB:          cfg.RedisDB,
			RedisPoolSize:    cfg.RedisPoolSize,
			RedisTimeout:     cfg.RedisTimeout,
		}

		// Validate rate limiter configuration
		if err := ratelimit.ValidateConfig(rateLimiterConfig); err != nil {
			log.Fatal("Invalid rate limiter configuration:", err)
		}

		// Create rate limiter based on store type
		switch cfg.RateLimitStore {
		case "redis":
			var err error
			rateLimiter, err = ratelimit.NewRedisRateLimiter(rateLimiterConfig)
			if err != nil {
				log.Fatal("Failed to create Redis rate limiter:", err)
			}
			log.Printf("✅ Redis rate limiter initialized")
		case "memory":
			rateLimiter = ratelimit.NewDefaultMemoryRateLimiter(rateLimiterConfig)
			log.Printf("✅ Memory rate limiter initialized")
		default:
			log.Fatal("Invalid rate limit store:", cfg.RateLimitStore)
		}

		log.Printf("✅ Rate limiter enabled: %d uploads/%d min, %s/hour, %d whitelisted IPs, %d custom endpoints",
			cfg.RateLimitUploadsPerMinute,
			cfg.RateLimitWindowMinutes,
			utils.FormatBytes(cfg.RateLimitBytesPerHour),
			len(cfg.RateLimitWhitelistIPs),
			len(cfg.RateLimitCustomLimits))
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: int(cfg.MaxFileSize),
	})

	// Setup middleware
	setupMiddleware(app, cfg, staticService, rateLimiter)

	// Setup routes
	setupRoutes(app, cfg, apiHandler, webHandler, fileHandler, pasteHandler, rateLimiter)

	// Start cleanup routine
	go cleanupService.Start()

	// Print startup information
	printStartupInfo(cfg)

	// Start server
	log.Printf("🚀 Server starting on %s", cfg.Port)
	log.Fatal(app.Listen(cfg.Port))
}

// setupMiddleware configures middleware based on configuration
func setupMiddleware(app *fiber.App, cfg *config.Config, staticService *services.StaticService, rateLimiter ratelimit.RateLimiter) {
	// Security headers
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "1; mode=block")
		return c.Next()
	})

	// Conditionally add middleware based on config
	if cfg.EnableLogging {
		app.Use(logger.New(logger.Config{
			Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
		}))
	}

	if cfg.EnableCORS {
		corsConfig := cors.New()
		if cfg.CORSOrigins != "*" {
			corsConfig = cors.New(cors.Config{
				AllowOrigins: cfg.CORSOrigins,
			})
		}
		app.Use(corsConfig)
	}

	// Rate limiting middleware (before routes)
	if cfg.EnableRateLimit && rateLimiter != nil {
		ipDetector := ratelimit.NewIPDetectorWithWhitelist(
			cfg.RateLimitTrustedProxies,
			cfg.RateLimitIPHeaders,
			cfg.RateLimitWhitelistIPs,
		)

		rateLimiterMiddleware := middleware.NewRateLimiter(middleware.RateLimiterConfig{
			RateLimiter: rateLimiter,
			IPDetector:  ipDetector,
			SkipPaths:   []string{"/health", "/static"},
		})

		app.Use(rateLimiterMiddleware)
		log.Println("✅ Rate limiting middleware configured")
	}

	// Serve embedded static files if Web UI is enabled
	if cfg.EnableWebUI && staticService != nil {
		app.Get("/static/*", staticService.Handler())
		log.Println("✅ Embedded static files configured at /static/*")
	}
}

// convertCustomLimits converts config custom limits to rate limiter format
func convertCustomLimits(configLimits map[string]config.RateLimitEndpointConfig) map[string]ratelimit.EndpointConfig {
	result := make(map[string]ratelimit.EndpointConfig)
	for endpoint, limit := range configLimits {
		result[endpoint] = ratelimit.EndpointConfig{
			UploadsPerMinute: limit.UploadsPerMinute,
			BytesPerHour:     limit.BytesPerHour,
			WindowMinutes:    limit.WindowMinutes,
		}
	}
	return result
}

// setupRoutes configures application routes
func setupRoutes(app *fiber.App, cfg *config.Config, apiHandler *handlers.APIHandler, webHandler *handlers.WebHandler, fileHandler *handlers.FileHandler, pasteHandler *handlers.PasteHandler, rateLimiter ratelimit.RateLimiter) {
	// Health check endpoint (most specific first)
	app.Get("/health", apiHandler.HealthCheck)

	// Routes
	if cfg.EnableWebUI && webHandler != nil {
		// Web UI routes (specific routes first)
		app.Get("/success", webHandler.SuccessPage)
		app.Get("/", webHandler.UploadPage)

		// Upload route with post-processing middleware
		if cfg.EnableRateLimit && rateLimiter != nil {
			postProcessMiddleware := middleware.NewRateLimiterPostProcess(rateLimiter)
			app.Post("/", webHandler.UploadFileHandler, postProcessMiddleware)
		} else {
			app.Post("/", webHandler.UploadFileHandler)
		}
	} else {
		// API only routes
		if cfg.EnableRateLimit && rateLimiter != nil {
			postProcessMiddleware := middleware.NewRateLimiterPostProcess(rateLimiter)
			app.Post("/", apiHandler.UploadFile, postProcessMiddleware)
		} else {
			app.Post("/", apiHandler.UploadFile)
		}
	}

	// Paste routes (must be before wildcard download route)
	if pasteHandler != nil {
		if cfg.EnableRateLimit && rateLimiter != nil {
			postProcessMiddleware := middleware.NewRateLimiterPostProcess(rateLimiter)
			app.Post("/paste", pasteHandler.CreatePaste, postProcessMiddleware)
		} else {
			app.Post("/paste", pasteHandler.CreatePaste)
		}
		app.Get("/p/:id", pasteHandler.ViewPaste)
		app.Get("/p/:id/raw", pasteHandler.ViewPasteRaw)
	}

	// File download route (wildcard route LAST)
	app.Get("/:filename", fileHandler.DownloadFile)
}

// printStartupInfo prints configuration information at startup
func printStartupInfo(cfg *config.Config) {
	log.Println("📁 TempFiles Server Configuration:")
	log.Printf("   Environment: %s", cfg.AppEnv)
	log.Printf("   Port: %s", cfg.Port)
	log.Printf("   Public URL: %s", cfg.PublicURL)
	log.Printf("   Upload Directory: %s", cfg.UploadDir)
	log.Printf("   Max File Size: %s", utils.FormatBytes(cfg.MaxFileSize))
	log.Printf("   File Expiry: %d hour(s)", cfg.FileExpiryHours)
	log.Printf("   Cleanup Interval: %d second(s)", cfg.CleanupIntervalSeconds)
	log.Printf("   CORS Enabled: %v", cfg.EnableCORS)
	log.Printf("   Logging Enabled: %v", cfg.EnableLogging)
	log.Printf("   Web UI Enabled: %v", cfg.EnableWebUI)
	if cfg.EnableWebUI {
		log.Printf("   Static Assets: Embedded (standalone binary)")
	}
	log.Printf("   Debug Mode: %v", cfg.Debug)

	// Rate limiting info
	if cfg.EnableRateLimit {
		log.Printf("   Rate Limiting: Enabled")
		log.Printf("     Store: %s", cfg.RateLimitStore)
		log.Printf("     Upload Limit: %d per %d minutes", cfg.RateLimitUploadsPerMinute, cfg.RateLimitWindowMinutes)
		log.Printf("     Bytes Limit: %s per hour", utils.FormatBytes(cfg.RateLimitBytesPerHour))
		log.Printf("     Trusted Proxies: %d configured", len(cfg.RateLimitTrustedProxies))
	} else {
		log.Printf("   Rate Limiting: Disabled")
	}
}
