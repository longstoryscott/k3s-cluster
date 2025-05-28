package storage

import (
	"context"
	"fmt"
	"math"
	"proxyllama/util"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
)

// Memory represents a stored memory for a user
type Memory struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// InitMemorySchema initializes the database schema for memory storage
func InitMemorySchema(ctx context.Context) error {
	util.LogInfo("Initializing memory schema...")

	// Create memories table and hypertable
	_, err := Pool.Exec(ctx, GetQuery("memory.init_memory_schema"))
	if err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}
	util.LogInfo("Created memories table")

	// Create indexes for memories
	_, err = Pool.Exec(ctx, GetQuery("memory.create_memory_indexes"))
	if err != nil {
		return fmt.Errorf("failed to create memory indexes: %w", err)
	}
	util.LogInfo("Created memory indexes")

	// Enable compression on memories
	_, err = Pool.Exec(ctx, GetQuery("memory.enable_memories_compression"))
	if err != nil {
		return fmt.Errorf("failed to enable memories compression: %w", err)
	}
	util.LogInfo("Enabled memories compression")

	// Add compression policy for memories
	_, err = Pool.Exec(ctx, GetQuery("memory.memories_compression_policy"))
	if err != nil {
		util.LogWarning("Failed to add memories compression policy", logrus.Fields{"error": err})
	}
	util.LogInfo("Added memories compression policy")

	// Add retention policy for memories
	_, err = Pool.Exec(ctx, GetQuery("memory.memories_retention_policy"))
	if err != nil {
		util.LogWarning("Failed to add memories retention policy", logrus.Fields{"error": err})
	}
	util.LogInfo("Added memories retention policy")

	_, err = Pool.Exec(ctx, GetQuery("memory.create_memory_cascade_delete_trigger"))
	if err != nil {
		util.LogWarning("Failed to create memory cascade delete trigger", logrus.Fields{"error": err})
	} else {
		util.LogInfo("Memory cascade delete trigger created successfully")
	}

	util.LogInfo("Memory schema initialized successfully")
	return nil
}

// StoreMemory stores a memory with embedding for a user
func StoreMemory(ctx context.Context, userID, source string, sourceID int, embedding []float32) error {
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return util.HandleError(fmt.Errorf("failed to begin transaction: %w", err))
	}
	if err := StoreMemoryWithTx(ctx, userID, source, sourceID, embedding, tx); err != nil {
		tx.Rollback(ctx) // rollback on error
		return util.HandleError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return util.HandleError(fmt.Errorf("failed to commit transaction: %w", err))
	}
	return nil
}

// StoreMemoryWithTx stores a memory with embedding for a user within a transaction
func StoreMemoryWithTx(ctx context.Context, userID, source string, sourceID int, embedding []float32, tx pgx.Tx) error {
	pe, _ := processEmbedding(embedding)
	embeddingStr := formatEmbeddingForPgVector(pe)
	_, err := tx.Exec(ctx, GetQuery("memory.store_memory"), userID, sourceID, source, embeddingStr)
	if err != nil {
		return fmt.Errorf("failed to store memory: %w", err)
	}

	return nil
}

// DeleteMemory deletes a memory by ID
func DeleteMemory(ctx context.Context, id, userID string) error {
	_, err := Pool.Exec(ctx, GetQuery("memory.delete_memory"), id, userID)

	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	return nil
}

// DeleteAllUserMemories deletes all memories for a user
func DeleteAllUserMemories(ctx context.Context, userID string) error {
	_, err := Pool.Exec(ctx, GetQuery("memory.delete_all_user_memories"), userID)

	if err != nil {
		return fmt.Errorf("failed to delete all user memories: %w", err)
	}

	return nil
}

// Vector processing utilities (migrated from embedding.go)
// formatEmbeddingForPgVector converts a []float32 to pgvector's string format
func formatEmbeddingForPgVector(embedding []float32) string {
	strValues := make([]string, len(embedding))
	for i, val := range embedding {
		strValues[i] = fmt.Sprintf("%f", val)
	}
	return "[" + strings.Join(strValues, ",") + "]"
}

// processEmbedding adjusts an embedding to fit in 768 dimensions
// Returns the processed embedding and the original dimension
func processEmbedding(embedding []float32) ([]float32, int) {
	originalDimension := len(embedding)
	targetDimension := 768
	switch {
	case originalDimension == targetDimension:
		return embedding, originalDimension
	case originalDimension < targetDimension:
		return padVector(embedding, targetDimension), originalDimension
	case originalDimension > targetDimension:
		return reduceVector(embedding, targetDimension), originalDimension
	}
	return embedding, originalDimension
}

func padVector(vec []float32, targetDimension int) []float32 {
	result := make([]float32, targetDimension)
	copy(result, vec)
	return result
}

func reduceVector(vec []float32, targetDimension int) []float32 {
	originalDimension := len(vec)
	result := make([]float32, targetDimension)
	ratio := float64(originalDimension) / float64(targetDimension)
	for i := 0; i < targetDimension; i++ {
		startIdx := int(math.Floor(float64(i) * ratio))
		endIdx := int(math.Floor(float64(i+1) * ratio))
		if endIdx > originalDimension {
			endIdx = originalDimension
		}
		if startIdx >= endIdx {
			if i < originalDimension {
				result[i] = vec[i]
			}
			continue
		}
		var sum float32 = 0
		for j := startIdx; j < endIdx; j++ {
			sum += vec[j]
		}
		result[i] = sum / float32(endIdx-startIdx)
	}
	return normalizeVector(result)
}

func normalizeVector(vec []float32) []float32 {
	var sum float32 = 0
	for _, v := range vec {
		sum += v * v
	}
	if sum < 1e-10 {
		return vec
	}
	magnitude := float32(math.Sqrt(float64(sum)))
	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = v / magnitude
	}
	return result
}

// GetSimilarMessages finds semantically similar messages based on vector similarity
func GetSimilarMessages(ctx context.Context, queryEmbedding []float32, similarityThreshold float32, limit int) ([]Message, error) {
	// Ensure the query embedding is 768 dimensions
	processedEmbedding, _ := processEmbedding(queryEmbedding)
	embeddingStr := formatEmbeddingForPgVector(processedEmbedding)

	rows, err := Pool.Query(ctx, GetQuery("memory.similar_messages"), embeddingStr, similarityThreshold, limit, "messages")
	if err != nil {
		return nil, util.HandleError(fmt.Errorf("failed to query similar messages: %w", err))
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var similarity float32

		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt, &similarity); err != nil {
			return nil, util.HandleError(fmt.Errorf("failed to scan message row: %w", err))
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}
