// Package models defines data structures used across the proxyllama application
package models

import (
	"time"

	"github.com/google/uuid"
)

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
	Format         any             `json:"format"`
	ConversationId *int            `json:"conversationId,omitempty"` // ui sends camelCase
}

// OllamaResp represents a response from the Ollama API
type OllamaResp struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            OllamaMessage `json:"message"`
	Done               bool          `json:"done"`
	TotalDuration      float64       `json:"total_duration"`
	LoadDuration       float64       `json:"load_duration"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	PromptEvalDuration float64       `json:"prompt_eval_duration"`
	EvalCount          int           `json:"eval_count"`
	EvalDuration       float64       `json:"eval_duration"`
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

type ModelParameters struct {
	NumCtx        int     `json:"num_ctx,omitempty"`
	RepeatLastN   int     `json:"repeat_last_n,omitempty"`
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	Seed          int     `json:"seed,omitempty"`
	Stop          string  `json:"stop,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	TopP          float64 `json:"top_p,omitempty"`
	MinP          float64 `json:"min_p,omitempty"`
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
	ID             string             `json:"id"`
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
	TaskID             string             `json:"task_id"`
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
