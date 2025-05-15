package storage

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pgvector/pgvector-go"
	"github.com/sirupsen/logrus"
)

// HasEmbedding checks if a message already has an embedding
func HasEmbedding(ctx context.Context, messageID int) (bool, error) {
	var exists bool
	err := Pool.QueryRow(ctx, GetQuery("embedding.has_embedding"), messageID).Scan(&exists)
	return exists, err
}

// StoreMessageEmbedding stores a vector embedding for a message
func StoreMessageEmbedding(ctx context.Context, messageID int, embedding []float32) error {
	// Use a transaction for atomicity
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Convert []float32 to pgvector type
	pgVector := pgvector.NewVector(embedding)
	copy(pgVector.Slice(), embedding)

	// Use the consolidated SQL query
	_, err = tx.Exec(ctx, GetQuery("embedding.store_message_embedding"),
		messageID, pgVector, "default") // Using 'default' as model name for now

	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":      filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":      line,
		"messageID": messageID,
	}).Info("Successfully stored embedding for message")
	return nil
}

// GetSimilarMessages finds semantically similar messages based on vector similarity
func GetSimilarMessages(ctx context.Context, queryEmbedding []float32, similarityThreshold float32, limit int) ([]Message, error) {
	embeddingStr := formatEmbeddingForPgVector(queryEmbedding)

	rows, err := Pool.Query(ctx, GetQuery("embedding.similar_messages"), embeddingStr, similarityThreshold, limit)
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

// DeleteMessageEmbeddings removes all embedding records for a specific message ID
func DeleteMessageEmbeddings(ctx context.Context, messageID int) error {
	_, err := Pool.Exec(ctx, GetQuery("embedding.delete_embeddings"), messageID)

	if err != nil {
		return fmt.Errorf("failed to delete embeddings: %w", err)
	}

	return nil
}

// GetMessageEmbedding retrieves an embedding for a message ID
func GetMessageEmbedding(ctx context.Context, messageID int) ([]float32, string, int, error) {
	var embedding []float32
	var modelName string
	var originalDimension int

	err := Pool.QueryRow(ctx, GetQuery("embedding.get_embedding"), messageID).Scan(&embedding, &modelName, &originalDimension)

	if err != nil {
		return nil, "", 0, fmt.Errorf("no embedding found for message ID %d: %w", messageID, err)
	}

	return embedding, modelName, originalDimension, nil
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

// StoreEmbedding stores a vector embedding for a message
// For vectors > 768 dimensions, it performs dimensionality reduction
// For vectors < 768 dimensions, it pads with zeros
func StoreEmbedding(ctx context.Context, messageID int, embedding []float32) error {
	if embedding == nil {
		return fmt.Errorf("embedding cannot be nil")
	}

	// Process embedding to fit in the 768-dimension table
	processedEmbedding, originalDimension := processEmbedding(embedding)

	// Store embedding in the table
	_, err := Pool.Exec(ctx, GetQuery("embedding.store_embedding"),
		messageID, processedEmbedding, originalDimension)

	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	return nil
}

// SearchSimilarMessages finds messages with similar embeddings to the query
func SearchSimilarMessages(ctx context.Context, queryEmbedding []float32, limit int) ([]int, error) {
	if queryEmbedding == nil {
		return nil, fmt.Errorf("query embedding cannot be nil")
	}

	// Process the query embedding to match the 768-dimension format
	processedEmbedding, _ := processEmbedding(queryEmbedding)

	// Query for similar messages
	rows, err := Pool.Query(ctx, GetQuery("embedding.search_similar"), processedEmbedding, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to search similar messages: %w", err)
	}
	defer rows.Close()

	var result []int
	for rows.Next() {
		var messageID int
		if err := rows.Scan(&messageID); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result = append(result, messageID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// GetMessageEmbeddingStats returns statistics about embeddings stored for each model
func GetMessageEmbeddingStats(ctx context.Context) (map[string]map[string]int, error) {
	stats := make(map[string]map[string]int)
	dimensionStats := make(map[string]int) // Track different original dimensions

	// Get model stats
	rows, err := Pool.Query(ctx, GetQuery("embedding.get_model_stats"))

	if err != nil {
		return nil, fmt.Errorf("failed to get embedding stats: %w", err)
	}
	defer rows.Close()

	modelStats := make(map[string]int)
	for rows.Next() {
		var modelName string
		var count int
		if err := rows.Scan(&modelName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan model stats row: %w", err)
		}
		modelStats[modelName] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating model stats rows: %w", err)
	}

	// Get dimension stats
	rows2, err := Pool.Query(ctx, GetQuery("embedding.get_dimension_stats"))

	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Failed to get dimension stats")
	} else {
		defer rows2.Close()

		for rows2.Next() {
			var dimension int
			var count int
			if err := rows2.Scan(&dimension, &count); err != nil {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"error": err,
				}).Warn("Failed to scan dimension stats row")
				continue
			}
			dimensionStats[fmt.Sprintf("%d", dimension)] = count
		}

		if err := rows2.Err(); err != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": err,
			}).Warn("Error iterating dimension stats rows")
		}
	}

	// Build the final stats map
	stats["models"] = modelStats
	stats["dimensions"] = dimensionStats

	return stats, nil
}

// processEmbedding adjusts an embedding to fit in 768 dimensions
// Returns the processed embedding and the original dimension
func processEmbedding(embedding []float32) ([]float32, int) {
	originalDimension := len(embedding)
	targetDimension := 768

	switch {
	case originalDimension == targetDimension:
		// Perfect match, no processing needed
		return embedding, originalDimension

	case originalDimension < targetDimension:
		// Smaller vector - pad with zeros
		return padVector(embedding, targetDimension), originalDimension

	case originalDimension > targetDimension:
		// Larger vector - reduce dimensions
		return reduceVector(embedding, targetDimension), originalDimension
	}

	return embedding, originalDimension // Shouldn't reach here
}

// padVector pads a vector with zeros to reach the target dimension
func padVector(vec []float32, targetDimension int) []float32 {
	result := make([]float32, targetDimension)
	copy(result, vec)
	return result
}

// reduceVector reduces a vector to the target dimension using a simple projection technique
// We're using "mean pooling" to compress groups of dimensions together
func reduceVector(vec []float32, targetDimension int) []float32 {
	originalDimension := len(vec)
	result := make([]float32, targetDimension)

	// Using mean pooling to reduce dimensions
	// Each target dimension is an average of a group of source dimensions
	ratio := float64(originalDimension) / float64(targetDimension)

	for i := 0; i < targetDimension; i++ {
		// Calculate the range of source dimensions to average for this target dimension
		startIdx := int(math.Floor(float64(i) * ratio))
		endIdx := int(math.Floor(float64(i+1) * ratio))
		if endIdx > originalDimension {
			endIdx = originalDimension
		}

		// Handle edge case
		if startIdx >= endIdx {
			if i < originalDimension {
				result[i] = vec[i]
			}
			continue
		}

		// Calculate mean for this window
		var sum float32 = 0
		for j := startIdx; j < endIdx; j++ {
			sum += vec[j]
		}
		result[i] = sum / float32(endIdx-startIdx)
	}

	// Normalize the reduced vector
	return normalizeVector(result)
}

// normalizeVector normalizes a vector to have unit length (L2 norm)
func normalizeVector(vec []float32) []float32 {
	var sum float32 = 0
	for _, v := range vec {
		sum += v * v
	}

	// If vector is all zeros or very small, return as is
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
