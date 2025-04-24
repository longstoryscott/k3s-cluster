package storage

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

type Message struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	Role           string    `json:"role"` // "user" or "assistant"
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChunkResponse represents the structure of a single chunk in the stream
type ChunkResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// ContentAggregator maintains state between chunk processing
type ContentAggregator struct {
	mu           sync.Mutex
	sessions     map[string]*SessionContent
	cleanupTimer *time.Timer
}

// SessionContent represents content for a specific session
type SessionContent struct {
	Content     string
	LastUpdated time.Time
}

var (
	contentAggregator *ContentAggregator
	once              sync.Once = sync.Once{}
)

// GetContentAggregator creates a new content aggregator with cleanup timer
func GetContentAggregator() *ContentAggregator {
	if contentAggregator != nil {
		return contentAggregator
	}

	once.Do(func() {
		contentAggregator = &ContentAggregator{
			sessions: make(map[string]*SessionContent),
		}

		// Start cleanup timer to remove old sessions every 5 minutes
		contentAggregator.cleanupTimer = time.AfterFunc(5*time.Minute, func() {
			contentAggregator.cleanup()
		})
	})
	return contentAggregator

}

// HandleChunk processes a chunk and aggregates content
func (ca *ContentAggregator) HandleChunk(sessionID string, chunk []byte) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Create or get session
	session, exists := ca.sessions[sessionID]
	if !exists {
		session = &SessionContent{}
		ca.sessions[sessionID] = session
	}

	// Update last updated time
	session.LastUpdated = time.Now()

	// Process the chunk data - it might be in Server-Sent Events format
	// Each SSE message starts with "data: " and ends with "\n\n"
	str := string(chunk)
	log.Printf("Processing chunk: %s", str)

	// Try to parse as a ChunkResponse
	var response ChunkResponse
	err := json.Unmarshal(chunk, &response)
	if err != nil {
		log.Printf("Error parsing chunk: %v, data: %s", err, str)
		return
	}

	// Aggregate content
	if response.Message.Content != "" {
		session.Content += response.Message.Content
	}

	// Check if done
	if response.Done {
		log.Printf("Completed aggregating content for session %s", sessionID)
		log.Printf("Final content: %s", session.Content)
		return
	}
}

// GetAggregatedContent returns the current aggregated content for a session
func (ca *ContentAggregator) GetAggregatedContent(sessionID string) string {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if session, exists := ca.sessions[sessionID]; exists {
		return session.Content
	}
	return ""
}

// cleanup removes sessions older than 30 minutes
func (ca *ContentAggregator) cleanup() {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	cutoff := time.Now().Add(-30 * time.Minute)
	for id, session := range ca.sessions {
		if session.LastUpdated.Before(cutoff) {
			delete(ca.sessions, id)
		}
	}

	// Reset timer
	ca.cleanupTimer.Reset(5 * time.Minute)
}

// AddMessage adds a message to a conversation
func AddMessage(ctx context.Context, conversationID int, role, content string) (int, error) {
	var messageID int
	err := DB.QueryRow(ctx, `
        INSERT INTO messages (conversation_id, role, content) 
        VALUES ($1, $2, $3) 
        RETURNING id
    `, conversationID, role, content).Scan(&messageID)

	// Update the conversation's updated_at timestamp
	if err == nil {
		_, err = DB.Exec(ctx, `
            UPDATE conversations 
            SET updated_at = NOW() 
            WHERE id = $1
        `, conversationID)
	}

	return messageID, err
}
