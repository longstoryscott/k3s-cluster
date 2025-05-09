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

// RegisterOllamaRoutes adds Ollama-related endpoints to the app
func RegisterOllamaRoutes(app *fiber.App) {
	app.Post("/api/chat", ReverseProxyHandler)
}

// ReverseProxyHandler forwards the request to Ollama and streams the response back to the client
func ReverseProxyHandler(c *fiber.Ctx) error {
	// Parse the incoming request
	var chatReq models.OllamaReq
	if err := c.BodyParser(&chatReq); err != nil {
		return handleError(err, fiber.StatusBadRequest, "Invalid request body")
	}

	// Always set streaming to true to prevent timeouts
	chatReq.Stream = true

	uid := c.UserContext().Value(auth.UserIDKey).(string)
	// Get or create conversation context
	ctx := c.Context()
	convCtx, err := pxcx.GetOrCreateConversation(ctx, uid, chatReq.Model, chatReq.ConversationId)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to process conversation")
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

	// Retrieve memories if there was no error adding the user message
	if !contextError {
		// Attempt to retrieve and inject relevant memories for the user's query
		if err := convCtx.RetrieveAndInjectMemories(c.Context(), userMessage); err != nil {
			log.Printf("Error retrieving memories: %v", err)
			// Non-critical error, we can continue without memories
		}
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
			return handleError(jsonErr, fiber.StatusInternalServerError, "Failed to format request")
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
		return handleError(err, fiber.StatusBadGateway, "Error during streaming")
	}
	c.Status(statusCode)
	var res string

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		res, err = handler(w)
		if err != nil {
			log.Printf("Error during handler execution: %v", err)
			return
		}

		// Only store the assistant response if there was no context error
		if res != "" && !contextError {
			// Apply refinement steps to the response
			refinedRes := res // Start with the original response
			conf := config.GetConfig()

			// Step 1: Apply basic filtering if enabled
			if conf.Summarization.EnableResponseFiltering {
				refinedRes = pxcx.FilterResponseText(refinedRes)
			}

			// Step 2: Apply self-critique if enabled
			if conf.Summarization.EnableResponseCritique {
				critique, err := pxcx.GetCritiqueForResponse(c.Context(), refinedRes, convCtx.Model)
				if err == nil && critique != "" {
					log.Printf("Got critique for response: length %d characters", len(critique))
					improvedRes, err := pxcx.ImproveResponseWithCritique(c.Context(), userMessage, refinedRes, critique, convCtx.Model)
					if err != nil {
						log.Printf("Failed to improve response with critique: %v", err)
					} else if improvedRes != "" {
						log.Printf("Applied critique improvements to response")
						refinedRes = improvedRes
					}
				} else if err != nil {
					log.Printf("Failed to get critique: %v", err)
				}
			}

			// Store the refined response
			var finalRes string
			if refinedRes != res {
				log.Printf("Response was refined/improved before storing")
				finalRes = refinedRes
			} else {
				finalRes = res
			}

			log.Printf("Storing assistant response (length: %d)", len(finalRes))
			go func(finalResponse string) {
				dbCtx := context.Background()
				ctx, cancel := context.WithTimeout(dbCtx, time.Minute*10)
				defer cancel()

				if err := convCtx.AddAssistantMessage(ctx, finalResponse); err != nil {
					log.Printf("Error storing assistant response: %v", err)
				} else {
					log.Printf("Successfully stored assistant message in conversation %d", convCtx.ConversationID)
				}
			}(finalRes)
		} else if res == "" {
			log.Printf("Empty assistant response, not storing")
		}

		log.Printf("Chat streaming complete")
	})

	return nil
}
