package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
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

// Stream handles streaming responses from Ollama and collects the full assistant response
// It properly streams to the client while also returning the accumulated content
func Stream(ctx context.Context, reqBody []byte, url, method string) (func(w *bufio.Writer) (string, error), int, error) {
	log.Printf("Proxying request to: %s", url)

	// Create a response content builder
	var responseContent strings.Builder

	// Create the request
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fiber.StatusInternalServerError, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Create HTTP client
	client := createStreamingHTTPClient()

	// Make the request to Ollama
	resp, err := client.Do(req)
	if err != nil {
		return nil, fiber.StatusBadGateway, err
	}

	// Handle error responses from Ollama
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Ollama error response: %s", string(body))
		return nil, resp.StatusCode, err
	}

	// Use a buffered channel to wait for streaming to complete
	return func(w *bufio.Writer) (string, error) {
		return responseHandler(w, resp, &responseContent)
	}, resp.StatusCode, nil
}

func responseHandler(w *bufio.Writer, resp *http.Response, responseContent *strings.Builder) (string, error) {
	defer resp.Body.Close() // Ensure connection is closed when done
	buffer := make([]byte, 1024)
	lastFlushTime := time.Now()
	log.Printf("Starting to stream chat response")

	// Read the stream chunk by chunk
	for {
		// Read a chunk from the response - each JSON object is on a separate line
		n, err := resp.Body.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from Ollama: %v", err)
			}
			break
		}
		if n > 0 {
			data := buffer[:n]
			// Extract content from the JSON chunk
			content := extractContent(data)
			if content != "" {
				responseContent.WriteString(content)
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

			// Check if this is the last chunk (done=true)
			if isDone(data) {
				break
			}
		}
	}
	w.Flush()
	return responseContent.String(), nil
}

// extractContent parses Ollama message content from the response
func extractContent(data []byte) string {
	// Try to parse the JSON data
	var chunkData ChunkData
	if err := json.Unmarshal(data, &chunkData); err == nil {
		return chunkData.Message.Content
	}
	return ""
}

// isDone checks if a chunk indicates the end of the stream
func isDone(data []byte) bool {
	var chunkData ChunkData
	if err := json.Unmarshal(data, &chunkData); err == nil {
		return chunkData.Done
	}
	return false
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
