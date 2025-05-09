package storage

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// SQL statements for embedding operations
const (
	sqlStoreMessageEmbedding = `
		INSERT INTO message_embeddings (message_id, embedding)
		VALUES ($1, $2)
		ON CONFLICT (message_id) DO UPDATE SET embedding = $2
	`

	sqlGetSimilarMessages = `
		SELECT m.id, m.conversation_id, m.role, m.content, m.created_at,
		       1 - (e.embedding <=> $1) as similarity
		FROM message_embeddings e
		JOIN messages m ON e.message_id = m.id
		WHERE 1 - (e.embedding <=> $1) > $2
		ORDER BY similarity DESC
		LIMIT $3
	`

	sqlGetMessageEmbedding = `
		SELECT embedding
		FROM message_embeddings
		WHERE message_id = $1
	`

	sqlHasEmbedding = `
		SELECT EXISTS(
			SELECT 1
			FROM message_embeddings
			WHERE message_id = $1
		)
	`
)

// StoreMessageEmbedding stores a vector embedding for a message in the database
func StoreMessageEmbedding(ctx context.Context, messageID int, embedding []float32) error {
	// Convert []float32 to pgvector string format "[f1,f2,...]"
	embeddingStr := formatEmbeddingForPgVector(embedding)

	_, err := Pool.Exec(ctx, sqlStoreMessageEmbedding, messageID, embeddingStr)
	if err != nil {
		return fmt.Errorf("failed to store message embedding for message %d: %w", messageID, err)
	}
	log.Printf("Stored embedding for message %d", messageID)
	return nil
}

// HasEmbedding checks if a message already has an embedding
func HasEmbedding(ctx context.Context, messageID int) (bool, error) {
	var exists bool
	err := Pool.QueryRow(ctx, sqlHasEmbedding, messageID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if message %d has embedding: %w", messageID, err)
	}
	return exists, nil
}

// GetMessageEmbedding retrieves the embedding for a specific message
func GetMessageEmbedding(ctx context.Context, messageID int) ([]float32, error) {
	var embeddingStr string
	err := Pool.QueryRow(ctx, sqlGetMessageEmbedding, messageID).Scan(&embeddingStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding for message %d: %w", messageID, err)
	}

	// Parse the string back to []float32
	return parseEmbeddingFromPgVector(embeddingStr)
}

// GetSimilarMessages finds semantically similar messages based on vector similarity
func GetSimilarMessages(ctx context.Context, queryEmbedding []float32, similarityThreshold float32, limit int) ([]Message, error) {
	embeddingStr := formatEmbeddingForPgVector(queryEmbedding)

	rows, err := Pool.Query(ctx, sqlGetSimilarMessages, embeddingStr, similarityThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var similarity float32

		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt, &similarity); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

// formatEmbeddingForPgVector converts a []float32 to pgvector's string format
func formatEmbeddingForPgVector(embedding []float32) string {
	// Create string representation
	strValues := make([]string, len(embedding))
	for i, val := range embedding {
		strValues[i] = fmt.Sprintf("%f", val)
	}

	return "[" + strings.Join(strValues, ",") + "]"
}

// parseEmbeddingFromPgVector converts pgvector's string format to []float32
func parseEmbeddingFromPgVector(embeddingStr string) ([]float32, error) {
	// Remove brackets
	embeddingStr = strings.Trim(embeddingStr, "[]")

	// Split by comma
	parts := strings.Split(embeddingStr, ",")
	result := make([]float32, len(parts))

	for i, part := range parts {
		var val float64
		_, err := fmt.Sscanf(part, "%f", &val)
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedding value: %w", err)
		}
		result[i] = float32(val)
	}

	return result, nil
}
