package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"proxyllama/models"
)

// Schema SQL statements
const (
	sqlCreateResearchTasksTable = `
		CREATE TABLE IF NOT EXISTS research_tasks (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			query TEXT NOT NULL,
			model TEXT NOT NULL,
			conversation_id INTEGER,
			status TEXT NOT NULL,
			error_message TEXT,
			plan JSONB,
			results JSONB,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			completed_at TIMESTAMPTZ
		)
	`
	sqlCreateSubtasksTable = `
		CREATE TABLE IF NOT EXISTS research_subtasks (
			id SERIAL PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES research_tasks(id) ON DELETE CASCADE,
			question_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			gathered_info JSONB,
			information_sources JSONB,
			synthesized_answer TEXT,
			error_message TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(task_id, question_id)
		)
	`
)

// SQL queries for research operations
const (
	sqlSaveResearchTask = `
		INSERT INTO research_tasks 
		(id, user_id, query, model, conversation_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	sqlUpdateResearchTaskStatus = `
		UPDATE research_tasks 
		SET status = $2, error_message = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	sqlUpdateResearchTask = `
		UPDATE research_tasks 
		SET status = $2, error_message = $3, updated_at = NOW(),
		    completed_at = CASE WHEN $2 IN ('COMPLETED', 'FAILED', 'CANCELED') THEN NOW() ELSE completed_at END
		WHERE id = $1
		RETURNING updated_at
	`
	sqlStoreResearchPlan = `
		UPDATE research_tasks 
		SET plan = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	sqlStoreFinalResearchResult = `
		UPDATE research_tasks 
		SET results = jsonb_build_array($2), updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	sqlCreateResearchSubtask = `
		INSERT INTO research_subtasks 
		(task_id, question_id, status)
		VALUES ($1, $2, 'PENDING')
		RETURNING id
	`
	sqlUpdateSubtaskStatus = `
		UPDATE research_subtasks 
		SET status = $3, error_message = $4, updated_at = NOW()
		WHERE task_id = $1 AND question_id = $2
		RETURNING updated_at
	`
	sqlStoreSubtaskGatheredInfo = `
		UPDATE research_subtasks 
		SET gathered_info = $3, information_sources = $4, updated_at = NOW()
		WHERE task_id = $1 AND question_id = $2
		RETURNING updated_at
	`
	sqlStoreSubtaskResult = `
		UPDATE research_subtasks 
		SET synthesized_answer = $3, status = 'COMPLETED', updated_at = NOW()
		WHERE task_id = $1 AND question_id = $2
		RETURNING updated_at
	`
	sqlGetResearchTask = `
		SELECT id, user_id, query, model, conversation_id, status, error_message, 
		       plan, results, created_at, updated_at, completed_at
		FROM research_tasks
		WHERE id = $1
	`
	sqlGetResearchTaskByUserID = `
		SELECT id, user_id, query, model, conversation_id, status, error_message, 
		       plan, results, created_at, updated_at, completed_at
		FROM research_tasks
		WHERE id = $1 AND user_id = $2
	`
	sqlGetUserResearchTasks = `
		SELECT id, user_id, query, model, conversation_id, status, error_message, 
		       plan, results, created_at, updated_at, completed_at
		FROM research_tasks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
)

// InitResearchSchema initializes the research database schema
func InitResearchSchema(ctx context.Context) {
	log.Println("Initializing research database schema...")

	// Create research_tasks table
	if _, err := Pool.Exec(ctx, sqlCreateResearchTasksTable); err != nil {
		log.Fatalf("Failed to create research_tasks table: %v", err)
	}

	// Create research_subtasks table
	if _, err := Pool.Exec(ctx, sqlCreateSubtasksTable); err != nil {
		log.Fatalf("Failed to create research_subtasks table: %v", err)
	}

	// Create indexes
	if _, err := Pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_research_tasks_user_id ON research_tasks(user_id);
		CREATE INDEX IF NOT EXISTS idx_research_subtasks_task_id ON research_subtasks(task_id);
	`); err != nil {
		log.Printf("Warning: Error creating research indexes: %v", err)
	}

	log.Println("Research database schema initialized successfully")
}

