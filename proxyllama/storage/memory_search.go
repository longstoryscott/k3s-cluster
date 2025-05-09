package storage

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// Search query templates for memory retrieval
const (
	sqlSearchMessagesByKeyword = `
		SELECT m.id, m.role, m.content, m.conversation_id
		FROM messages m
		WHERE m.conversation_id = $1
		AND (to_tsvector('english', m.content) @@ to_tsquery('english', $2) OR
			 m.content ILIKE $3)
		ORDER BY 
			CASE 
				WHEN m.content ILIKE $3 THEN 1 
				ELSE 2 
			END, 
			m.id DESC
		LIMIT $4
	`

	sqlSearchAllMessagesByKeyword = `
		SELECT m.id, m.role, m.content, m.conversation_id
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		WHERE c.user_id = $1
		AND (to_tsvector('english', m.content) @@ to_tsquery('english', $2) OR
			 m.content ILIKE $3)
		ORDER BY 
			CASE 
				WHEN m.content ILIKE $3 THEN 1 
				ELSE 2 
			END, 
			m.id DESC
		LIMIT $4
	`

	sqlSearchMessagesBySimilarity = `
		SELECT m.id, m.role, m.content, m.conversation_id, 
		       1 - (m.embedding <=> $2) as similarity
		FROM messages m
		WHERE m.conversation_id = $1
		AND m.embedding IS NOT NULL
		ORDER BY similarity DESC
		LIMIT $3
	`

	sqlSearchAllMessagesBySimilarity = `
		SELECT m.id, m.role, m.content, m.conversation_id, 
		       1 - (m.embedding <=> $1) as similarity
		FROM messages m
		WHERE m.embedding IS NOT NULL
		AND (1 - (m.embedding <=> $1)) > $2
		ORDER BY similarity DESC
		LIMIT $3
	`
)

// Message represents a message with its metadata for search results
type SearchMessage struct {
	ID             int
	Role           string
	Content        string
	ConversationID int
	Similarity     float32 // Used for vector similarity search results
}

// SearchMessagesByKeyword searches for messages containing keywords in a conversation
func SearchMessagesByKeyword(ctx context.Context, conversationID int, query string, limit int) ([]SearchMessage, error) {
	// Format the query for PostgreSQL's to_tsquery
	tsQuery := formatTSQuery(query)
	likeQuery := "%" + query + "%"

	log.Printf("Searching for messages with keyword query: %s (TS: %s)", query, tsQuery)

	rows, err := Pool.Query(ctx, sqlSearchMessagesByKeyword, conversationID, tsQuery, likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages by keyword: %w", err)
	}
	defer rows.Close()

	var results []SearchMessage
	for rows.Next() {
		var msg SearchMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	log.Printf("Found %d messages matching keyword query", len(results))
	return results, nil
}

// SearchAllMessagesByKeyword searches for messages containing keywords across all conversations
func SearchAllMessagesByKeyword(ctx context.Context, userID string, query string, limit int) ([]SearchMessage, error) {
	// Format the query for PostgreSQL's to_tsquery
	tsQuery := formatTSQuery(query)
	likeQuery := "%" + query + "%"

	rows, err := Pool.Query(ctx, sqlSearchAllMessagesByKeyword, userID, tsQuery, likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search all messages by keyword: %w", err)
	}
	defer rows.Close()

	var results []SearchMessage
	for rows.Next() {
		var msg SearchMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	return results, nil
}

// SearchMessagesBySimilarity searches for semantically similar messages in a conversation
func SearchMessagesBySimilarity(ctx context.Context, conversationID int, embedding []float32, limit int) ([]SearchMessage, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	rows, err := Pool.Query(ctx, sqlSearchMessagesBySimilarity, conversationID, embedding, limit)
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

	log.Printf("Found %d messages by vector similarity", len(results))
	return results, nil
}

// SearchAllMessagesBySimilarity searches for semantically similar messages across all conversations
func SearchAllMessagesBySimilarity(ctx context.Context, embedding []float32, minSimilarity float32, limit int) ([]SearchMessage, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	rows, err := Pool.Query(ctx, sqlSearchAllMessagesBySimilarity, embedding, minSimilarity, limit)
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
