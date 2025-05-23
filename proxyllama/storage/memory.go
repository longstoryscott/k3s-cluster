package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
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

	// Create indexes for memories
	_, err = Pool.Exec(ctx, GetQuery("memory.create_memory_indexes"))
	if err != nil {
		return fmt.Errorf("failed to create memory indexes: %w", err)
	}

	// Enable compression on memories
	_, err = Pool.Exec(ctx, GetQuery("memory.enable_memories_compression"))
	if err != nil {
		return fmt.Errorf("failed to enable memories compression: %w", err)
	}

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

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Memory schema initialized successfully")
	return nil
}

// StoreMemory stores a memory with embedding for a user
func StoreMemory(ctx context.Context, id, userID, content, source string, embedding []float32) error {
	_, err := Pool.Exec(ctx, GetQuery("memory.store_memory"),
		id, userID, content, source, embedding, time.Now())

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
