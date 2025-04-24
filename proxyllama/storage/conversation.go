package storage

import (
	"context"
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

// CreateConversation starts a new conversation for a user
func CreateConversation(ctx context.Context, userID, model string, title string) (int, error) {
	// Ensure the user exists
	if err := EnsureUser(ctx, userID); err != nil {
		return 0, err
	}

	var conversationID int
	err := DB.QueryRow(ctx, `
        INSERT INTO conversations (user_id, model, title) 
        VALUES ($1, $2, $3) 
        RETURNING id
    `, userID, model, title).Scan(&conversationID)
	return conversationID, err
}

// GetConversationHistory retrieves all messages for a conversation
func GetConversationHistory(ctx context.Context, conversationID int) ([]Message, error) {
	rows, err := DB.Query(ctx, `
		SELECT role, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`, conversationID)
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

	return messages, nil
}

// GetUserConversations gets all conversations for a user, ordered by most recent
func GetUserConversations(ctx context.Context, userID string) ([]Conversation, error) {
	rows, err := DB.Query(ctx, `
        SELECT id, user_id, title, model, created_at, updated_at 
        FROM conversations 
        WHERE user_id = $1 
        ORDER BY updated_at DESC
    `, userID)
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

	return conversations, nil
}

// GetConversation retrieves a single conversation by ID
func GetConversation(ctx context.Context, conversationID int) (*Conversation, error) {
	var conv Conversation
	err := DB.QueryRow(ctx, `
        SELECT id, user_id, title, model, created_at, updated_at 
        FROM conversations 
        WHERE id = $1
    `, conversationID).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Model, &conv.CreatedAt, &conv.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &conv, nil
}

// UpdateConversationTitle updates the title of a conversation
func UpdateConversationTitle(ctx context.Context, conversationID int, title string) error {
	_, err := DB.Exec(ctx, `
        UPDATE conversations 
        SET title = $1, updated_at = NOW() 
        WHERE id = $2
    `, title, conversationID)
	return err
}

// DeleteConversation deletes a conversation and all its messages
func DeleteConversation(ctx context.Context, conversationID int) error {
	_, err := DB.Exec(ctx, `
        DELETE FROM messages 
        WHERE conversation_id = $1;
    `, conversationID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(ctx, `
		DELETE FROM conversations	
		WHERE id = $1;
	`, conversationID)
	return err
}
