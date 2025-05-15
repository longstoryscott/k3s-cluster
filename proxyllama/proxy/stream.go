package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"proxyllama/models"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// Stream handles streaming responses from Ollama and collects the full assistant response
// It properly streams to the client while also returning the accumulated content
func Stream(ctx context.Context, reqBody []byte, url, method string) (func(w *bufio.Writer) (string, error), int, error) {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
		"url":  url,
	}).Info("Proxying request")

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
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":       filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":       line,
			"statusCode": resp.StatusCode,
		}).Error("Ollama error response: ", string(body))
		return nil, resp.StatusCode, fiber.NewError(resp.StatusCode, string(body))
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

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Starting to stream chat response")

	// Read the stream chunk by chunk
	for {
		// Read a chunk from the response - each JSON object is on a separate line
		n, err := resp.Body.Read(buffer)
		if err != nil {
			if err != io.EOF {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"error": err,
				}).Error("Error reading from Ollama")
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
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"error": writeErr,
				}).Error("Error writing to response")
				break
			}

			// Flush frequently to avoid client timeouts
			// Always flush at least every 10 seconds even if no data
			if time.Since(lastFlushTime) > 10*time.Second {
				if flushErr := w.Flush(); flushErr != nil {
					_, file, line, _ := runtime.Caller(0)
					logrus.WithFields(logrus.Fields{
						"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
						"line":  line,
						"error": flushErr,
					}).Error("Error flushing response")
					break
				}
				lastFlushTime = time.Now()
			} else if flushErr := w.Flush(); flushErr != nil {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"error": flushErr,
				}).Error("Error flushing response")
				break
			}

			// Check if this is the last chunk (done=true)
			if isDone(data) {
				break
			}
		}
	}
	return responseContent.String(), nil
}

// extractContent parses Ollama message content from the response
func extractContent(data []byte) string {
	// Try to parse the JSON data
	var chunkData models.ChunkData
	if err := json.Unmarshal(data, &chunkData); err == nil {
		return chunkData.Message.Content
	}
	return ""
}

// isDone checks if a chunk indicates the end of the stream
func isDone(data []byte) bool {
	var chunkData models.ChunkData
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
