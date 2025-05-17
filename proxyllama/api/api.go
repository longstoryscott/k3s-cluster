package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"proxyllama/auth"
	"proxyllama/config"
	"proxyllama/context"
	"proxyllama/storage"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// handleError is a helper function that logs the error and returns a fiber error
// to standardize error handling across API endpoints
func handleError(err error, status int, message string) error {
	_, file, line, _ := runtime.Caller(1)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"error": err,
	}).Error(message)
	return fiber.NewError(status, message)
}

// RegisterConversationRoutes adds conversation management endpoints
func RegisterConversationRoutes(app *fiber.App) {
	app.Get("/api/conversations", GetUserConversations)
	app.Get("/api/conversations/:id", GetConversation)
	app.Get("/api/conversations/:id/messages", GetConversationMessages)
	app.Delete("/api/conversations/:id", DeleteConversation)
	app.Put("/api/conversations/:id", UpdateConversation)
	app.Post("/api/conversations", CreateConversation)
	app.Get("/api/models", GetModels)
	app.Get("/api/conversations/:id/summarize", SummarizeMessages)
}

// GetUserConversations returns all conversations for the authenticated user
func GetUserConversations(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)

	conversations, err := storage.GetUserConversations(c.Context(), userID)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to retrieve conversations")
	}

	return c.JSON(conversations)
}

// GetConversation returns a specific conversation
func GetConversation(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	conversation, err := storage.GetConversation(c.Context(), conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}

	// Verify ownership
	if conversation.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "Access denied")
	}

	return c.JSON(conversation)
}

// GetConversationMessages returns all messages in a conversation
func GetConversationMessages(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	// Verify ownership
	conversation, err := storage.GetConversation(c.Context(), conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}
	if conversation.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "Access denied")
	}

	messages, err := storage.GetConversationHistory(c.Context(), conversationID)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to retrieve messages")
	}

	return c.JSON(messages)
}

// DeleteConversation deletes a conversation and all its messages
func DeleteConversation(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return handleError(err, fiber.StatusBadRequest, "Invalid conversation ID")
	}

	// Verify ownership
	conversation, err := storage.GetConversation(c.Context(), conversationID)
	if err != nil {
		return handleError(err, fiber.StatusNotFound, "Conversation not found")
	}
	if conversation.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "Access denied")
	}

	// Add deletion function to storage package
	err = storage.DeleteConversation(c.Context(), conversationID)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to delete conversation")
	}

	return c.SendStatus(fiber.StatusOK)
}

// UpdateConversation updates conversation details (title)
func UpdateConversation(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	// Parse request body
	var req struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Verify ownership
	conversation, err := storage.GetConversation(c.Context(), conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}
	if conversation.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "Access denied")
	}

	// Update the title
	err = storage.UpdateConversationTitle(c.Context(), conversationID, req.Title)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to update conversation")
	}

	return c.SendStatus(fiber.StatusOK)
}

// CreateConversation creates a new conversation
func CreateConversation(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)
	var req struct {
		Model string `json:"model"`
		Title string `json:"title"`
	}
	if err := c.BodyParser(&req); err != nil {
		return handleError(err, fiber.StatusBadRequest, "Invalid request body")
	}

	cc, err := context.GetOrCreateConversation(c.Context(), userID, nil)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to create conversation context")
	}

	return c.JSON(fiber.Map{"conversation_id": cc.ConversationID})
}

// GetModels returns the available models
func GetModels(c *fiber.Ctx) error {
	conf := config.GetConfig()
	ollamaURL := conf.Ollama.BaseURL
	targetURL := fmt.Sprintf("%s/api/tags", strings.TrimSuffix(ollamaURL, "/"))

	// Create a request to Ollama's /api/tags endpoint
	req, err := http.NewRequestWithContext(c.Context(), "GET", targetURL, nil)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to create proxy request")
	}

	// Copy headers from original request
	c.Request().Header.VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)
		if strings.ToLower(k) != "host" && strings.ToLower(k) != "connection" {
			req.Header.Set(k, v)
		}
	})

	// Create HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make the request to Ollama
	resp, err := client.Do(req)
	if err != nil {
		return handleError(err, fiber.StatusBadGateway, "Failed to contact Ollama")
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to read Ollama response")
	}

	// If Ollama returns an error, pass it through
	if resp.StatusCode >= 400 {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":       filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":       line,
			"statusCode": resp.StatusCode,
		}).Error("Ollama error response: " + string(body))
		return c.Status(resp.StatusCode).Send(body)
	}

	// Return the response to the client
	c.Set("Content-Type", "application/json")
	return c.Status(resp.StatusCode).Send(body)
}

// SummarizeMessages summarizes the messages with the specified ids
func SummarizeMessages(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return handleError(err, fiber.StatusBadRequest, "Invalid conversation ID")
	}

	convCtx, err := context.GetCachedConversation(userID, conversationID)
	if err != nil {
		return handleError(err, fiber.StatusNotFound, "Conversation not found")
	}

	// Verify ownership
	conversation, err := storage.GetConversation(c.Context(), conversationID)
	if err != nil {
		return handleError(err, fiber.StatusNotFound, "Conversation not found")
	}
	if conversation.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "Access denied")
	}

	summary, err := convCtx.SummarizeMessages(c.Context())
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to summarize messages")
	}

	return c.JSON(summary)
}
