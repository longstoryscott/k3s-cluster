package api

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"proxyllama/auth"
	"proxyllama/config"
	pxcx "proxyllama/context"
	"proxyllama/models"
	"proxyllama/proxy"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ReverseProxyHandler forwards the request to Ollama and streams the response back to the client
func ReverseProxyHandler(c *fiber.Ctx) error {
	// Parse the incoming request
	var chatReq models.OllamaReq
	if err := c.BodyParser(&chatReq); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Always set streaming to true to prevent timeouts
	chatReq.Stream = true

	uid := c.UserContext().Value(auth.UserIDKey).(string)
	// Get or create conversation context
	ctx := c.Context()
	convCtx, err := pxcx.GetOrCreateConversation(ctx, uid, chatReq.Model, chatReq.ConversationId)
	if err != nil {
		log.Printf("Error getting conversation context: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to process conversation")
	}

	// Get the user message from the request (last message from user)
	var userMessage string
	for i := len(chatReq.Messages) - 1; i >= 0; i-- {
		if chatReq.Messages[i].Role == "user" {
			userMessage = chatReq.Messages[i].Content
			break
		}
	}

	// Add the user message to context
	contextError := false
	if err := convCtx.AddUserMessage(c.Context(), userMessage); err != nil {
		log.Printf("Error adding user message: %v", err)
		contextError = true
	}

	var ollamaReqBody []byte
	var jsonErr error

	// Only use context if there was no error adding the user message
	if !contextError {
		// Convert conversation context to Ollama format (includes summaries)
		ollamaReqBody, jsonErr = convCtx.ToJSON()
		if jsonErr != nil {
			log.Printf("Error converting context to JSON: %v", jsonErr)
			contextError = true
		}
	}

	// Fall back to the original request if we had any context errors
	if contextError {
		log.Printf("Falling back to original request without context")
		// Use the original request but ensure the model is set correctly and stream is true
		fallbackReq := map[string]interface{}{
			"model":    chatReq.Model,
			"messages": chatReq.Messages,
			"stream":   true, // Always force streaming
		}
		ollamaReqBody, jsonErr = json.Marshal(fallbackReq)
		if jsonErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to format request")
		}
	}

	// Build the target URL
	targetUrl := config.GetConfig().Ollama.BaseURL + c.Path()

	// Set response headers for streaming
	c.Set("Content-Type", "application/json")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no")

	handler, statusCode, err := proxy.Stream(c.Context(), ollamaReqBody, targetUrl, http.MethodPost)
	if err != nil {
		log.Printf("Error during streaming: %v", err)
		return c.Status(fiber.StatusBadGateway).SendString("Error during streaming")
	}
	c.Status(statusCode)
	var res string

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		res, err = handler(w)
		if err != nil {
			log.Printf("Error during handler execution: %v", err)
			c.Status(fiber.StatusInternalServerError).SendString("Error during handler execution")
			return
		}

		// Only store the assistant response if there was no context errorå
		if !contextError {
			if res != "" {
				log.Printf("Storing assistant response (length: %d)", len(res))
				go func(res string) {
					dbCtx := context.Background()
					ctx, cancel := context.WithTimeout(dbCtx, time.Minute*10)
					defer cancel()

					if err := convCtx.AddAssistantMessage(ctx, res); err != nil {
						log.Printf("Error storing assistant response: %v", err)
					} else {
						log.Printf("Successfully stored assistant message in conversation %d", convCtx.ConversationID)
					}
				}(res)
			} else {
				log.Printf("Empty assistant response, not storing")
			}
		}

		log.Printf("Chat streaming complete")
	})

	return nil
}
