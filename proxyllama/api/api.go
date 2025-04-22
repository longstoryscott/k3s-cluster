package api

import (
	"fmt"
	"proxyllama/auth"
	"proxyllama/storage"

	"github.com/gofiber/fiber/v2"
)

// RegisterConversationRoutes adds conversation management endpoints
func RegisterConversationRoutes(app *fiber.App) {
	app.Get("/api/conversations", GetUserConversations)
	app.Get("/api/conversations/:id", GetConversation)
	app.Get("/api/conversations/:id/messages", GetConversationMessages)
	app.Delete("/api/conversations/:id", DeleteConversation)
	app.Put("/api/conversations/:id", UpdateConversation)
	app.Post("/api/conversations", CreateConversation)
}

// GetUserConversations returns all conversations for the authenticated user
func GetUserConversations(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)

	fmt.Println("GetUserConversations User ID:", userID)

	conversations, err := storage.GetUserConversations(c.Context(), userID)
	if err != nil {
		fmt.Println("Error retrieving conversations:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve conversations")
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
		fmt.Println("Error retrieving messages:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve messages")
	}

	return c.JSON(messages)
}

// DeleteConversation deletes a conversation and all its messages
func DeleteConversation(c *fiber.Ctx) error {
	userID := c.UserContext().Value(auth.UserIDKey).(string)
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		// Log the error for debugging
		fmt.Println("Error parsing conversation ID:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	// Verify ownership
	conversation, err := storage.GetConversation(c.Context(), conversationID)
	if err != nil {
		// Log the error for debugging
		fmt.Println("Error retrieving conversation:", err)
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}
	if conversation.UserID != userID {
		// Log the error for debugging
		fmt.Println("Access denied for user:", userID, "to conversation:", conversationID)
		return fiber.NewError(fiber.StatusForbidden, "Access denied")
	}

	// Add deletion function to storage package
	err = storage.DeleteConversation(c.Context(), conversationID)
	if err != nil {
		// Log the error for debugging
		fmt.Println("Error deleting conversation:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete conversation")
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
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update conversation")
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
		// Log the error for debugging
		fmt.Println("Error parsing request body:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	conversationID, err := storage.CreateConversation(c.Context(), userID, req.Model, req.Title)
	if err != nil {
		// Log the error for debugging
		fmt.Println("Error creating conversation:", err)
		// Return a 500 error with a message
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create conversation")
	}

	return c.JSON(fiber.Map{"conversation_id": conversationID})
}
