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
	"proxyllama/proxy"
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

	// Init DB
	if err := storage.InitDB(psqlconn); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	// Initialize database schema
	storage.InitSchema(context.Background())

	// Create a new Fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
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

	// Setup reverse proxy handler
	app.All("/*", proxy.StreamProxyHandler(conf.Ollama.BaseURL, func(chunk []byte) {
		// Handle the chunk of data here
		log.Printf("Received chunk: %s", string(chunk))
	}))

	// Start the server
	log.Println("Server started on :8080")
	log.Fatal(app.Listen(":8080"))
}
