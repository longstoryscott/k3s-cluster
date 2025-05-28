package storage

import (
	"context"
	"fmt"
	"proxyllama/util"
	"strings"
)

// Message represents a message with its metadata for search results
type SearchMessage struct {
	ID             int
	Role           string
	Content        string
	ConversationID int
	Similarity     float32 // Used for vector similarity search results
}

// SearchMessagesBySimilarity searches for semantically similar messages in a conversation
func SearchMessagesBySimilarity(ctx context.Context, conversationID int, embedding []float32, limit int) ([]SearchMessage, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	rows, err := Pool.Query(ctx, GetQuery("memory.search_conversation_similarity"), conversationID, embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages by similarity: %w", err)
	}
	defer rows.Close()

	var results []SearchMessage
	for rows.Next() {
		var msg SearchMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID, &msg.Similarity); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	util.LogInfo("Found messages by vector similarity")
	return results, nil
}

// SearchAllMessagesBySimilarity searches for semantically similar messages across all conversations
func SearchAllMessagesBySimilarity(ctx context.Context, embedding []float32, minSimilarity float64, limit int) ([]SearchMessage, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	rows, err := Pool.Query(ctx, GetQuery("memory.search_all_similarity"), embedding, minSimilarity, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search all messages by similarity: %w", err)
	}
	defer rows.Close()

	var results []SearchMessage
	for rows.Next() {
		var msg SearchMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID, &msg.Similarity); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	return results, nil
}

// formatTSQuery formats a text query for PostgreSQL's to_tsquery
// It replaces spaces with '&' for AND operations and handles basic query formatting
func formatTSQuery(query string) string {
	// Remove special characters that might cause issues with to_tsquery
	query = strings.ReplaceAll(query, "'", "")

	// Split the query into words
	words := strings.Fields(query)

	// Filter out stop words and words that are too short
	var filteredWords []string
	for _, word := range words {
		if len(word) > 2 { // Skip words that are too short
			filteredWords = append(filteredWords, word+":*") // Add stemming
		}
	}

	// If no valid words, return a simple query
	if len(filteredWords) == 0 {
		return query
	}

	// Join with & for AND operation
	return strings.Join(filteredWords, " & ")
}
