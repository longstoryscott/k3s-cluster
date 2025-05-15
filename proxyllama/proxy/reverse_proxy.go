package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"proxyllama/config"
	"proxyllama/models"
	"runtime"

	"github.com/sirupsen/logrus"
)

// SendOllamaRequest sends a request to the Ollama API and handles streaming the response
func SendOllamaRequest(ctx context.Context, model string, messages []models.OllamaMessage, endpoint string) (string, error) {
	conf := config.GetConfig()
	url := conf.Ollama.BaseURL + endpoint

	requestBody := models.OllamaReq{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":         filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":         line,
		"url":          url,
		"messageCount": len(messages),
	}).Info("Sending request to Ollama")

	handler, _, err := Stream(ctx, reqBody, url, http.MethodPost)
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
