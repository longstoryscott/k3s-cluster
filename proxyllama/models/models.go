// Package context provides conversation context management for the proxy
package models

// Message represents a single exchange in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	ID      int    `json:"-"` // Internal use only, not sent to LLM
}

// Summary represents a consolidated summary of messages or other summaries
type Summary struct {
	Content string `json:"content"`
	Level   int    `json:"-"` // Internal use only, not sent to LLM
	ID      int    `json:"-"` // Internal use only, not sent to LLM
}

// OllamaMessage represents a message in the format expected by Ollama API
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaReq represents a request to the Ollama API
type OllamaReq struct {
	Model          string          `json:"model"`
	Messages       []OllamaMessage `json:"messages"`
	Stream         bool            `json:"stream"`
	ConversationId *int            `json:"conversationId,omitempty"` // ui sends camelCase
}

// ChunkData represents the structure of a streaming chunk response
type ChunkData struct {
	Message            OllamaMessage `json:"message"`
	Done               bool          `json:"done"`
	DoneReason         string        `json:"done_reason"`
	TotalDuration      float64       `json:"total_duration"`
	LoadDuration       float64       `json:"load_duration"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	PromptEvalDuration float64       `json:"prompt_eval_duration"`
	EvalCount          int           `json:"eval_count"`
	EvalDuration       float64       `json:"eval_duration"`
}
