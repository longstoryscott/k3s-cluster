package storage

import (
	"context"
	"log"
	"proxyllama/config"
	"proxyllama/proxy"
	"time"
)

type Message struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	Role           string    `json:"role"` // "user" or "assistant"
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// SQL query templates for message operations
const (
	sqlAddMessage = `
        INSERT INTO messages (conversation_id, role, content) 
        VALUES ($1, $2, $3) 
        RETURNING id
    `
	sqlGetMessage = `
		SELECT id, conversation_id, role, content, created_at
		FROM messages
		WHERE id = $1
	`
	sqlGetConversationHistory = `
		SELECT id, conversation_id, role, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`
)

// AddMessage adds a message to a conversation
func AddMessage(ctx context.Context, conversationID int, role, content string) (int, error) {
	// Start a transaction for atomicity
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	// Use defer with a named error return to ensure we correctly handle transaction state
	defer func() {
		if err != nil {
			tx.Rollback(ctx) // rollback on error
		}
	}()

	var messageID int
	// Use the SQL query directly instead of a prepared statement
	err = tx.QueryRow(ctx, sqlAddMessage, conversationID, role, content).Scan(&messageID)
	if err != nil {
		return 0, err
	}

	// Update the conversation's updated_at timestamp
	_, err = tx.Exec(ctx, `
        UPDATE conversations 
        SET updated_at = NOW() 
        WHERE id = $1
    `, conversationID)
	if err != nil {
		return 0, err
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}

	// Create message object for caching
	message := &Message{
		ID:             messageID,
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		CreatedAt:      time.Now(), // Approximate time until we fetch from DB
	}

	// Cache the new message
	if err := CacheMessage(ctx, message); err != nil {
		// Log but don't fail on cache error
		log.Printf("Warning: Failed to cache message %d: %v", messageID, err)
		// We'll just have a cache miss next time
	}

	// Invalidate the conversation message list cache
	InvalidateConversationMessagesCache(ctx, conversationID)

	// After successful commit and getting messageID, generate and store embedding in the background
	conf := config.GetConfig()
	if conf.Summarization.EnableRAG {
		go func(mID int, mContent string) {
			// Use a background context for the goroutine
			bgCtx := context.Background()
			ctx, cancel := context.WithTimeout(bgCtx, 30*time.Second)
			defer cancel()

			// Choose embedding model from config
			embeddingModel := conf.Summarization.EmbeddingModel
			if embeddingModel == "" {
				log.Println("Embedding model not configured, skipping embedding generation.")
				return
			}

			// Check if this message already has an embedding
			hasEmbedding, err := HasEmbedding(ctx, mID)
			if err != nil {
				log.Printf("Error checking embedding for message %d: %v", mID, err)
				return
			}

			// Skip if embedding already exists
			if hasEmbedding {
				log.Printf("Message %d already has an embedding, skipping generation", mID)
				return
			}

			log.Printf("Generating embedding for message %d", mID)
			embedding, err := proxy.GetEmbedding(ctx, mContent, embeddingModel)
			if err != nil {
				log.Printf("Error generating embedding for message %d: %v", mID, err)
				return
			}

			if len(embedding) == 0 {
				log.Printf("Warning: Got empty embedding for message %d", mID)
				return
			}

			log.Printf("Storing embedding for message %d (vector size: %d)", mID, len(embedding))
			if err := StoreMessageEmbedding(ctx, mID, embedding); err != nil {
				log.Printf("Error storing embedding for message %d: %v", mID, err)
			}
		}(messageID, content)
	}

	return messageID, nil
}

// GetMessage gets a single message by ID
func GetMessage(ctx context.Context, messageID int) (*Message, error) {
	// Try to get from cache first
	if msg, found := GetMessageFromCache(ctx, messageID); found {
		return msg, nil
	}

	// Not in cache, get from database
	var msg Message
	err := Pool.QueryRow(ctx, sqlGetMessage, messageID).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Cache for future use
	if err := CacheMessage(ctx, &msg); err != nil {
		// Just log, don't fail on cache error
		log.Printf("Warning: Failed to cache message %d: %v", msg.ID, err)
	}

	return &msg, nil
}

// GetConversationHistory retrieves all messages for a conversation
func GetConversationHistory(ctx context.Context, conversationID int) ([]Message, error) {
	// Try to get from cache first
	if messages, found := GetMessagesByConversationIDFromCache(ctx, conversationID); found {
		return messages, nil
	}

	// Not in cache, get from database
	rows, err := Pool.Query(ctx, sqlGetConversationHistory, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Cache the message list
	if err := CacheMessagesByConversationID(ctx, conversationID, messages); err != nil {
		// Just log, don't fail on cache error
		log.Printf("Warning: Failed to cache messages for conversation %d: %v", conversationID, err)
	}

	return messages, nil
}
