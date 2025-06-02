package storage

import (
	"context"
	"fmt"
	"proxyllama/models"
	"proxyllama/util"
)

// SearchMessagesBySimilarity searches for semantically similar messages in a conversation
func SearchMessagesBySimilarity(ctx context.Context, conversationID int, embedding []float32, limit int) ([]models.Memory, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	rows, err := Pool.Query(ctx, GetQuery("memory.search_conversation_similarity"), conversationID, formatEmbeddingForPgVector(embedding), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages by similarity: %w", err)
	}
	defer rows.Close()

	var results []models.Memory
	for rows.Next() {
		var msg models.Memory
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID, &msg.Similarity, &msg.SourceType); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	util.LogInfo("Found messages by vector similarity")
	return results, nil
}

// SearchAllMessagesBySimilarity searches for semantically similar messages across all conversations
func SearchAllMessagesBySimilarity(ctx context.Context, embedding []float32, minSimilarity float64, limit int) ([]models.Memory, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	rows, err := Pool.Query(ctx, GetQuery("memory.search_all_similarity"), formatEmbeddingForPgVector(embedding), minSimilarity, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search all messages by similarity: %w", err)
	}
	defer rows.Close()

	var results []models.Memory
	for rows.Next() {
		var msg models.Memory
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID, &msg.Similarity, &msg.SourceType); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	return results, nil
}
