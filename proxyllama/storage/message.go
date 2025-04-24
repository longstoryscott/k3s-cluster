package storage

import (
	"context"
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

	return messageID, nil
}
