package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kartex/imageprovider/internal/handlers"
	"github.com/kartex/imageprovider/internal/middleware"
	"github.com/kartex/imageprovider/internal/services"
	"github.com/kartex/imageprovider/internal/storage"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get storage path from environment or use default
	storagePath := os.Getenv("STORAGE_PATH")
	if storagePath == "" {
		storagePath = "./data"
	}

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Initialize storage
	fileStorage, err := storage.NewFileSystemStorage(storagePath)
	if err != nil {
		log.Fatalf("Failed to initialize file storage: %v", err)
	}

	// Initialize S3 storage if credentials are available
	var s3Storage storage.Storage
	if os.Getenv("S3_ENDPOINT") != "" {
		s3Storage, err = storage.NewS3Storage()
		if err != nil {
			log.Printf("Warning: Failed to initialize S3 storage: %v", err)
		}
	}

	// Initialize image service
	imageService := services.NewImageService(fileStorage, s3Storage)

	// Initialize handlers
	imageHandler := handlers.NewImageHandler(imageService)

	// Create router
	router := gin.Default()

	// Set release mode if not in debug
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Configure trusted proxies
	router.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// Apply middleware
	router.Use(middleware.SecurityMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RateLimitMiddleware())

	// Public routes
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/images/:id", imageHandler.GetImage)

	// Protected routes
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/images", imageHandler.CreateImage)
		protected.DELETE("/images/:id", imageHandler.DeleteImage)
		protected.GET("/images", imageHandler.ListImages)
	}

	// Start server
	bindAddress := os.Getenv("BIND_ADDRESS")
	if bindAddress == "" {
		bindAddress = ":8080"
	}
	log.Printf("Server starting on %s (configured from BIND_ADDRESS environment variable)", bindAddress)
	if err := router.Run(bindAddress); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
