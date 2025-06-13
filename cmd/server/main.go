package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/pandeptwidyaop/tempfile/internal/config"
	"github.com/pandeptwidyaop/tempfile/internal/handlers"
	"github.com/pandeptwidyaop/tempfile/internal/services"
	"github.com/pandeptwidyaop/tempfile/internal/utils"
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

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	// Initialize services
	uploadService := services.NewUploadService(cfg)
	cleanupService := services.NewCleanupService(cfg)

	var templateService *services.TemplateService
	if cfg.EnableWebUI {
		templateService = services.NewTemplateService(cfg)
		if err := templateService.Initialize(); err != nil {
			log.Fatal("Failed to initialize templates:", err)
		}
	}

	// Initialize handlers
	apiHandler := handlers.NewAPIHandler(cfg, uploadService)
	fileHandler := handlers.NewFileHandler(cfg)

	var webHandler *handlers.WebHandler
	if cfg.EnableWebUI {
		webHandler = handlers.NewWebHandler(cfg, uploadService, templateService)
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: int(cfg.MaxFileSize),
	})

	// Setup middleware
	setupMiddleware(app, cfg)

	// Setup routes
	setupRoutes(app, cfg, apiHandler, webHandler, fileHandler)

	// Start cleanup routine
	go cleanupService.Start()

	// Print startup information
	printStartupInfo(cfg)

	// Start server
	log.Printf("🚀 Server starting on %s", cfg.Port)
	log.Fatal(app.Listen(cfg.Port))
}

// setupMiddleware configures middleware based on configuration
func setupMiddleware(app *fiber.App, cfg *config.Config) {
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

	// Serve static files if Web UI is enabled
	if cfg.EnableWebUI {
		app.Static("/static", cfg.StaticDir)
	}
}

// setupRoutes configures application routes
func setupRoutes(app *fiber.App, cfg *config.Config, apiHandler *handlers.APIHandler, webHandler *handlers.WebHandler, fileHandler *handlers.FileHandler) {
	// Health check endpoint (most specific first)
	app.Get("/health", apiHandler.HealthCheck)

	// Routes
	if cfg.EnableWebUI && webHandler != nil {
		// Web UI routes (specific routes first)
		app.Get("/success", webHandler.SuccessPage)
		app.Get("/", webHandler.UploadPage)
		app.Post("/", webHandler.UploadFileHandler)
	} else {
		// API only routes
		app.Post("/", apiHandler.UploadFile)
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
	log.Printf("   Debug Mode: %v", cfg.Debug)
}
