package storage

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Initializing memory schema...")

	// Create memories table and hypertable
	_, err := Pool.Exec(ctx, GetQuery("memory.init_memory_schema"))
	if err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Created memories table")

	// Create indexes for memories
	_, err = Pool.Exec(ctx, GetQuery("memory.create_memory_indexes"))
	if err != nil {
		return fmt.Errorf("failed to create memory indexes: %w", err)
	}
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Created memory indexes")

	// Enable compression on memories
	_, err = Pool.Exec(ctx, GetQuery("memory.enable_memories_compression"))
	if err != nil {
		return fmt.Errorf("failed to enable memories compression: %w", err)
	}
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Enabled memories compression")

	// Add compression policy for memories
	_, err = Pool.Exec(ctx, GetQuery("memory.memories_compression_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Failed to add memories compression policy")
	}
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Added memories compression policy")

	// Add retention policy for memories
	_, err = Pool.Exec(ctx, GetQuery("memory.memories_retention_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Failed to add memories retention policy")
	}
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Added memories retention policy")

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Memory schema initialized successfully")
	return nil
}

// StoreMemory stores a memory with embedding for a user
func StoreMemory(ctx context.Context, userID, source string, sourceID int, embedding []float32) error {
	_, err := Pool.Exec(ctx, GetQuery("memory.store_memory"),
		userID, sourceID, source, embedding)

	if err != nil {
		return fmt.Errorf("failed to store memory: %w", err)
	}

	return nil
}

// SearchMemories searches for memories based on semantic similarity
func SearchMemories(ctx context.Context, userID, query string, limit int) ([]Memory, error) {
	// Generate embedding for the query
	embedding, err := generateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for query: %w", err)
	}

	rows, err := Pool.Query(ctx, GetQuery("memory.search_memories"),
		userID, embedding, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var memory Memory
		if err := rows.Scan(&memory.ID, &memory.UserID, &memory.Content, &memory.Source, &memory.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		memories = append(memories, memory)
	}

	return memories, nil
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

// generateEmbedding generates an embedding for a text using the embedding model
func generateEmbedding(text string) ([]float32, error) {
	// TODO: Implement embedding generation using a proper embedding model
	// For now, return a placeholder embedding
	// In a real implementation, this would call an embedding API like OpenAI's

	// Creating placeholder embedding with 1536 dimensions (OpenAI's standard)
	placeholder := make([]float32, 1536)
	for i := range placeholder {
		placeholder[i] = 0.0
	}

	return placeholder, nil
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

// HasMemoryEmbedding checks if a memory already has an embedding (by id)
func HasMemoryEmbedding(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := Pool.QueryRow(ctx, GetQuery("memory.has_embedding"), id).Scan(&exists)
	return exists, err
}

// GetSimilarMemories finds semantically similar memories for a user
func GetSimilarMemories(ctx context.Context, userID string, queryEmbedding []float32, similarityThreshold float32, limit int) ([]Memory, error) {
	processedEmbedding, _ := processEmbedding(queryEmbedding)
	rows, err := Pool.Query(ctx, GetQuery("memory.similar_memories"), userID, processedEmbedding, similarityThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar memories: %w", err)
	}
	defer rows.Close()
	var memories []Memory
	for rows.Next() {
		var memory Memory
		var similarity float32
		if err := rows.Scan(&memory.ID, &memory.UserID, &memory.Content, &memory.Source, &memory.CreatedAt, &similarity); err != nil {
			return nil, fmt.Errorf("failed to scan memory row: %w", err)
		}
		memories = append(memories, memory)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating memory rows: %w", err)
	}
	return memories, nil
}

// GetSimilarMessages finds semantically similar messages based on vector similarity
func GetSimilarMessages(ctx context.Context, queryEmbedding []float32, similarityThreshold float32, limit int) ([]Message, error) {
	// Ensure the query embedding is 768 dimensions
	processedEmbedding, _ := processEmbedding(queryEmbedding)
	embeddingStr := formatEmbeddingForPgVector(processedEmbedding)

	rows, err := Pool.Query(ctx, GetQuery("memory.similar_messages"), embeddingStr, similarityThreshold, limit)
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