// SaveResearchTask saves a new research task
func SaveResearchTask(ctx context.Context, task *models.ResearchTask) error {
	_, err := Pool.Exec(ctx, sqlSaveResearchTask,
		task.ID, task.UserID, task.Query, task.Model, task.ConversationID,
		task.Status, task.CreatedAt, task.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save research task: %w", err)
	}
	return nil
}

// UpdateResearchTaskStatus updates the status of a research task
func UpdateResearchTaskStatus(ctx context.Context, taskID string, status string, errorMsg *string) error {
	var updatedAt time.Time
	err := Pool.QueryRow(ctx, sqlUpdateResearchTaskStatus,
		taskID, status, errorMsg).Scan(&updatedAt)

	if err != nil {
		return fmt.Errorf("failed to update research task status: %w", err)
	}
	return nil
}

// UpdateResearchTask updates a research task
func UpdateResearchTask(ctx context.Context, task *models.ResearchTask) error {
	var updatedAt time.Time
	err := Pool.QueryRow(ctx, sqlUpdateResearchTask,
		task.ID, task.Status, task.ErrorMessage).Scan(&updatedAt)

	if err != nil {
		return fmt.Errorf("failed to update research task: %w", err)
	}

	// Update the task's updated_at field with the value from the database
	task.UpdatedAt = updatedAt
	return nil
}

// StoreResearchPlan stores the plan for a research task
func StoreResearchPlan(ctx context.Context, taskID string, plan *models.ResearchPlan) error {
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal research plan: %w", err)
	}

	var updatedAt time.Time
	err = Pool.QueryRow(ctx, sqlStoreResearchPlan, taskID, planJSON).Scan(&updatedAt)
	if err != nil {
		return fmt.Errorf("failed to store research plan: %w", err)
	}

	return nil
}

// StoreFinalResearchResult stores the final result of a research task
func StoreFinalResearchResult(ctx context.Context, taskID string, result string) error {
	var updatedAt time.Time
	err := Pool.QueryRow(ctx, sqlStoreFinalResearchResult, taskID, result).Scan(&updatedAt)
	if err != nil {
		return fmt.Errorf("failed to store final research result: %w", err)
	}

	return nil
}

// CreateResearchSubtask creates a subtask for a research task
func CreateResearchSubtask(ctx context.Context, taskID string, questionID int, question string) (int, error) {
	var subtaskID int
	err := Pool.QueryRow(ctx, sqlCreateResearchSubtask, taskID, questionID).Scan(&subtaskID)
	if err != nil {
		return 0, fmt.Errorf("failed to create research subtask: %w", err)
	}

	return subtaskID, nil
}

// UpdateSubtaskStatus updates the status of a research subtask
func UpdateSubtaskStatus(ctx context.Context, taskID string, questionID int, status string, errorMsg *string) error {
	var updatedAt time.Time
	err := Pool.QueryRow(ctx, sqlUpdateSubtaskStatus, taskID, questionID, status, errorMsg).Scan(&updatedAt)
	if err != nil {
		return fmt.Errorf("failed to update subtask status: %w", err)
	}

	return nil
}

// StoreSubtaskGatheredInfo stores gathered information for a subtask
func StoreSubtaskGatheredInfo(ctx context.Context, taskID string, questionID int, info []string, sources []string) error {
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal gathered info: %w", err)
	}

	sourcesJSON, err := json.Marshal(sources)
	if err != nil {
		return fmt.Errorf("failed to marshal information sources: %w", err)
	}

	var updatedAt time.Time
	err = Pool.QueryRow(ctx, sqlStoreSubtaskGatheredInfo, taskID, questionID, infoJSON, sourcesJSON).Scan(&updatedAt)
	if err != nil {
		return fmt.Errorf("failed to store subtask gathered info: %w", err)
	}

	return nil
}

