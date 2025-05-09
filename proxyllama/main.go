package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"proxyllama/api"
	"proxyllama/auth"
	"proxyllama/config"
	convContext "proxyllama/context"
	"proxyllama/storage"
)

func main() {
	conf := config.GetConfig()
	psqlconn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		conf.Database.User,
		conf.Database.Password,
		conf.Database.Host,
		conf.Database.Port,
		conf.Database.DBName,
		conf.Database.SSLMode,
	)

	// Init DB with a context for timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), conf.Database.ConnectTimeout)
	defer cancel()

	if err := storage.InitDB(psqlconn); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	// Initialize database schema
	storage.InitSchema(ctx)

	// Initialize research schema
	storage.InitResearchSchema(ctx)

	// Initialize Redis for storage caching
	if err := storage.InitStorageCache(); err != nil {
		log.Printf("Warning: Failed to initialize Redis storage cache: %v", err)
		log.Printf("Storage will operate without Redis caching")
	} else if conf.Redis.Enabled {
		log.Printf("Redis storage cache initialized successfully")
		// Clean up Redis connections when the application exits
		defer storage.CloseRedisCache()
	}

	// Initialize the conversation cache with configured settings
	if conf.Redis.Enabled {
		log.Printf("Initializing conversation cache with Redis at %s:%d, TTL: %v",
			conf.Redis.Host, conf.Redis.Port, conf.Redis.ConversationTTL)
	} else {
		log.Printf("Redis disabled, using in-memory cache with TTL: %v", conf.Redis.ConversationTTL)
	}
	convContext.InitCache(conf.Redis.ConversationTTL)

	// Create a new Fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
		// Add streaming capability
		StreamRequestBody: true,
	})

	// Add logger middleware
	app.Use(logger.New(
		logger.Config{
			Format:     "${time} ${status} - ${latency} ${method} ${path}\n",
			TimeFormat: "2006-01-02 15:04:05",
			TimeZone:   "Local",
		},
	)).Use(func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Set("Access-Control-Max-Age", "3600")
		return c.Next()
	}).Options("/*", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	}).Use(auth.WithAuth)

	// Register conversation API routes
	api.RegisterConversationRoutes(app)

	// Register research API routes
	api.RegisterResearchRoutes(app)

	// Setup reverse proxy handler with chunk processing
	app.All("/*", api.ReverseProxyHandler)

	// Use port from configuration instead of hardcoding it
	serverAddress := fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)
	log.Printf("Server started on %s", serverAddress)
	log.Fatal(app.Listen(serverAddress))
}
