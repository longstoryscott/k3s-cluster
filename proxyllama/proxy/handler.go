package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"proxyllama/config"
	"proxyllama/models"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// GetProxyHandler handles streaming responses from Ollama and collects the full assistant response
// It properly streams to the client while also returning the accumulated content
func GetProxyHandler(ctx context.Context, reqBody []byte, path, method string, stream bool) (func(w *bufio.Writer) (string, error), int, error) {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
		"url":  path,
	}).Info("Proxying request")
	conf := config.GetConfig()
	url := conf.Ollama.BaseURL + path

	// Create a response content builder
	var responseContent strings.Builder

	// Create the request
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fiber.StatusInternalServerError, err
	}

	// Copy headers from original request
	// This is important for Ollama to understand the request
	for key, value := range GetHeadersFromContext(ctx) {
		req.Header.Set(key, value)
	}

	// Create HTTP client
	client := createHTTPClient()

	// Make the request to Ollama
	resp, err := client.Do(req)
	if err != nil {
		return nil, fiber.StatusBadGateway, err
	}

	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("Cache-Control", "no-cache")
	resp.Header.Set("Connection", "keep-alive")
	if stream {
		resp.Header.Set("Transfer-Encoding", "chunked")
		resp.Header.Set("X-Accel-Buffering", "no")
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
		if stream {
			return streamHandler(w, resp, &responseContent)
		}
		return resHandler(w, resp)
	}, resp.StatusCode, nil
}

func streamHandler(w *bufio.Writer, resp *http.Response, responseContent *strings.Builder) (string, error) {
	defer resp.Body.Close() // Ensure connection is closed when done
	buffer := make([]byte, 1024)
	// lastFlushTime := time.Now()

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
			// if time.Since(lastFlushTime) > 10*time.Second {
			// 	if flushErr := w.Flush(); flushErr != nil {
			// 		_, file, line, _ := runtime.Caller(0)
			// 		logrus.WithFields(logrus.Fields{
			// 			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			// 			"line":  line,
			// 			"error": flushErr,
			// 		}).Error("Error flushing response")
			// 		break
			// 	}
			// 	lastFlushTime = time.Now()
			// } else

			if flushErr := w.Flush(); flushErr != nil {
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

func resHandler(w *bufio.Writer, resp *http.Response) (string, error) {
	defer resp.Body.Close() // Ensure connection is closed when done

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Starting to stream chat response")

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Error("Error reading from Ollama")
		return "", err
	}
	if len(res) == 0 {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": "empty response",
		}).Error("Error reading from Ollama")
		return "", err
	}
	// Write to response
	if _, writeErr := w.Write(res); writeErr != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": writeErr,
		}).Error("Error writing to response")
		return "", writeErr
	}
	// Flush frequently to avoid client timeouts
	if flushErr := w.Flush(); flushErr != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": flushErr,
		}).Error("Error flushing response")
		return "", flushErr
	}
	return string(res), nil
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

// createHTTPClient returns a configured HTTP client optimized for streaming responses
func createHTTPClient() *http.Client {
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
