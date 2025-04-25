package api

import (
	"bufio"
	"bytes"
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
func Stream(c *fiber.Ctx, reqBody []byte, url, method string) (string, error) {
	log.Printf("Proxying request to: %s", url)

	// Create a response content builder
	var responseContent strings.Builder
	var debugChunkCount int

	// Create the request
	req, err := http.NewRequestWithContext(c.Context(), method, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fiber.NewError(fiber.StatusInternalServerError, "Failed to create proxy request")
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Create HTTP client
	client := &http.Client{
		Timeout: 0, // No timeout for streaming
	}

	// Make the request to Ollama
	resp, err := client.Do(req)
	if err != nil {
		return "", fiber.NewError(fiber.StatusBadGateway, "Failed to contact Ollama: "+err.Error())
	}
	defer resp.Body.Close()

	// Handle error responses from Ollama
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Ollama error response: %s", string(body))
		return "", fiber.NewError(resp.StatusCode, "Ollama error: "+string(body))
	}

	// Set response headers for streaming
	c.Set("Content-Type", "application/json")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Status(resp.StatusCode)
	// Use a buffered channel to wait for streaming to complete
	done := make(chan bool)

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer close(done)       // Signal that streaming is complete
		defer resp.Body.Close() // Ensure connection is closed when done
		buffer := make([]byte, 1024)
		lastFlushTime := time.Now()
		log.Printf("Starting to stream chat response")

		// Read the stream chunk by chunk
		for {
			// Read a chunk from the response - each JSON object is on a separate line
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := buffer[:n]

				log.Printf("DEBUG: Read chunk %v", string(data))
				// Check if the chunk is empty
				if err != nil {
					if err != io.EOF {
						log.Printf("Error reading from Ollama: %v", err)
					}
					break
				}
				// Extract content from the JSON chunk
				// content := extractContent(data)
				// if content != "" {
				// 	responseContent.WriteString(content)
				// 	debugChunkCount++
				// }

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

		log.Printf("DEBUG: Final accumulated content size: %d bytes, chunk count: %d",
			responseContent.Len(), debugChunkCount)
	})

	// Wait for streaming to complete before returning
	<-done

	return responseContent.String(), nil
}

// extractContent parses Ollama message content from the response
func extractContent(data []byte) string {
	// Try to parse the JSON data
	log.Printf("DEBUG: Parsing JSON chunk: %s", string(data))
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
