package models

import (
	"encoding/json"
	"fmt"
)

type OllamaResponse interface {
	IsDone() bool
	GetChunkContent() string
	UnmarshalJSON(data []byte) error
}

// OllamaChatResp represents a response from the Ollama API
type OllamaChatResp struct {
	Model              string            `json:"model"`
	CreatedAt          string            `json:"created_at"`
	Message            OllamaChatMessage `json:"message"`
	Done               bool              `json:"done"`
	DoneReason         string            `json:"done_reason"`
	TotalDuration      float64           `json:"total_duration"`
	LoadDuration       float64           `json:"load_duration"`
	PromptEvalCount    int               `json:"prompt_eval_count"`
	PromptEvalDuration float64           `json:"prompt_eval_duration"`
	EvalCount          int               `json:"eval_count"`
	EvalDuration       float64           `json:"eval_duration"`
}

func (r *OllamaChatResp) UnmarshalJSON(data []byte) error {
	type Alias OllamaChatResp
	aux := (*Alias)(r)
	return json.Unmarshal(data, aux)
}

func (r *OllamaChatResp) GetChunkContent() string {
	if r.Message.Content != "" {
		return r.Message.Content
	}
	return ""
}

func (r *OllamaChatResp) IsDone() bool {
	return r.Done
}

// OllamaGenerateResponse represents the response from Ollama's generate API
type OllamaGenerateResponse struct {
	Model              string  `json:"model"`
	CreatedAt          string  `json:"created_at"`
	Response           string  `json:"response"`
	Done               bool    `json:"done"`
	DoneReason         string  `json:"done_reason"`
	Context            []int   `json:"context,omitempty"` // context IDs for the response
	PromptEvalCount    int     `json:"prompt_eval_count"`
	PromptEvalDuration float64 `json:"prompt_eval_duration"`
	EvalCount          int     `json:"eval_count"`
	EvalDuration       float64 `json:"eval_duration"`
	TotalDuration      float64 `json:"total_duration"`
	LoadDuration       float64 `json:"load_duration"`
}

func (r *OllamaGenerateResponse) UnmarshalJSON(data []byte) error {
	type Alias OllamaGenerateResponse
	aux := (*Alias)(r)
	return json.Unmarshal(data, aux)
}
func (r *OllamaGenerateResponse) GetChunkContent() string {
	if r.Response != "" {
		return r.Response
	}
	return ""
}
func (r *OllamaGenerateResponse) IsDone() bool {
	return r.Done
}

// OllamaEmbeddingResponse represents the response from Ollama's embeddings API
type OllamaEmbeddingResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float32 `json:"embeddings"`
	TotalDuration   int         `json:"total_duration"`
	LoadDuration    int         `json:"load_duration"`
	PromptEvalCount int         `json:"prompt_eval_count"`
}

func (r *OllamaEmbeddingResponse) UnmarshalJSON(data []byte) error {
	type Alias OllamaEmbeddingResponse
	aux := (*Alias)(r)
	return json.Unmarshal(data, aux)
}
func (r *OllamaEmbeddingResponse) GetChunkContent() string {
	if len(r.Embeddings) > 0 && len(r.Embeddings[0]) > 0 {
		// Convert the first embedding to a string representation
		return fmt.Sprintf("%v", r.Embeddings[0])
	}
	return ""
}
func (r *OllamaEmbeddingResponse) IsDone() bool {
	return len(r.Embeddings) > 0
}
