package models

import (
	"time"
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
	ErrorMessage   string             `json:"error_message,omitempty"`
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
	QuestionID         int                `json:"question_id"`
	Status             ResearchTaskStatus `json:"status"`
	GatheredInfo       []string           `json:"gathered_info,omitempty"`
	InformationSources []string           `json:"information_sources,omitempty"`
	SynthesizedAnswer  string             `json:"synthesized_answer,omitempty"`
	ErrorMessage       string             `json:"error_message,omitempty"`
}

// ResearchQuestionResult represents the result of processing a sub-question
type ResearchQuestionResult struct {
	ID                int    `json:"id"`
	Question          string `json:"question"`
	SynthesizedAnswer string `json:"synthesized_answer,omitempty"`
	Error             error  `json:"-"`
	ErrorMessage      string `json:"error_message,omitempty"`
}
