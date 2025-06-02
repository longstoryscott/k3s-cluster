package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"proxyllama/config"
	"proxyllama/models"
	"testing"
)

func Test_ChatRequest(t *testing.T) {
	confFile := "testdata/.config.yaml"
	config.GetConfig(&confFile)
	// Create a sample chat request
	req := models.OllamaChatReq{
		Model: config.DefaultPrimaryProfile.ModelName,
		Messages: []models.OllamaChatMessage{
			{Role: "user", Content: "Why is the sky blue?"},
		},
	}

	// Marshal the request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal chat request: %v", err)
	}

	handler, status, err := GetProxyHandler(context.Background(), data, "/api/chat", "POST", true, func() *models.OllamaChatResp { return &models.OllamaChatResp{} })
	if err != nil {
		t.Fatalf("Failed to get proxy handler: %v", err)
	}
	if status != 200 {
		t.Fatalf("Expected status 200, got %d", status)
	}

	// Create a buffer to capture the response
	w := &bytes.Buffer{}
	wr := bufio.NewWriter(w)

	res, err := handler(wr)
	if err != nil {
		if res == "" {
			t.Fatalf("Chat request failed with empty response: %v", err)
		} else {
			t.Fatalf("connection closed, but response completed: %v", err)
		}
	}

	t.Log("Chat response:", res)
}

func Test_EmbeddingRequest(t *testing.T) {
	confFile := "testdata/.config.yaml"
	config.GetConfig(&confFile)
	// Create a sample embedding request
	req := models.OllamaEmbeddingReq{
		Model: config.DefaultEmbeddingProfile.ModelName,
		Input: []string{"Can I get and example of splitting a model accross multiple gpus in code?"},
	}

	// Marshal the request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal embedding request: %v", err)
	}

	handler, status, err := GetProxyHandler(context.Background(), data, "/api/embed", "POST", false, func() *models.OllamaEmbeddingResponse { return &models.OllamaEmbeddingResponse{} })
	if err != nil {
		t.Fatalf("Failed to get proxy handler: %v", err)
	}
	if status != 200 {
		t.Fatalf("Expected status 200, got %d", status)
	}

	// Create a buffer to capture the response
	w := &bytes.Buffer{}
	wr := bufio.NewWriter(w)

	res, err := handler(wr)
	if err != nil {
		t.Fatalf("Embedding request failed: %v", err)
	}

	t.Log("Embedding response:", res)

	var embeddingResponse models.OllamaEmbeddingResponse
	if err := json.Unmarshal([]byte(res), &embeddingResponse); err != nil {
		t.Fatalf("Failed to decode embedding response: %v", err)
	}
	if len(embeddingResponse.Embeddings) == 0 {
		t.Fatalf("Received empty embedding response")
	}
}

func Test_GenerateRequest(t *testing.T) {
	confFile := "testdata/.config.yaml"
	config.GetConfig(&confFile)
	// Create a sample generate request
	req := models.OllamaGenerateReq{
		Model:  config.DefaultPrimaryProfile.ModelName,
		Prompt: "Tell me a joke about llamas.",
	}

	// Marshal the request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal generate request: %v", err)
	}

	handler, status, err := GetProxyHandler(context.Background(), data, "/api/generate", "POST", true, func() *models.OllamaGenerateResponse { return &models.OllamaGenerateResponse{} })
	if err != nil {
		t.Fatalf("Failed to get proxy handler: %v", err)
	}
	if status != 200 {
		t.Fatalf("Expected status 200, got %d", status)
	}

	// Create a buffer to capture the response
	w := &bytes.Buffer{}
	wr := bufio.NewWriter(w)

	res, err := handler(wr)
	if err != nil {
		t.Fatalf("Generate request failed: %v", err)
	}

	if res == "" {
		t.Fatalf("Generate request returned empty response")
	}

	t.Log("Generate response:", res)
}
