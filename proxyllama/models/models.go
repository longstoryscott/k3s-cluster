// Package models defines data structures used across the proxyllama application
package models

import (
	"time"

	"github.com/google/uuid"
)

// Conversation represents a conversation between a user and the LLM
type Conversation struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message represents a single exchange in the conversation
type Message struct {
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	Images         []string  `json:"images,omitempty"`     // Optional images associated with the message
	ToolCalls      []any     `json:"tool_calls,omitempty"` // Optional tool calls associated with the message
	ID             int       `json:"-"`                    // Internal use only, not sent to LLM
	CreatedAt      time.Time `json:"-"`                    // Timestamp of when the message was created
	ConversationID int       `json:"-"`                    // ID of the conversation this message belongs to
}

// Summary represents a consolidated summary of messages or other summaries
type Summary struct {
	Content        string    `json:"content"`
	Level          int       `json:"level"`
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"` // ID of the conversation this summary belongs to
	SourceIDs      []int     `json:"source_ids"`
	CreatedAt      time.Time `json:"created_at"`
}

// OllamaChatMessage represents a message in the format expected by Ollama API
type OllamaChatMessage struct {
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	Images    []string `json:"images,omitempty"`     // Optional images associated with the message
	ToolCalls []any    `json:"tool_calls,omitempty"` // Optional tool calls associated with the message
}

// OllamaChatReq represents a request to the Ollama API
type OllamaChatReq struct {
	Model          *string             `json:"model,omitempty"`          // model name, used internally only to set the model name in the request to ollama based on the profile
	Messages       []OllamaChatMessage `json:"messages"`                 // messages to send to the model, each message is a struct with role and content
	Stream         bool                `json:"stream"`                   // if true, the response will be streamed back as a series of events
	Format         any                 `json:"format"`                   // the format to return a response in. Format can be json or a JSON schema
	ConversationId *int                `json:"conversationId,omitempty"` // ui sends camelCase
	KeepAlive      string              `json:"keep_alive,omitempty"`     // controls how long the model will stay loaded into memory
	Options        map[string]any      `json:"options,omitempty"`        // additional model parameters listed in the documentation for the Modelfile such as temperature
	Tools          []any               `json:"tools,omitempty"`          // tools to use for the request, if any
}

// OllamaGenerateReq represents a request to the Ollama embeddings API
type OllamaGenerateReq struct {
	Model     string         `json:"model"`
	Prompt    string         `json:"prompt"`
	Suffix    string         `json:"suffix,omitempty"`
	Images    []string       `json:"images,omitempty"`
	Format    any            `json:"format,omitempty"`     // the format to return a response in. Format can be json or a JSON schema
	Options   map[string]any `json:"options,omitempty"`    // additional model parameters listed in the documentation for the Modelfile such as temperature
	System    string         `json:"system,omitempty"`     // system message to (overrides what is defined in the Modelfile)
	Template  string         `json:"template,omitempty"`   // the prompt template to use (overrides what is defined in the Modelfile)
	Stream    bool           `json:"stream,omitempty"`     // if false the response will be returned as a single response object
	Raw       bool           `json:"raw,omitempty"`        // if true no formatting will be applied to the prompt
	KeepAlive string         `json:"keep_alive,omitempty"` // controls how long the model will stay loaded into memory
	Context   string         `json:"context,omitempty"`    // (deprecated): the context parameter returned from a previous request
}

// OllamaEmbeddingReq represents a request to the Ollama embeddings API
type OllamaEmbeddingReq struct {
	Model     string         `json:"model"`
	Input     []string       `json:"input"`
	Truncate  bool           `json:"truncate,omitempty"`   // whether to truncate the input to the model's maximum length
	Options   map[string]any `json:"options,omitempty"`    // additional model parameters listed in the documentation for the Modelfile such as temperature
	KeepAlive string         `json:"keep_alive,omitempty"` // controls how long the model will stay loaded into memory
}

