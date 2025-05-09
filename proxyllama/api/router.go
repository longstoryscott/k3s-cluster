package api

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterAllRoutes registers all the API routes with the fiber app
func RegisterAllRoutes(app *fiber.App) {
	// Register conversation routes
	RegisterConversationRoutes(app)

	// Register Ollama proxy routes
	RegisterOllamaRoutes(app)

	// Register research routes
	RegisterResearchRoutes(app)
}
