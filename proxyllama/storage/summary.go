package storage

import (
	"context"
	"encoding/json"
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

// CreateSummary creates a new summary in the database
func CreateSummary(ctx context.Context, conversationID int, content string, level int, sourceIDs []int) (int, error) {
	sourceIDsJSON, err := json.Marshal(sourceIDs)
	if err != nil {
		return 0, err
	}

	var summaryID int
	err = DB.QueryRow(ctx, `
		INSERT INTO summaries (conversation_id, content, level, source_ids, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id
	`, conversationID, content, level, sourceIDsJSON).Scan(&summaryID)

	return summaryID, err
}

// GetSummariesForConversation retrieves all summaries for a conversation
func GetSummariesForConversation(ctx context.Context, conversationID int) ([]Summary, error) {
	rows, err := DB.Query(ctx, `
		SELECT id, conversation_id, content, level, source_ids, created_at
		FROM summaries
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var summary Summary
		var sourceIDsJSON []byte

		if err := rows.Scan(&summary.ID, &summary.ConversationID, &summary.Content, &summary.Level, &sourceIDsJSON, &summary.CreatedAt); err != nil {
			return nil, err
		}

		// Parse source IDs from JSON
		if err := json.Unmarshal(sourceIDsJSON, &summary.SourceIDs); err != nil {
			return nil, err
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// GetRecentSummaries gets summaries for a specific level, ordered by most recent
func GetRecentSummaries(ctx context.Context, conversationID, level, limit int) ([]Summary, error) {
	rows, err := DB.Query(ctx, `
		SELECT id, conversation_id, content, level, source_ids, created_at
		FROM summaries
		WHERE conversation_id = $1 AND level = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, conversationID, level, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var summary Summary
		var sourceIDsJSON []byte

		if err := rows.Scan(&summary.ID, &summary.ConversationID, &summary.Content, &summary.Level, &sourceIDsJSON, &summary.CreatedAt); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(sourceIDsJSON, &summary.SourceIDs); err != nil {
			return nil, err
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// DeleteSummariesForConversation deletes all summaries for a conversation
func DeleteSummariesForConversation(ctx context.Context, conversationID int) error {
	_, err := DB.Exec(ctx, `
		DELETE FROM summaries
		WHERE conversation_id = $1
	`, conversationID)
	return err
}
