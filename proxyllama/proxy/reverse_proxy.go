package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"proxyllama/context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
)

// ChatRequest represents the structure of incoming chat requests
type ChatRequest struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	ConversationID *int      `json:"conversation_id,omitempty"`
}

// Message represents a single message in a chat
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ModelMap struct {
	Model      string `json:"model"`
	PathPrefix string `json:"path_prefix"`
}

type ModelMapList []ModelMap

var modelMapList ModelMapList = ModelMapList{
	{Model: "default", PathPrefix: ""},
	{Model: "phi3.5", PathPrefix: "/phi3-5"},
	{Model: "llama3-8b", PathPrefix: "/llama3-8b"},
}

func NewReverseProxy(target string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		log.Printf("Request path: %s", path)
		// Build the target URL
		targetURL := strings.TrimSuffix(target, "/") + path

		// Map the model to the correct path prefix
		for _, modelMap := range modelMapList {
			if strings.Contains(path, modelMap.Model) {
				targetURL = fmt.Sprintf("%s%s", targetURL, modelMap.PathPrefix)
			}
		}

		// Configure the proxy for streaming
		if err := proxy.Do(c, targetURL); err != nil {
			log.Printf("Proxy error: %v", err)
			return err
		}

		// Ensure streaming headers are set correctly
		if strings.Contains(path, "/generate") || strings.Contains(path, "/chat") {
			c.Set("Content-Type", "text/event-stream")
			c.Set("Cache-Control", "no-cache")
			c.Set("Connection", "keep-alive")
			c.Set("Transfer-Encoding", "chunked")
		}

		return nil
	}
}

// StreamProxyHandler forwards the request to Ollama and streams the response back to the client
func StreamProxyHandler(ollamaUrl string, chunk func([]byte)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		requestBody := string(c.Body())

		// Extract the path prefix based on model name in request body
		var pathPrefix string
		for _, modelMap := range modelMapList {
			if strings.Contains(requestBody, fmt.Sprintf("\"model\":\"%s\"", modelMap.Model)) ||
				strings.Contains(requestBody, fmt.Sprintf("\"model\": \"%s\"", modelMap.Model)) {
				pathPrefix = modelMap.PathPrefix
				log.Printf("Detected model %s, using path prefix: %s", modelMap.Model, pathPrefix)
				break
			}
		}

		// Build the full target URL by combining base URL, path prefix, and path
		targetUrl := fmt.Sprintf("%s%s%s",
			strings.TrimSuffix(ollamaUrl, "/"),
			pathPrefix,
			path)

		log.Printf("Proxying request to: %s", targetUrl)
		// Prepare the proxy request to Ollama
		req, err := http.NewRequestWithContext(c.Context(), c.Method(), targetUrl, bytes.NewReader(c.Body()))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create proxy request")
		}

		// Copy query parameters
		req.URL.RawQuery = string(c.Request().URI().QueryString())

		// Copy all headers from the incoming request
		c.Request().Header.VisitAll(func(key, value []byte) {
			k := string(key)
			v := string(value)
			if strings.ToLower(k) != "host" && strings.ToLower(k) != "connection" {
				req.Header.Set(k, v)
			}
		})

		// Set specific headers for streaming if needed
		if strings.Contains(path, "/generate") || strings.Contains(path, "/chat") {
			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("Cache-Control", "no-cache")
		}

		// Create client with longer timeouts for streaming
		client := &http.Client{
			Timeout: 0, // No timeout for streaming
			Transport: &http.Transport{
				IdleConnTimeout:     0, // Important: prevent idle timeout during streaming
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
			},
		}

		// Make the request to Ollama
		resp, err := client.Do(req)
		if err != nil {
			return fiber.NewError(fiber.StatusBadGateway, "Failed to contact Ollama")
		}

		// Log response status
		log.Printf("Ollama responded with status: %d", resp.StatusCode)

		// If Ollama returns an error, pass it through
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close() // Close here since we've read everything
			log.Printf("Ollama error response: %s", string(body))
			// Return the error body
			return c.Status(resp.StatusCode).Send(body)
		}

		// Set headers from Ollama response
		for k, values := range resp.Header {
			for _, v := range values {
				c.Set(k, v)
			}
		}

		// Set streaming headers
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")
		c.Set("X-Accel-Buffering", "no") // Disable buffering in Nginx if you're using it

		c.Status(resp.StatusCode)

		// Stream response body as it arrives
		c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
			defer resp.Body.Close() // Close the body when streaming is finished

			buffer := make([]byte, 1024) // Smaller buffer for more frequent flushes

			log.Printf("Starting to stream response")
			for {
				n, err := resp.Body.Read(buffer)
				if n > 0 {
					go chunk(buffer[:n])
					if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
						log.Printf("Error writing to response: %v", writeErr)
						break
					}
					if flushErr := w.Flush(); flushErr != nil {
						log.Printf("Error flushing response: %v", flushErr)
						break
					}
				}

				if err != nil {
					if err != io.EOF {
						log.Printf("Error reading from Ollama: %v", err)
					} else {
						log.Printf("Finished streaming response (EOF)")
					}
					break
				}
			}
			log.Printf("Streaming complete")
		})

		return nil
	}
}

