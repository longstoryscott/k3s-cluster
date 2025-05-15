package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

type Summary struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	Content        string    `json:"content"`
	Level          int       `json:"level"` // 1 = first level, 2 = summary of summaries, etc.
	SourceIDs      []int     `json:"source_ids"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateSummary adds a new summary for a conversation
func CreateSummary(ctx context.Context, conversationID int, content string, level int, sourceIDs []int) (int, error) {
	// Convert source IDs to JSON
	sourceIDsJSON, err := json.Marshal(sourceIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal source IDs: %w", err)
	}

	var summaryID int
	err = Pool.QueryRow(ctx, GetQuery("summary.create_summary"),
		conversationID, content, level, sourceIDsJSON).Scan(&summaryID)
	if err != nil {
		return 0, fmt.Errorf("failed to create summary: %w", err)
	}

	// Invalidate the conversation summaries cache
	InvalidateConversationSummariesCache(ctx, conversationID)

	return summaryID, nil
}

// GetSummariesForConversation gets all summaries for a conversation
func GetSummariesForConversation(ctx context.Context, conversationID int) ([]Summary, error) {
	// Try to get from cache first
	if summaries, found := GetSummariesByConversationIDFromCache(ctx, conversationID); found {
		return summaries, nil
	}

	// Not in cache, get from database
	rows, err := Pool.Query(ctx, GetQuery("summary.get_summaries_for_conversation"), conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query summaries: %w", err)
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var s Summary
		var sourceIDsJSON []byte

		if err := rows.Scan(&s.ID, &s.ConversationID, &s.Content, &s.Level, &sourceIDsJSON, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan summary row: %w", err)
		}

		// Parse source IDs JSON
		if err := json.Unmarshal(sourceIDsJSON, &s.SourceIDs); err != nil {
			// If JSON parsing fails, initialize an empty slice
			s.SourceIDs = make([]int, 0)
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": err,
			}).Warn("Failed to parse source IDs JSON")
		}

		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating summary rows: %w", err)
	}

	// Cache the results
	if err := CacheSummariesByConversationID(ctx, conversationID, summaries); err != nil {
		// Just log, don't fail on cache error
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":           line,
			"error":          err,
			"conversationId": conversationID,
		}).Warn("Failed to cache summaries for conversation")
	}

	return summaries, nil
}

// GetRecentSummaries gets recent summaries for a conversation at a specific level
func GetRecentSummaries(ctx context.Context, conversationID int, level int, limit int) ([]Summary, error) {
	// This one doesn't use cache because it's more dynamic with limit parameter

	rows, err := Pool.Query(ctx, GetQuery("summary.get_recent_summaries"), conversationID, level, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent summaries: %w", err)
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var s Summary
		var sourceIDsJSON []byte

		if err := rows.Scan(&s.ID, &s.ConversationID, &s.Content, &s.Level, &sourceIDsJSON, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan recent summary row: %w", err)
		}

		// Parse source IDs JSON
		if err := json.Unmarshal(sourceIDsJSON, &s.SourceIDs); err != nil {
			// If JSON parsing fails, initialize an empty map
			s.SourceIDs = make([]int, 0)
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": err,
			}).Warn("Failed to parse source IDs JSON")
		}

		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent summary rows: %w", err)
	}

	return summaries, nil
}

// DeleteSummariesForConversation deletes all summaries for a conversation
func DeleteSummariesForConversation(ctx context.Context, conversationID int) error {
	_, err := Pool.Exec(ctx, GetQuery("summary.delete_summaries"), conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete summaries: %w", err)
	}

	// Invalidate the conversation summaries cache
	InvalidateConversationSummariesCache(ctx, conversationID)

	return nil
}

// GetSummary gets a single summary by ID
func GetSummary(ctx context.Context, summaryID int) (*Summary, error) {
	var s Summary
	var sourceIDsJSON []byte

	err := Pool.QueryRow(ctx, GetQuery("summary.get_summary"), summaryID).Scan(
		&s.ID, &s.ConversationID, &s.Content, &s.Level, &sourceIDsJSON, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	// Parse source IDs JSON
	if err := json.Unmarshal(sourceIDsJSON, &s.SourceIDs); err != nil {
		// If JSON parsing fails, initialize an empty map
		s.SourceIDs = make([]int, 0)
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Failed to parse source IDs JSON")
	}

	return &s, nil
}
