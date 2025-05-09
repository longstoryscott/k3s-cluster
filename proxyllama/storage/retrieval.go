package storage

import (
	"context"
	"fmt"
	"log"
)

// Deprecated constants - use constants from memory_search.go
// These are kept only for backward compatibility
const (
	sqlDeprecatedSearchMessagesByKeyword = `
		SELECT id, conversation_id, role, content, created_at
		FROM messages
		WHERE conversation_id = $1 AND to_tsvector('english', content) @@ plainto_tsquery('english', $2)
		ORDER BY created_at DESC
		LIMIT $3
	`
)

// SearchMessagesByKeywordLegacy is deprecated - use the version from memory_search.go
// This is kept only for backward compatibility
func SearchMessagesByKeywordLegacy(ctx context.Context, conversationID int, query string, limit int) ([]Message, error) {
	log.Printf("Warning: Using deprecated SearchMessagesByKeywordLegacy from retrieval.go")

	if limit <= 0 {
		limit = 5 // Default limit
	}

	rows, err := Pool.Query(ctx, sqlDeprecatedSearchMessagesByKeyword, conversationID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages by keyword: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

// SearchMessagesBySimilarityLegacy is deprecated - use the version from memory_search.go
// This is kept only for backward compatibility
func SearchMessagesBySimilarityLegacy(ctx context.Context, conversationID int, queryEmbedding []float32, limit int) ([]Message, error) {
	log.Printf("Warning: Using deprecated SearchMessagesBySimilarityLegacy from retrieval.go")

	if limit <= 0 {
		limit = 5 // Default limit
	}

	// Use the memory_search implementation
	searchResults, err := SearchMessagesBySimilarity(ctx, conversationID, queryEmbedding, limit)
	if err != nil {
		return nil, err
	}

	var messages []Message
	for _, result := range searchResults {
		msg := Message{
			ID:             result.ID,
			ConversationID: result.ConversationID,
			Role:           result.Role,
			Content:        result.Content,
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// SearchAllMessagesBySimilarityLegacy is deprecated - use the version from memory_search.go
// This is kept only for backward compatibility
func SearchAllMessagesBySimilarityLegacy(ctx context.Context, queryEmbedding []float32, similarityThreshold float32, limit int) ([]Message, error) {
	log.Printf("Warning: Using deprecated SearchAllMessagesBySimilarityLegacy from retrieval.go")

	if limit <= 0 {
		limit = 5 // Default limit
	}

	if similarityThreshold <= 0 || similarityThreshold > 1.0 {
		similarityThreshold = 0.7 // Default threshold
	}

	// Use the memory_search implementation
	searchResults, err := SearchAllMessagesBySimilarity(ctx, queryEmbedding, similarityThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search all messages by similarity: %w", err)
	}

	var messages []Message
	for _, result := range searchResults {
		msg := Message{
			ID:             result.ID,
			ConversationID: result.ConversationID,
			Role:           result.Role,
			Content:        result.Content,
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// truncateContent is a helper function to truncate content for logging
func truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}
