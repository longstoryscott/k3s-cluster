package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"proxyllama/auth"
	"proxyllama/config"
	pxcx "proxyllama/context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ChatRequest represents the structure of incoming chat requests
type ChatRequest struct {
	Model          string        `json:"model"`
	Messages       []ChatMessage `json:"messages"`
	ConversationID *int          `json:"conversationId"` // ui sends camelCase
	Stream         bool          `json:"stream,omitempty"`
}

// ChatMessage represents a single message in a chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChunkData represents the structure of a streaming chunk response
type ChunkData struct {
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	DoneReason         string      `json:"done_reason"`
	TotalDuration      float64     `json:"total_duration"`
	LoadDuration       float64     `json:"load_duration"`
	PromptEvalCount    int         `json:"prompt_eval_count"`
	PromptEvalDuration float64     `json:"prompt_eval_duration"`
	EvalCount          int         `json:"eval_count"`
	EvalDuration       float64     `json:"eval_duration"`
}

// ReverseProxyHandler forwards the request to Ollama and streams the response back to the client
func ReverseProxyHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		log.Printf("Received request for path: %s", path)

		// Special handling for chat endpoints
		if strings.Contains(path, "/chat") {
			log.Printf("Chat endpoint detected: %s", path)
			log.Printf("Got request body: %s", string(c.Body()))
			return handleChatRequest(c)
		}

		// For non-chat endpoints, just pass through
		return handleRegularProxyRequest(c)
	}
}

// createStreamingHTTPClient returns a configured HTTP client optimized for streaming responses
func createStreamingHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 0, // No timeout for streaming
		Transport: &http.Transport{
			IdleConnTimeout:       0,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			DisableKeepAlives:     false,
			ResponseHeaderTimeout: 0,
			ExpectContinueTimeout: 0,
			TLSHandshakeTimeout:   0,
		},
	}
}

// handleChatRequest handles requests to the chat endpoint with context tracking
func handleChatRequest(c *fiber.Ctx) error {
	// Parse the incoming request
	var chatReq ChatRequest
	if err := c.BodyParser(&chatReq); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	log.Printf("Parsed chat request: %+v", chatReq)

	// Always set streaming to true to prevent timeouts
	chatReq.Stream = true

	uid := c.UserContext().Value(auth.UserIDKey).(string)
	// Get or create conversation context
	ctx := c.Context()
	convCtx, err := pxcx.GetOrCreateConversation(ctx, uid, chatReq.Model, chatReq.ConversationID)
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
	if err := convCtx.AddUserMessage(ctx, userMessage); err != nil {
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

	log.Printf("Proxying request to: %s", targetUrl)
	// Log the request body for debugging
	log.Printf("Request body: %s", string(ollamaReqBody))

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", targetUrl, bytes.NewReader(ollamaReqBody))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create proxy request")
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive") // Add keep-alive for persistent connections

	// Use the shared HTTP client
	client := createStreamingHTTPClient()

	// Make the request to Ollama
	resp, err := client.Do(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "Failed to contact Ollama")
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("Ollama error response: %s", string(body))
		return c.Status(resp.StatusCode).Send(body)
	}

	// Set response headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no")

	c.Status(resp.StatusCode)

	// Stream response and collect assistant response
	var assistantResponse strings.Builder
	var debugChunkCount int

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer resp.Body.Close() // Ensure connection is closed when done

		buffer := make([]byte, 1024)
		lastFlushTime := time.Now()

		log.Printf("Starting to stream chat response")
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := buffer[:n]

				// Process Ollama data to extract the assistant response
				content := extractContent(data)
				if content != "" {
					assistantResponse.WriteString(content)
					debugChunkCount++
				}

				// Write to response
				if _, writeErr := w.Write(data); writeErr != nil {
					log.Printf("Error writing to response: %v", writeErr)
					break
				}

				// Flush frequently to avoid client timeouts
				// Always flush at least every 10 seconds even if no data
				if time.Since(lastFlushTime) > 10*time.Second {
					if flushErr := w.Flush(); flushErr != nil {
						log.Printf("Error flushing response: %v", flushErr)
						break
					}
					lastFlushTime = time.Now()
				} else if flushErr := w.Flush(); flushErr != nil {
					log.Printf("Error flushing response: %v", flushErr)
					break
				}
			}

			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from Ollama: %v", err)
				}
				break
			}
		}

		// Only store the assistant response if there was no context error
		if !contextError {
			// Store the complete assistant response
			fullResponse := assistantResponse.String()
			log.Printf("DEBUG: Final accumulated content size: %d bytes, chunk count: %d",
				len(fullResponse), debugChunkCount)

			if fullResponse != "" {
				log.Printf("Storing assistant response (length: %d)", len(fullResponse))

				// Create a separate context with timeout for database operation
				dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				// Use a transaction for reliability
				if err := convCtx.AddAssistantMessage(dbCtx, fullResponse); err != nil {
					log.Printf("Error storing assistant response: %v", err)
				} else {
					log.Printf("Successfully stored assistant message in conversation %d", convCtx.ConversationID)
				}
			} else {
				log.Printf("Empty assistant response, not storing")
				// Dump the raw data we received for debugging
			}

			// Add conversation ID to response metadata (last chunk)
			metadataChunk := fmt.Sprintf("\ndata: {\"conversation_id\": %d}\n\n", convCtx.ConversationID)
			if _, err := w.WriteString(metadataChunk); err != nil {
				log.Printf("Error writing metadata chunk: %v", err)
			}
			w.Flush()
		}

		log.Printf("Chat streaming complete")
	})

	return nil
}

