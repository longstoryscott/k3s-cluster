package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Summary represents a consolidated summary of messages or other summaries
type Summary struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	Content        string    `json:"content"`
	Level          int       `json:"level"`
	SourceIDs      []int     `json:"source_ids"` // IDs of messages or summaries used to create this summary
	CreatedAt      time.Time `json:"created_at"`
}

// SQL query templates for summary operations
const (
	sqlCreateSummary = `
		INSERT INTO summaries (conversation_id, content, level, source_ids, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id
	`
	sqlGetSummariesForConversation = `
		SELECT id, conversation_id, content, level, source_ids, created_at
		FROM summaries
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`
	sqlGetRecentSummaries = `
		SELECT id, conversation_id, content, level, source_ids, created_at
		FROM summaries
		WHERE conversation_id = $1 AND level = $2
		ORDER BY created_at DESC
		LIMIT $3
	`
	sqlDeleteSummariesForConversation = `
		DELETE FROM summaries
		WHERE conversation_id = $1
	`
)

// CreateSummary creates a new summary in the database using a transaction
func CreateSummary(ctx context.Context, conversationID int, content string, level int, sourceIDs []int) (int, error) {
	// Start a transaction for atomicity
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Will be a no-op if transaction is committed

	sourceIDsJSON, err := json.Marshal(sourceIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal source IDs: %w", err)
	}

	var summaryID int
	err = tx.QueryRow(ctx, sqlCreateSummary,
		conversationID, content, level, sourceIDsJSON).Scan(&summaryID)
	if err != nil {
		return 0, fmt.Errorf("failed to create summary: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return summaryID, nil
}

// GetSummariesForConversation retrieves all summaries for a conversation with efficient error handling
func GetSummariesForConversation(ctx context.Context, conversationID int) ([]Summary, error) {
	rows, err := Pool.Query(ctx, sqlGetSummariesForConversation, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query summaries: %w", err)
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var summary Summary
		var sourceIDsJSON []byte

		if err := rows.Scan(&summary.ID, &summary.ConversationID, &summary.Content, &summary.Level, &sourceIDsJSON, &summary.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan summary row: %w", err)
		}

		// Parse source IDs from JSON directly into the struct field
		if err := json.Unmarshal(sourceIDsJSON, &summary.SourceIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source IDs: %w", err)
		}

		summaries = append(summaries, summary)
	}

	// Check for any errors that occurred during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating summary rows: %w", err)
	}

	return summaries, nil
}

// GetRecentSummaries gets summaries for a specific level, ordered by most recent
// with improved error handling
func GetRecentSummaries(ctx context.Context, conversationID, level, limit int) ([]Summary, error) {
	rows, err := Pool.Query(ctx, sqlGetRecentSummaries, conversationID, level, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent summaries: %w", err)
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var summary Summary
		var sourceIDsJSON []byte

		if err := rows.Scan(&summary.ID, &summary.ConversationID, &summary.Content, &summary.Level, &sourceIDsJSON, &summary.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan recent summary row: %w", err)
		}

		if err := json.Unmarshal(sourceIDsJSON, &summary.SourceIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source IDs for recent summary: %w", err)
		}

		summaries = append(summaries, summary)
	}

	// Check for any errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent summary rows: %w", err)
	}

	return summaries, nil
}

// DeleteSummariesForConversation deletes all summaries for a conversation
// using a transaction for consistency
func DeleteSummariesForConversation(ctx context.Context, conversationID int) error {
	// Start a transaction
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, sqlDeleteSummariesForConversation, conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete summaries: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSummary retrieves a single summary by its ID
func GetSummary(ctx context.Context, summaryID int) (*Summary, error) {
	const sqlGetSummary = `
		SELECT id, conversation_id, content, level, source_ids, created_at
		FROM summaries
		WHERE id = $1
	`

	var summary Summary
	var sourceIDsJSON []byte

	err := Pool.QueryRow(ctx, sqlGetSummary, summaryID).Scan(
		&summary.ID, &summary.ConversationID, &summary.Content, &summary.Level,
		&sourceIDsJSON, &summary.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	// Parse source IDs from JSON
	if err := json.Unmarshal(sourceIDsJSON, &summary.SourceIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source IDs: %w", err)
	}

	return &summary, nil
}