type ModelParameters struct {
	NumCtx        int      `json:"num_ctx,omitempty"`
	RepeatLastN   int      `json:"repeat_last_n,omitempty"`
	RepeatPenalty float64  `json:"repeat_penalty,omitempty"`
	Temperature   float64  `json:"temperature,omitempty"`
	Seed          int      `json:"seed,omitempty"`
	Stop          []string `json:"stop,omitempty"`
	NumPredict    int      `json:"num_predict,omitempty"`
	TopK          int      `json:"top_k,omitempty"`
	TopP          float64  `json:"top_p,omitempty"`
	MinP          float64  `json:"min_p,omitempty"`
}

func (p *ModelParameters) ToMap() map[string]any {
	return map[string]any{
		"num_ctx":        p.NumCtx,
		"repeat_last_n":  p.RepeatLastN,
		"repeat_penalty": p.RepeatPenalty,
		"temperature":    p.Temperature,
		"seed":           p.Seed,
		"stop":           p.Stop,
		"num_predict":    p.NumPredict,
		"top_k":          p.TopK,
		"top_p":          p.TopP,
		"min_p":          p.MinP,
	}
}

type ModelProfile struct {
	ID           uuid.UUID        `json:"id"`
	UserID       string           `json:"userId"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	ModelName    string           `json:"modelName"`
	Parameters   ModelParameters  `json:"parameters"`
	SystemPrompt string           `json:"systemPrompt"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
	ModelVersion string           `json:"modelVersion"`
	Type         ModelProfileType `json:"type"`
}

type ModelProfileType int

const (
	// ModelProfileTypePrimary represents the primary model profile type
	ModelProfileTypePrimary ModelProfileType = iota
	// ModelProfileTypePrimarySummary represents a primary summary model profile type
	ModelProfileTypePrimarySummary
	// ModelProfileMasterSummary represents a master summary model profile type
	ModelProfileTypeMasterSummary
	// ModelProfileTypeBriefSummary represents a brief summary model profile type
	ModelProfileTypeBriefSummary
	// ModelProfileTypeKeyPoints represents a key points model profile type
	ModelProfileTypeKeyPoints
	// ModelProfileTypeSelfCritique represents a self critique model profile type
	ModelProfileTypeSelfCritique
	// ModelProfileTypeImprovement represents an improvement model profile type
	ModelProfileTypeImprovement
	// ModelProfileTypeMemoryRetrieval represents a memory retrieval model profile type
	ModelProfileTypeMemoryRetrieval
	// ModelProfileTypeAnalysis represents an analysis model profile type
	ModelProfileTypeAnalysis
	// ModelProfileTypeResearchTask represents a research model profile type
	ModelProfileTypeResearchTask
	// ModelProfileTypeResearchPlan represents a research planning model profile type
	ModelProfileTypeResearchPlan
	// ModelProfileTypeResearchConsolidation represents a research consolidation model profile type
	ModelProfileTypeResearchConsolidation
	// ModelProfileTypeResearchAnalysis represents a research analysis model profile type
	ModelProfileTypeResearchAnalysis
	// ModelProfileTypeEmbedding represents an embedding model profile type
	ModelProfileTypeEmbedding
	// ModelProfileTypeFormatting represents a format model profile type
	ModelProfileTypeFormatting
)

// ResearchTaskStatus represents the current state of a research task
type ResearchTaskStatus string

const (
	ResearchTaskStatusPending      ResearchTaskStatus = "PENDING"      // Task has been created but not yet started
	ResearchTaskStatusPlanning     ResearchTaskStatus = "PLANNING"     // Task is in planning phase
	ResearchTaskStatusGathering    ResearchTaskStatus = "GATHERING"    // Task is gathering information
	ResearchTaskStatusProcessing   ResearchTaskStatus = "PROCESSING"   // Task is processing gathered information
	ResearchTaskStatusSynthesizing ResearchTaskStatus = "SYNTHESIZING" // Task is synthesizing findings
	ResearchTaskStatusCompleted    ResearchTaskStatus = "COMPLETED"    // Task has been completed successfully
	ResearchTaskStatusFailed       ResearchTaskStatus = "FAILED"       // Task has failed
	ResearchTaskStatusCanceled     ResearchTaskStatus = "CANCELED"     // Task was canceled by the user
)

