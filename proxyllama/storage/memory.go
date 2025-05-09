package storage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Memory represents a stored memory for a user
type Memory struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// Memory schema SQL statements
const (
	sqlCreateMemoriesTable = `
		CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			content TEXT NOT NULL,
			source TEXT NOT NULL,
			embedding vector(1536),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`

	sqlCreateMemoriesIndex = `
		CREATE INDEX IF NOT EXISTS idx_memories_user_id ON memories(user_id);
		CREATE INDEX IF NOT EXISTS idx_memories_vector ON memories USING hnsw (embedding vector_cosine_ops);
	`
)

// InitMemorySchema initializes the memory database schema
func InitMemorySchema(ctx context.Context) {
	log.Println("Initializing memory database schema...")

	// Create memories table
	if _, err := Pool.Exec(ctx, sqlCreateMemoriesTable); err != nil {
		log.Fatalf("Failed to create memories table: %v", err)
	}

	// Create memories indexes
	if _, err := Pool.Exec(ctx, sqlCreateMemoriesIndex); err != nil {
		log.Fatalf("Failed to create memories indexes: %v", err)
	}

	log.Println("Memory database schema initialized successfully")
}

// StoreMemory stores a memory with embedding for a user
func StoreMemory(ctx context.Context, id, userID, content, source string, embedding []float32) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO memories (id, user_id, content, source, embedding, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding
	`, id, userID, content, source, embedding, time.Now())

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

	rows, err := Pool.Query(ctx, `
		SELECT id, user_id, content, source, created_at
		FROM memories
		WHERE user_id = $1
		ORDER BY embedding <=> $2
		LIMIT $3
	`, userID, embedding, limit)

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
	_, err := Pool.Exec(ctx, `
		DELETE FROM memories
		WHERE id = $1 AND user_id = $2
	`, id, userID)

	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	return nil
}

// DeleteAllUserMemories deletes all memories for a user
func DeleteAllUserMemories(ctx context.Context, userID string) error {
	_, err := Pool.Exec(ctx, `
		DELETE FROM memories
		WHERE user_id = $1
	`, userID)

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
