package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"proxyllama/config"
	"time"
)

// OllamaEmbeddingResponse represents the response from Ollama's embeddings API
type OllamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
	Error     string    `json:"error,omitempty"`
}

// GetEmbedding retrieves a vector embedding for the provided text from Ollama
func GetEmbedding(ctx context.Context, textToEmbed string, modelName string) ([]float32, error) {
	// Ensure modelName is an embedding model or one that supports it
	conf := config.GetConfig()
	url := conf.Ollama.BaseURL + "/api/embeddings" // Make sure this is the correct endpoint

	requestPayload := map[string]string{
		"model":  modelName, // This should be a valid embedding model loaded in Ollama
		"prompt": textToEmbed,
	}
	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second} // Adjust timeout as needed
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send embedding request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama embedding request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var embeddingResponse OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResponse); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if embeddingResponse.Error != "" {
		return nil, fmt.Errorf("Ollama embedding error: %s", embeddingResponse.Error)
	}

	return embeddingResponse.Embedding, nil
}
