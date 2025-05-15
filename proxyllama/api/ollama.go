package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"proxyllama/auth"
	"proxyllama/config"
	pxcx "proxyllama/context"
	"proxyllama/models"
	"proxyllama/proxy"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
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
	cc, err := pxcx.GetCachedConversation(uid, *chatReq.ConversationId)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to process conversation")
	}

	usrCfg, err := pxcx.GetUserConfig(cc.UserID)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to get user config")
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
	if err := cc.AddUserMessage(c.Context(), userMessage); err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Error("Error adding user message")
		contextError = true
	}

	// Retrieve memories if there was no error adding the user message
	if !contextError {
		// Attempt to retrieve and inject relevant memories for the user's query
		if err := cc.RetrieveAndInjectMemories(c.Context(), userMessage); err != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": err,
			}).Warn("Error retrieving memories")
			// Non-critical error, we can continue without memories
		}
	}

	var ollamaReqBody []byte
	var jsonErr error

	// Only use context if there was no error adding the user message
	if !contextError {
		// Convert conversation context to Ollama format (includes summaries)
		ollamaReqBody, jsonErr = cc.ToJSON()
		if jsonErr != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": jsonErr,
			}).Error("Error converting context to JSON")
			contextError = true
		}
	}

	// Fall back to the original request if we had any context errors
	if contextError {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Warn("Falling back to original request without context")
		// Use the original request but ensure the model is set correctly and stream is true
		fallbackReq := map[string]any{
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

	// Create a copy of important variables for use in the goroutine
	// This ensures we don't have references to request-scoped objects after the request ends
	conversationID := cc.ConversationID
	finalUserMsg := userMessage
	ccUserID := cc.UserID
	enableResponseCritique := *usrCfg.Summarization.EnableResponseCritique
	enableResponseFiltering := *usrCfg.Summarization.EnableResponseFiltering

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		res, err = handler(w)
		if err != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": err,
			}).Error("Error during handler execution")
			return
		}

		// Only store the assistant response if there was no context error
		if res != "" {
			// Apply refinement steps to the response in a separate goroutine
			// to avoid keeping the client connection open longer than necessary
			go func(response string) {
				// Create a new background context for this goroutine
				// This is crucial to avoid using the request context which may be canceled
				bgCtx := context.Background()
				ctx, cancel := context.WithTimeout(bgCtx, time.Minute*10)
				defer cancel()

				// Get a fresh conversation context reference to avoid stale data
				freshCC, err := pxcx.GetCachedConversation(ccUserID, conversationID)
				if err != nil {
					_, file, line, _ := runtime.Caller(0)
					logrus.WithFields(logrus.Fields{
						"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
						"line":  line,
						"error": err,
					}).Error("Error getting fresh conversation context")
					return
				}

				refinedRes := response // Start with the original response

				// Step 1: Apply basic filtering if enabled
				if enableResponseFiltering {
					refinedRes = pxcx.FilterResponseText(refinedRes)
				}

				// Step 2: Apply self-critique if enabled
				if enableResponseCritique {
					// Use the fresh background context and conversation context
					critique, err := freshCC.GetCritiqueForResponse(ctx, refinedRes)
					if err == nil && critique != "" {
						_, file, line, _ := runtime.Caller(0)
						logrus.WithFields(logrus.Fields{
							"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
							"line":   line,
							"length": len(critique),
						}).Info("Got critique for response")
						improvedRes, err := freshCC.ImproveResponseWithCritique(ctx, finalUserMsg, refinedRes, critique)
						if err != nil {
							_, file, line, _ := runtime.Caller(0)
							logrus.WithFields(logrus.Fields{
								"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
								"line":  line,
								"error": err,
							}).Error("Failed to improve response with critique")
						} else if improvedRes != "" {
							_, file, line, _ := runtime.Caller(0)
							logrus.WithFields(logrus.Fields{
								"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
								"line": line,
							}).Info("Applied critique improvements to response")
							refinedRes = improvedRes
						}
					} else if err != nil {
						_, file, line, _ := runtime.Caller(0)
						logrus.WithFields(logrus.Fields{
							"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
							"line":  line,
							"error": err,
						}).Error("Failed to get critique")
					}
				}

				// Store the refined response
				var finalRes string
				if refinedRes != response {
					_, file, line, _ := runtime.Caller(0)
					logrus.WithFields(logrus.Fields{
						"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
						"line": line,
					}).Info("Response was refined/improved before storing")
					finalRes = refinedRes
				} else {
					finalRes = response
				}

				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":   line,
					"length": len(finalRes),
				}).Info("Storing assistant response")
				if err := freshCC.AddAssistantMessage(ctx, finalRes); err != nil {
					_, file, line, _ := runtime.Caller(0)
					logrus.WithFields(logrus.Fields{
						"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
						"line":  line,
						"error": err,
					}).Error("Error storing assistant response")
				} else {
					_, file, line, _ := runtime.Caller(0)
					logrus.WithFields(logrus.Fields{
						"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
						"line":           line,
						"conversationId": freshCC.ConversationID,
					}).Info("Successfully stored assistant message in conversation")
				}
			}(res)
		} else if res == "" {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line": line,
			}).Warn("Empty assistant response, not storing")
		}

		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Chat streaming complete")
	})

	return nil
}
