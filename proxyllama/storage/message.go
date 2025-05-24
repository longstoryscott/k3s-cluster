package storage

import (
	"context"
	"path/filepath"
	"proxyllama/config"
	"proxyllama/proxy"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

type Message struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	Role           string    `json:"role"` // "user" or "assistant"
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// AddMessage adds a message to a conversation
func AddMessage(ctx context.Context, conversationID int, role, content string, usrCfg *config.UserConfig) (int, error) {
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
	// Use the SQL query from our loader
	err = tx.QueryRow(ctx, GetQuery("message.add_message"), conversationID, role, content).Scan(&messageID)
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
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":      filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":      line,
			"messageId": messageID,
			"error":     err,
		}).Warn("Warning: Failed to cache message")
		// We'll just have a cache miss next time
	}

	// Invalidate the conversation message list cache
	InvalidateConversationMessagesCache(ctx, conversationID)

	profile, err := GetModelProfile(ctx, usrCfg.ModelProfiles.EmbeddingProfileID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to get model profile for embedding")
		return messageID, nil // Proceed without embedding if profile retrieval fails
	}

	// After successful commit and getting messageID, generate and store embedding in the background
	if usrCfg.Summarization.EnableRAG {
		go func(mID int, mContent, modelName, userID string) {
			bgCtx := context.Background()
			ctx, cancel := context.WithTimeout(bgCtx, 30*time.Second)
			defer cancel()
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":      filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":      line,
				"messageId": mID,
			}).Info("Generating embedding for message (memory API)")
			embedding, err := proxy.GetEmbedding(ctx, mContent, modelName)
			if err != nil {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":      filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":      line,
					"messageId": mID,
					"error":     err,
				}).Error("Error generating embedding for message (memory API)")
				return
			}
			if len(embedding) == 0 {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":      filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":      line,
					"messageId": mID,
				}).Warn("Warning: Got empty embedding for message (memory API)")
				return
			}
			_, file, line, _ = runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":       filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":       line,
				"messageId":  mID,
				"vectorSize": len(embedding),
			}).Info("Storing embedding for message in memories table")
			err = StoreMemory(ctx, userID, "message", messageID, embedding)
			if err != nil {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":      filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":      line,
					"messageId": mID,
					"error":     err,
				}).Error("Error storing memory embedding for message")
			}
		}(messageID, content, profile.ModelName, usrCfg.UserID)
	}

	return messageID, nil
}

// GetMessage gets a single message by ID
func GetMessage(ctx context.Context, messageID int) (*Message, error) {
	// Try to get from cache first
	if msg, found := GetMessageFromCache(ctx, messageID); found {
		return msg, nil
	}

	// Not in cache, get from database using the query from our loader
	var msg Message
	err := Pool.QueryRow(ctx, GetQuery("message.get_message"), messageID).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Cache for future use
	if err := CacheMessage(ctx, &msg); err != nil {
		// Just log, don't fail on cache error
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":      filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":      line,
			"messageId": msg.ID,
			"error":     err,
		}).Warn("Warning: Failed to cache message")
	}

	return &msg, nil
}

// GetConversationHistory retrieves all messages for a conversation
func GetConversationHistory(ctx context.Context, conversationID int) ([]Message, error) {
	// Try to get from cache first
	if messages, found := GetMessagesByConversationIDFromCache(ctx, conversationID); found {
		return messages, nil
	}

	// Not in cache, get from database using the query from our loader
	rows, err := Pool.Query(ctx, GetQuery("message.get_conversation_history"), conversationID)
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
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":           line,
			"conversationId": conversationID,
			"error":          err,
		}).Warn("Failed to cache messages for conversation")
	}

	return messages, nil
}
