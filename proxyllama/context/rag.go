package context

import (
	"context"
	"fmt"
	"log"
	"proxyllama/config"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/storage"
	"time"
)

// RetrievedMemory represents a message that was retrieved based on vector similarity
type RetrievedMemory struct {
	Message    storage.Message
	Similarity float32
}

// GetRelevantMemories retrieves semantically similar messages based on a query
func GetRelevantMemories(ctx context.Context, query string, limit int) ([]models.OllamaMessage, error) {
	conf := config.GetConfig()

	// Skip if RAG is not enabled
	if !conf.Summarization.EnableRAG {
		return nil, nil
	}

	// Set default limit if not specified
	if limit <= 0 {
		limit = 5 // Default to 5 most relevant memories
	}

	// Get embedding for the query
	embeddingModel := conf.Summarization.EmbeddingModel
	if embeddingModel == "" {
		return nil, fmt.Errorf("embedding model not configured")
	}

	log.Printf("Generating embedding for query to find relevant memories")
	queryEmbedding, err := proxy.GetEmbedding(ctx, query, embeddingModel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Define similarity threshold (adjust as needed)
	similarityThreshold := float32(0.7) // Messages with 70%+ similarity

	// Get similar messages from database
	similarMessages, err := storage.GetSimilarMessages(ctx, queryEmbedding, similarityThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve similar messages: %w", err)
	}

	if len(similarMessages) == 0 {
		log.Printf("No relevant memories found for query")
		return nil, nil
	}

	log.Printf("Found %d relevant memories for query", len(similarMessages))

	// Convert to OllamaMessages for context
	var ollamaMessages []models.OllamaMessage
	for _, msg := range similarMessages {
		ollamaMessages = append(ollamaMessages, models.OllamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return ollamaMessages, nil
}

// EnhanceRequestWithRAG adds relevant memories to the request based on the latest user query
func EnhanceRequestWithRAG(ctx context.Context, req *models.OllamaReq) error {
	// Find the latest user message to use as query
	var latestUserMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			latestUserMessage = req.Messages[i].Content
			break
		}
	}

	if latestUserMessage == "" {
		return fmt.Errorf("no user message found in request")
	}

	// Set context timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get relevant memories
	relevantMemories, err := GetRelevantMemories(timeoutCtx, latestUserMessage, 5)
	if err != nil {
		log.Printf("Warning: Failed to get relevant memories: %v", err)
		return nil // Continue without RAG enhancement
	}

	if len(relevantMemories) == 0 {
		return nil // No relevant memories found, continue with original request
	}

	// Create a new request with memories inserted at the right position
	var enhancedMessages []models.OllamaMessage

	// First add any system messages (these should come first)
	systemMessagesCount := 0
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			enhancedMessages = append(enhancedMessages, msg)
			systemMessagesCount++
		}
	}

	// Add a system message to explain the memories
	enhancedMessages = append(enhancedMessages, models.OllamaMessage{
		Role:    "system",
		Content: fmt.Sprintf("Here are %d relevant memories from previous conversations that may help answer the current query:", len(relevantMemories)),
	})

	// Add the retrieved memories
	enhancedMessages = append(enhancedMessages, relevantMemories...)

	// Add another system message to separate memories from the current conversation
	enhancedMessages = append(enhancedMessages, models.OllamaMessage{
		Role:    "system",
		Content: "Now continuing with the current conversation:",
	})

	// Add the non-system messages from the original request
	for i, msg := range req.Messages {
		if i >= systemMessagesCount {
			enhancedMessages = append(enhancedMessages, msg)
		}
	}

	// Replace messages in the request
	req.Messages = enhancedMessages

	log.Printf("Enhanced request with %d relevant memories", len(relevantMemories))
	return nil
}