// handleRegularProxyRequest handles regular proxy requests without context tracking
func handleRegularProxyRequest(c *fiber.Ctx) error {
	// Build the full target URL
	path := c.Path()
	targetUrl := strings.TrimSuffix(config.GetConfig().Ollama.BaseURL, "/") + c.Path()

	log.Printf("Proxying request to: %s", targetUrl)

	// Create the request
	req, err := http.NewRequestWithContext(c.Context(), c.Method(), targetUrl, bytes.NewReader(c.Body()))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create proxy request")
	}

	// Copy query parameters
	req.URL.RawQuery = string(c.Request().URI().QueryString())

	// Copy headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)
		if strings.ToLower(k) != "host" && strings.ToLower(k) != "connection" {
			req.Header.Set(k, v)
		}
	})

	// Set specific headers for streaming endpoints and force streaming when applicable
	isStreamingEndpoint := strings.Contains(path, "/generate") || strings.Contains(path, "/chat")
	if isStreamingEndpoint {
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")

		// If this is a POST request to a streaming endpoint, check and force stream=true
		if c.Method() == "POST" {
			var bodyData map[string]interface{}

			// Try to decode the body
			if err := json.Unmarshal(c.Body(), &bodyData); err == nil {
				// Force streaming to be true
				bodyData["stream"] = true

				// Rebuild the request body with updated streaming setting
				updatedBody, jsonErr := json.Marshal(bodyData)
				if jsonErr == nil {
					req, err = http.NewRequestWithContext(
						c.Context(),
						c.Method(),
						targetUrl,
						bytes.NewReader(updatedBody),
					)
					if err != nil {
						return fiber.NewError(fiber.StatusInternalServerError, "Failed to recreate proxy request")
					}

					// Re-add headers
					c.Request().Header.VisitAll(func(key, value []byte) {
						k := string(key)
						v := string(value)
						if strings.ToLower(k) != "host" && strings.ToLower(k) != "connection" {
							req.Header.Set(k, v)
						}
					})
					req.Header.Set("Content-Type", "application/json")
				}
			}
		}
	}

	// Use the shared HTTP client configuration
	client := createStreamingHTTPClient()

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "Failed to contact Ollama")
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("Ollama error response: %s", string(body))
		return c.Status(resp.StatusCode).Send(body)
	}

	// Copy headers from response
	for k, values := range resp.Header {
		for _, v := range values {
			c.Set(k, v)
		}
	}

	// Set streaming headers for streaming endpoints
	if isStreamingEndpoint {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")
		c.Set("X-Accel-Buffering", "no")
	}

	c.Status(resp.StatusCode)

	// Stream the response
	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer resp.Body.Close()

		buffer := make([]byte, 1024)
		lastFlushTime := time.Now()

		log.Printf("Starting to stream response for %s", path)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := buffer[:n]

				// Write to response
				if _, writeErr := w.Write(data); writeErr != nil {
					log.Printf("Error writing to response: %v", writeErr)
					break
				}

				// Flush frequently to avoid client timeouts
				// Always flush at least every 5 seconds even if no data
				if time.Since(lastFlushTime) > 5*time.Second {
					if flushErr := w.Flush(); flushErr != nil {
						log.Printf("Error flushing response: %v", flushErr)
						break
					}
					lastFlushTime = time.Now()
					log.Printf("Regular flush performed for %s", path)
				} else if isStreamingEndpoint {
					// For streaming endpoints, flush after every write
					if flushErr := w.Flush(); flushErr != nil {
						log.Printf("Error flushing response: %v", flushErr)
						break
					}
				}
			}

			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from Ollama: %v", err)
				}
				break
			}
		}

		// Final flush to ensure all data is sent
		w.Flush()
		log.Printf("Finished streaming response for %s", path)
	})

	return nil
}

// extractContent parses ollama message content (not SSE format) from the response
func extractContent(data []byte) string {
	// Try to parse the JSON data
	var chunkData ChunkData
	if err := json.Unmarshal(data, &chunkData); err == nil {
		return chunkData.Message.Content
	} else {
		// Log parsing errors for debugging
		log.Printf("DEBUG: Error parsing JSON chunk: %v, data: %s", err, string(data))
	}

	return ""
}