// StreamProxyHandler forwards the request to Ollama and streams the response back to the client
func StreamProxyHandler2(ollamaUrl string, chunk func([]byte)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Get user ID from context (set by auth middleware)
		userID := c.Locals("user_id").(string)

		// Special handling for chat endpoints
		if strings.Contains(path, "/chat") {
			return handleChatRequest(c, ollamaUrl, userID, chunk)
		}

		// For non-chat endpoints, just pass through
		return handleRegularProxyRequest(c, ollamaUrl, chunk)
	}
}

// handleChatRequest handles requests to the chat endpoint with context tracking
func handleChatRequest(c *fiber.Ctx, ollamaUrl, userID string, chunk func([]byte)) error {
	// Parse the incoming request
	var chatReq ChatRequest
	if err := c.BodyParser(&chatReq); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Get or create conversation context
	ctx := c.Context()
	convCtx, err := context.GetOrCreateConversation(ctx, userID, chatReq.Model, chatReq.ConversationID)
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
	if err := convCtx.AddUserMessage(ctx, userMessage); err != nil {
		log.Printf("Error adding user message: %v", err)
	}

	// Convert conversation context to Ollama format
	ollamaReqBody, err := convCtx.ToJSON()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to format request")
	}

	// Build the target URL
	targetUrl := fmt.Sprintf("%s/api/chat", strings.TrimSuffix(ollamaUrl, "/"))

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", targetUrl, bytes.NewReader(ollamaReqBody))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create proxy request")
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Create HTTP client
	client := &http.Client{
		Timeout: 0, // No timeout for streaming
		Transport: &http.Transport{
			IdleConnTimeout:     0,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

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

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer resp.Body.Close()

		buffer := make([]byte, 1024)

		log.Printf("Starting to stream chat response")
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := buffer[:n]

				// Process SSE data to extract the assistant response
				content := extractContent(data)
				if content != "" {
					assistantResponse.WriteString(content)
				}

				// Forward chunk to caller
				go chunk(data)

				// Write to response
				if _, writeErr := w.Write(data); writeErr != nil {
					log.Printf("Error writing to response: %v", writeErr)
					break
				}
				if flushErr := w.Flush(); flushErr != nil {
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

		// Store the complete assistant response
		fullResponse := assistantResponse.String()
		if fullResponse != "" {
			if err := convCtx.AddAssistantMessage(ctx, fullResponse); err != nil {
				log.Printf("Error storing assistant response: %v", err)
			}
		}

		// Add conversation ID to response metadata (last chunk)
		metadataChunk := fmt.Sprintf("\ndata: {\"conversation_id\": %d}\n\n", convCtx.ConversationID)
		w.WriteString(metadataChunk)
		w.Flush()

		log.Printf("Chat streaming complete")
	})

	return nil
}

// handleRegularProxyRequest handles regular proxy requests without context tracking
func handleRegularProxyRequest(c *fiber.Ctx, ollamaUrl string, chunk func([]byte)) error {
	// Build the full target URL
	path := c.Path()
	targetUrl := fmt.Sprintf("%s%s", strings.TrimSuffix(ollamaUrl, "/"), path)

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

	// Set specific headers for streaming if needed
	if strings.Contains(path, "/generate") {
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			IdleConnTimeout:     0,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

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

	// Set streaming headers
	if strings.Contains(path, "/generate") {
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

		log.Printf("Starting to stream response")
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := buffer[:n]
				go chunk(data)

				if _, writeErr := w.Write(data); writeErr != nil {
					log.Printf("Error writing to response: %v", writeErr)
					break
				}
				if flushErr := w.Flush(); flushErr != nil {
					log.Printf("Error flushing response: %v", flushErr)
					break
				}
			}

			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from Ollama: %v", err)
				} else {
					log.Printf("Finished streaming response (EOF)")
				}
				break
			}
		}
		log.Printf("Streaming complete")
	})

	return nil
}

// extractContent parses SSE data to extract message content
func extractContent(data []byte) string {
	str := string(data)
	lines := strings.Split(str, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")

			// Try to parse the JSON
			var response struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}

			if err := json.Unmarshal([]byte(jsonData), &response); err == nil {
				return response.Message.Content
			}
		}
	}

	return ""
}
