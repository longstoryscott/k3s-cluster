package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"path/filepath"
	"proxyllama/auth"
	pxcx "proxyllama/context"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/recherche"
	"runtime"

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

		cfg, err := pxcx.GetUserConfig(uid)
		if err != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": err,
			}).Error("Error retrieving user config")
			return handleError(err, fiber.StatusInternalServerError, "Failed to retrieve user configuration")
		}

		doWebSearch := cfg.WebSearch.Enabled

		if !doWebSearch && cfg.WebSearch.AutoDetect {
			doWebSearch = recherche.DetectSearchIntent(userMessage, nil)
		}

		// Check if web search is enabled
		if doWebSearch {
			// Attempt to perform a web search and inject results
			searchResult, err := recherche.QuickSearch(c.Context(), userMessage, cfg.WebSearch.MaxResults, cfg.WebSearch.IncludeResults)
			if err != nil {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"error": err,
				}).Warn("Error performing web search")
				// Non-critical error, we can continue without web search results
			}

			if searchResult != nil {
				// Inject the search results into the conversation context
				if err := cc.InjectWebSearchResult(c.Context(), *searchResult); err != nil {
					_, file, line, _ := runtime.Caller(0)
					logrus.WithFields(logrus.Fields{
						"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
						"line":  line,
						"error": err,
					}).Warn("Error injecting web search results")
					// Non-critical error, we can continue without web search results
				}
			}
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
	} else { // Fall back to the original request if we had any context errors
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Warn("Falling back to original request without context")
		// Use the original request but ensure the model is set correctly and stream is true
		fallbackReq := models.OllamaReq{
			Model:    chatReq.Model,
			Messages: chatReq.Messages,
			Stream:   true, // Always force streaming
		}
		ollamaReqBody, jsonErr = json.Marshal(fallbackReq)
		if jsonErr != nil {
			return handleError(jsonErr, fiber.StatusInternalServerError, "Failed to format request")
		}
	}

	headers := proxy.ContextHeaders{}
	c.Request().Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		valueStr := string(value)
		if keyStr != "Host" && keyStr != "Connection" {
			headers[keyStr] = valueStr
		}
	})

	c.Context().SetUserValue(proxy.ReqHeadersKey, headers)

	handler, statusCode, err := proxy.GetProxyHandler(c.Context(), ollamaReqBody, c.Path(), http.MethodPost, true)
	if err != nil {
		return handleError(err, fiber.StatusBadGateway, "Error during streaming")
	}
	c.Status(statusCode)
	var res string

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
			go func(response, userID string, convID int) {
				pxcx.RefineResponse(response, userMessage, userID, convID)
			}(res, cc.UserID, cc.ConversationID)
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