// StoreSubtaskResult stores the result of a subtask
func StoreSubtaskResult(ctx context.Context, taskID string, questionID int, result string) error {
	var updatedAt time.Time
	err := Pool.QueryRow(ctx, sqlStoreSubtaskResult, taskID, questionID, result).Scan(&updatedAt)
	if err != nil {
		return fmt.Errorf("failed to store subtask result: %w", err)
	}

	return nil
}

// GetResearchTask gets a research task by ID
func GetResearchTask(ctx context.Context, taskID string) (*models.ResearchTask, error) {
	task, err := scanResearchTask(Pool.QueryRow(ctx, sqlGetResearchTask, taskID))
	if err != nil {
		return nil, fmt.Errorf("failed to get research task: %w", err)
	}

	return task, nil
}

// GetResearchTaskByUserID gets a research task by ID and user ID
func GetResearchTaskByUserID(ctx context.Context, taskID, userID string) (*models.ResearchTask, error) {
	task, err := scanResearchTask(Pool.QueryRow(ctx, sqlGetResearchTaskByUserID, taskID, userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get research task: %w", err)
	}

	return task, nil
}

// GetUserResearchTasks gets all research tasks for a user
func GetUserResearchTasks(ctx context.Context, userID string) ([]*models.ResearchTask, error) {
	rows, err := Pool.Query(ctx, sqlGetUserResearchTasks, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user research tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ResearchTask
	for rows.Next() {
		task := &models.ResearchTask{}
		var plan, results []byte
		var conversationID *int
		var completedAt *time.Time

		err := rows.Scan(
			&task.ID, &task.UserID, &task.Query, &task.Model, &conversationID,
			&task.Status, &task.ErrorMessage, &plan, &results,
			&task.CreatedAt, &task.UpdatedAt, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan research task row: %w", err)
		}

		task.ConversationID = conversationID
		task.CompletedAt = completedAt

		// Parse plan if available
		if len(plan) > 0 {
			var researchPlan models.ResearchPlan
			if err := json.Unmarshal(plan, &researchPlan); err != nil {
				log.Printf("Warning: Failed to parse research plan for task %s: %v", task.ID, err)
			} else {
				task.Plan = &researchPlan
			}
		}

		// Parse results if available
		if len(results) > 0 {
			var resultsList []string
			if err := json.Unmarshal(results, &resultsList); err != nil {
				log.Printf("Warning: Failed to parse research results for task %s: %v", task.ID, err)
			} else {
				task.Results = resultsList
			}
		}

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating research task rows: %w", err)
	}

	return tasks, nil
}

// Helper function to scan a research task from a row
func scanResearchTask(row interface{}) (*models.ResearchTask, error) {
	var scanner interface {
		Scan(dest ...interface{}) error
	}

	switch r := row.(type) {
	case interface {
		Scan(dest ...interface{}) error
	}:
		scanner = r
	default:
		return nil, fmt.Errorf("unsupported row type: %T", row)
	}

	task := &models.ResearchTask{}
	var plan, results []byte
	var conversationID *int
	var completedAt *time.Time

	err := scanner.Scan(
		&task.ID, &task.UserID, &task.Query, &task.Model, &conversationID,
		&task.Status, &task.ErrorMessage, &plan, &results,
		&task.CreatedAt, &task.UpdatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	task.ConversationID = conversationID
	task.CompletedAt = completedAt

	// Parse plan if available
	if len(plan) > 0 {
		var researchPlan models.ResearchPlan
		if err := json.Unmarshal(plan, &researchPlan); err != nil {
			log.Printf("Warning: Failed to parse research plan: %v", err)
		} else {
			task.Plan = &researchPlan
		}
	}

	// Parse results if available
	if len(results) > 0 {
		var resultsList []string
		if err := json.Unmarshal(results, &resultsList); err != nil {
			log.Printf("Warning: Failed to parse research results: %v", err)
		} else {
			task.Results = resultsList
		}
	}

	return task, nil
}
