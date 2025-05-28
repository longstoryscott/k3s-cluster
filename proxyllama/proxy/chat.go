package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"proxyllama/models"
	"proxyllama/util"
)

// StreamOllamaChatRequest sends a request to the Ollama API and handles streaming the response
func StreamOllamaChatRequest(ctx context.Context, model string, messages []models.OllamaChatMessage) (string, error) {
	requestBody := models.OllamaChatReq{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	util.LogInfo("Sending request to Ollama")

	handler, _, err := GetProxyHandler(ctx, reqBody, "/api/chat", http.MethodPost, true, func() *models.OllamaChatResp {
		return &models.OllamaChatResp{}
	})
	if err != nil {
		return "", fmt.Errorf("error streaming request: %w", err)
	}

	w := &bytes.Buffer{}
	wr := bufio.NewWriter(w)

	res, err := handler(wr)
	if err != nil {
		return "", fmt.Errorf("error handling response: %w", err)
	}

	if err := wr.Flush(); err != nil {
		return "", fmt.Errorf("error flushing response: %w", err)
	}

	// Check if the response is empty
	if res == "" {
		return "", fmt.Errorf("empty response from Ollama")
	}

	return res, nil
}