// ResearchTask represents a deep research task
type ResearchTask struct {
	ID             int                `json:"id"`
	UserID         string             `json:"user_id"`
	Query          string             `json:"query"`
	Model          string             `json:"model"`
	ConversationID *int               `json:"conversation_id,omitempty"`
	Status         ResearchTaskStatus `json:"status"`
	ErrorMessage   *string            `json:"error_message,omitempty"`
	Plan           *ResearchPlan      `json:"plan,omitempty"`
	Results        []string           `json:"results,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	CompletedAt    *time.Time         `json:"completed_at,omitempty"`
}

// ResearchPlan represents the plan for executing a research task
type ResearchPlan struct {
	MainIntent   string             `json:"main_intent"`
	SubQuestions []ResearchQuestion `json:"sub_questions"`
	RawPlan      string             `json:"raw_plan,omitempty"` // Original plan text from LLM
}

// ResearchQuestion represents a sub-question in a research plan
type ResearchQuestion struct {
	ID       int      `json:"id"`
	Question string   `json:"question"`
	Keywords []string `json:"keywords"`
}

// CreateResearchRequest is the request to create a new research task
type CreateResearchRequest struct {
	Query          string `json:"query"`
	Model          string `json:"model,omitempty"`
	ConversationID int    `json:"conversation_id,omitempty"`
}

// ResearchSubtask represents a subtask within a research task
type ResearchSubtask struct {
	ID                 int                `json:"id"`
	TaskID             int                `json:"task_id"`
	QuestionID         int                `json:"question_id"`
	Status             ResearchTaskStatus `json:"status"`
	GatheredInfo       []string           `json:"gathered_info"`
	InformationSources []string           `json:"information_sources"`
	SynthesizedAnswer  *string            `json:"synthesized_answer,omitempty"`
	ErrorMessage       *string            `json:"error_message,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// ResearchQuestionResult represents the result of processing a sub-question
type ResearchQuestionResult struct {
	ID                int    `json:"id"`
	Question          string `json:"question"`
	SynthesizedAnswer string `json:"synthesized_answer,omitempty"`
	Error             error  `json:"-"`
	ErrorMessage      string `json:"error_message,omitempty"`
}

// SearchResultContent represents a content item from a web search result
type SearchResultContent struct {
	URL     string `json:"url"`     // URL of the content
	Title   string `json:"title"`   // Title of the content
	Content string `json:"content"` // Snippet or summary of the content
}

// SearchResult represents a search result from a web query
type SearchResult struct {
	IsFromUrlInUserQuery bool                  `json:"is_from_url_in_user_query"` // Indicates if this result is from a user query
	Query                string                `json:"query"`
	Contents             []SearchResultContent `json:"contents,omitempty"`
	Error                string                `json:"error,omitempty"`
}

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Uid       string    `json:"uid"`
	CreatedAt time.Time `json:"created_at"`
}

type MemorySource string

const (
	MemorySourceSummary MemorySource = "summary" // Memory type for conversation messages
	MemorySourceMessage MemorySource = "message" // Memory type for document content
)

// MemoryFragment represents a content item that is similar to a user's query
type MemoryFragment struct {
	ID      int    `json:"id"`
	Role    string `json:"role"`    // Role of the message (e.g., "user", "assistant")
	Content string `json:"content"` // Content of the message
}

// Memory represents a grouped memory for a user, which can be a summary or a question-answer pair
type Memory struct {
	Fragments      []MemoryFragment `json:"fragments"`
	Source         MemorySource     `json:"source"`
	CreatedAt      time.Time        `json:"created_at"`
	Similarity     float32          `json:"similarity"`                // Used for vector similarity search results
	SourceID       int              `json:"source_id"`                 // ID of the source document or conversation
	ConversationID int              `json:"conversation_id,omitempty"` // ID of the conversation this memory belongs to
}
