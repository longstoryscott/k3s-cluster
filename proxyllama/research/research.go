package research

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"proxyllama/models"
	"proxyllama/storage"
)

// CreateResearchRequest is the request to create a new research task
type CreateResearchRequest struct {
	Query          string `json:"query"`
	Model          string `json:"model,omitempty"`
	ConversationID int    `json:"conversation_id,omitempty"`
}

// StartResearchTask starts a new research task
func StartResearchTask(ctx context.Context, userID, query, model string, conversationID *int) (*models.ResearchTask, error) {
	// Generate a unique ID for the task
	taskID := uuid.New().String()

	// Create the research task
	task := &models.ResearchTask{
		ID:             taskID,
		UserID:         userID,
		Query:          query,
		Model:          model,
		Status:         models.ResearchTaskStatusPending,
		ConversationID: conversationID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Save the task to the database
	err := storage.SaveResearchTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to save research task: %w", err)
	}

	// Start the research process asynchronously using the deep research orchestrator
	go PerformDeepResearch(context.Background(), taskID, userID, query, model, conversationID)

	return task, nil
}

// GetResearchTask gets a research task by ID
func GetResearchTask(ctx context.Context, taskID, userID string) (*models.ResearchTask, error) {
	return storage.GetResearchTaskByUserID(ctx, taskID, userID)
}

// GetUserResearchTasks gets all research tasks for a user
func GetUserResearchTasks(ctx context.Context, userID string) ([]*models.ResearchTask, error) {
	return storage.GetUserResearchTasks(ctx, userID)
}

// CancelResearchTask cancels a research task
func CancelResearchTask(ctx context.Context, taskID, userID string) error {
	// Get the task first to verify ownership
	task, err := storage.GetResearchTaskByUserID(ctx, taskID, userID)
	if err != nil {
		return fmt.Errorf("failed to get research task: %w", err)
	}

	// Only allow cancellation if the task is not already completed or failed
	if task.Status == models.ResearchTaskStatusCompleted || task.Status == models.ResearchTaskStatusFailed {
		return fmt.Errorf("cannot cancel a task that is already %s", task.Status)
	}

	// Update the status to canceled
	task.Status = models.ResearchTaskStatusCanceled
	task.UpdatedAt = time.Now()

	return storage.UpdateResearchTask(ctx, task)
}
