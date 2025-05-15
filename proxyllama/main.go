package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"

	"proxyllama/api"
	"proxyllama/config"
	"proxyllama/context"
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

	if err := storage.InitDB(psqlconn); err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Fatal("Failed to connect to db")
	}
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":       filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":       line,
		"connection": psqlconn,
	}).Info("Connected to PostgreSQL database")

	// Initialize research schema
	// storage.InitResearchSchema(ctx)

	// Initialize Redis for storage caching
	if err := storage.InitStorageCache(); err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Failed to initialize Redis storage cache")
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Warn("Storage will operate without Redis caching")
	} else if conf.Redis.Enabled {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Redis storage cache initialized successfully")
		// Clean up Redis connections when the application exits
		defer storage.CloseRedisCache()
	}

	// Initialize the conversation cache with configured settings
	if conf.Redis.Enabled {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
			"host": conf.Redis.Host,
			"port": conf.Redis.Port,
			"ttl":  conf.Redis.ConversationTTL,
		}).Info("Initializing conversation cache with Redis")
	} else {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
			"ttl":  conf.Redis.ConversationTTL,
		}).Info("Redis disabled, using in-memory cache")
	}
	duration, err := time.ParseDuration(conf.Redis.ConversationTTL)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Fatal("Invalid conversation TTL")
	}
	context.InitCache(duration)

	// Create a new Fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
		// Add streaming capability
		StreamRequestBody: true,
	})

	// Router
	api.RegisterAllRoutes(app)

	// Use port from configuration instead of hardcoding it
	serverAddress := fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)
	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":    filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":    line,
		"address": serverAddress,
	}).Info("Server started")

	// Use logrus.Fatal for fatal error handling
	if err := app.Listen(serverAddress); err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Fatal("Server failed to start")
	}
}
