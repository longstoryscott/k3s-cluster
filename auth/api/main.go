package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/sirupsen/logrus"

	"usrmgr/auth"
	"usrmgr/config"
	"usrmgr/handlers"
)

func main() {
	app := fiber.New()

	conf := config.GetAppConfig()
	// Set logrus log level from config
	switch conf.LogLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
	fmtter := new(logrus.TextFormatter)
	fmtter.TimestampFormat = "2006-01-02 15:04:05"
	logrus.SetFormatter(fmtter)
	fmtter.FullTimestamp = true

	app.Use(logger.New()).Use(func(c fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Set("Access-Control-Max-Age", "3600")
		return c.Next()
	}).Options("/*", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	}).Use(auth.WithAuth)

	// Create API routes group
	api := app.Group("/api")

	// Register routes from separate handler files
	handlers.RegisterSearchHandler(api, config.GetLDAPConfig())
	handlers.RegisterChangePasswordHandler(api, config.GetLDAPConfig())
	handlers.RegisterAddUserHandler(api, config.GetLDAPConfig())
	handlers.RegisterDeleteUserHandler(api, config.GetLDAPConfig())

	// Start server
	log.Fatal(app.Listen(":3333"))
}
