package storage

import (
	"context"
	"fmt"
	"time"
)

type Conversation struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SQL query templates for conversation operations
const (
	sqlCreateConversation = `
		INSERT INTO conversations (user_id, model, title) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`
	sqlGetConversation = `
		SELECT id, user_id, title, model, created_at, updated_at 
		FROM conversations 
		WHERE id = $1
	`
	sqlGetUserConversations = `
		SELECT id, user_id, title, model, created_at, updated_at 
		FROM conversations 
		WHERE user_id = $1 
		ORDER BY updated_at DESC
	`
	sqlGetConversationHistory = `
		SELECT id, conversation_id, role, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`
	sqlUpdateConversationTitle = `
		UPDATE conversations 
		SET title = $1, updated_at = NOW() 
		WHERE id = $2
	`
	sqlDeleteConversation = `
		DELETE FROM conversations WHERE id = $1
	`
)

// CreateConversation starts a new conversation for a user
func CreateConversation(ctx context.Context, userID, model string, title string) (int, error) {
	// Ensure the user exists
	if err := EnsureUser(ctx, userID); err != nil {
		return 0, err
	}

	var conversationID int
	err := Pool.QueryRow(ctx, sqlCreateConversation, userID, model, title).Scan(&conversationID)
	return conversationID, err
}

// GetConversationHistory retrieves all messages for a conversation
func GetConversationHistory(ctx context.Context, conversationID int) ([]Message, error) {
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

	return messages, nil
}

// GetUserConversations gets all conversations for a user, ordered by most recent
func GetUserConversations(ctx context.Context, userID string) ([]Conversation, error) {
	rows, err := Pool.Query(ctx, sqlGetUserConversations, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		if err := rows.Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Model, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, err
		}
		conversations = append(conversations, conv)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

// GetConversation retrieves a single conversation by ID
func GetConversation(ctx context.Context, conversationID int) (*Conversation, error) {
	var conv Conversation
	err := Pool.QueryRow(ctx, sqlGetConversation, conversationID).Scan(
		&conv.ID, &conv.UserID, &conv.Title, &conv.Model, &conv.CreatedAt, &conv.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &conv, nil
}

// UpdateConversationTitle updates the title of a conversation
func UpdateConversationTitle(ctx context.Context, conversationID int, title string) error {
	_, err := Pool.Exec(ctx, sqlUpdateConversationTitle, title, conversationID)
	return err
}

// DeleteConversation deletes a conversation and all its messages using transaction
func DeleteConversation(ctx context.Context, conversationID int) error {
	// Start a transaction for atomicity
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer a rollback in case anything fails
	defer tx.Rollback(ctx)

	// Delete the conversation (triggers will handle dependent records)
	_, err = tx.Exec(ctx, sqlDeleteConversation, conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
