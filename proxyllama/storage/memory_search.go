package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Message represents a message with its metadata for search results
type SearchMessage struct {
	ID             int
	Role           string
	Content        string
	ConversationID int
	Similarity     float32 // Used for vector similarity search results
}

// SearchMessagesByKeyword searches for messages containing keywords in a conversation
func SearchMessagesByKeyword(ctx context.Context, conversationID int, query string, limit int) ([]SearchMessage, error) {
	// Format the query for PostgreSQL's to_tsquery
	tsQuery := formatTSQuery(query)
	likeQuery := "%" + query + "%"

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":    filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":    line,
		"query":   query,
		"tsQuery": tsQuery,
	}).Info("Searching for messages with keyword query")

	rows, err := Pool.Query(ctx, GetQuery("memory.search_conversation_keyword"), conversationID, tsQuery, likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages by keyword: %w", err)
	}
	defer rows.Close()

	var results []SearchMessage
	for rows.Next() {
		var msg SearchMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"count": len(results),
	}).Info("Found messages matching keyword query")
	return results, nil
}

// SearchAllMessagesByKeyword searches for messages containing keywords across all conversations
func SearchAllMessagesByKeyword(ctx context.Context, userID string, query string, limit int) ([]SearchMessage, error) {
	// Format the query for PostgreSQL's to_tsquery
	tsQuery := formatTSQuery(query)
	likeQuery := "%" + query + "%"

	rows, err := Pool.Query(ctx, GetQuery("memory.search_all_keyword"), userID, tsQuery, likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search all messages by keyword: %w", err)
	}
	defer rows.Close()

	var results []SearchMessage
	for rows.Next() {
		var msg SearchMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	return results, nil
}

// SearchMessagesBySimilarity searches for semantically similar messages in a conversation
func SearchMessagesBySimilarity(ctx context.Context, conversationID int, embedding []float32, limit int) ([]SearchMessage, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	rows, err := Pool.Query(ctx, GetQuery("memory.search_conversation_similarity"), conversationID, embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages by similarity: %w", err)
	}
	defer rows.Close()

	var results []SearchMessage
	for rows.Next() {
		var msg SearchMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID, &msg.Similarity); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"count": len(results),
	}).Info("Found messages by vector similarity")
	return results, nil
}

// SearchAllMessagesBySimilarity searches for semantically similar messages across all conversations
func SearchAllMessagesBySimilarity(ctx context.Context, embedding []float32, minSimilarity float64, limit int) ([]SearchMessage, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	rows, err := Pool.Query(ctx, GetQuery("memory.search_all_similarity"), embedding, minSimilarity, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search all messages by similarity: %w", err)
	}
	defer rows.Close()

	var results []SearchMessage
	for rows.Next() {
		var msg SearchMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ConversationID, &msg.Similarity); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		results = append(results, msg)
	}

	return results, nil
}

// formatTSQuery formats a text query for PostgreSQL's to_tsquery
// It replaces spaces with '&' for AND operations and handles basic query formatting
func formatTSQuery(query string) string {
	// Remove special characters that might cause issues with to_tsquery
	query = strings.ReplaceAll(query, "'", "")

	// Split the query into words
	words := strings.Fields(query)

	// Filter out stop words and words that are too short
	var filteredWords []string
	for _, word := range words {
		if len(word) > 2 { // Skip words that are too short
			filteredWords = append(filteredWords, word+":*") // Add stemming
		}
	}

	// If no valid words, return a simple query
	if len(filteredWords) == 0 {
		return query
	}

	// Join with & for AND operation
	return strings.Join(filteredWords, " & ")
}

// SearchMessagesText performs a full-text search on message content
func SearchMessagesText(ctx context.Context, query string, limit int) ([]Message, error) {
	// No caching for search results, always query the database
	rows, err := Pool.Query(ctx, GetQuery("memory.text_search"), query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var rank float32 // Ranking score

		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt, &rank); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return messages, nil
}

// SearchMessagesTextInConversation performs a full-text search on message content within a conversation
func SearchMessagesTextInConversation(ctx context.Context, query string, limit int, conversationID int) ([]Message, error) {
	// No caching for search results, always query the database
	rows, err := Pool.Query(ctx, GetQuery("memory.conversation_text_search"), query, limit, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute conversation search query: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var rank float32 // Ranking score

		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt, &rank); err != nil {
			return nil, fmt.Errorf("failed to scan conversation search result: %w", err)
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversation search results: %w", err)
	}

	return messages, nil
}

// SearchMessagesCombined performs a hybrid search using both vector similarity and text search
func SearchMessagesCombined(ctx context.Context, embedding []float32, textQuery string, limit int) ([]Message, error) {
	// Format the embedding as a string for PostgreSQL
	embeddingStr := formatEmbeddingForPgVector(embedding)

	// No caching for search results, always query the database
	rows, err := Pool.Query(ctx, GetQuery("memory.combined_search"), embeddingStr, textQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute combined search query: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var score float32 // Combined score

		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt, &score); err != nil {
			return nil, fmt.Errorf("failed to scan combined search result: %w", err)
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating combined search results: %w", err)
	}

	return messages, nil
}

// SearchMessagesCombinedInConversation performs a hybrid search within a conversation
func SearchMessagesCombinedInConversation(ctx context.Context, embedding []float32, textQuery string, limit int, conversationID int) ([]Message, error) {
	// Format the embedding as a string for PostgreSQL
	embeddingStr := formatEmbeddingForPgVector(embedding)

	// No caching for search results, always query the database
	rows, err := Pool.Query(ctx, GetQuery("memory.conversation_combined_search"), embeddingStr, textQuery, limit, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute combined conversation search query: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var score float32 // Combined score

		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt, &score); err != nil {
			return nil, fmt.Errorf("failed to scan combined conversation search result: %w", err)
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating combined conversation search results: %w", err)
	}

	return messages, nil
}

// ContextualSearch searches for relevant messages using the most appropriate method
// based on available parameters. It handles all search logic.
func ContextualSearch(ctx context.Context, query string, embedding []float32, conversationID *int, limit int) ([]Message, error) {
	var err error
	var messages []Message

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":     filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":     line,
			"duration": duration,
			"count":    len(messages),
		}).Info("ContextualSearch completed")
	}()

	// Set default limit if not specified or invalid
	if limit <= 0 {
		limit = 10
	}

	// Determine search method based on available parameters
	switch {
	// Case 1: Embedding and text query available - use combined search
	case embedding != nil && query != "":
		if conversationID != nil {
			messages, err = SearchMessagesCombinedInConversation(ctx, embedding, query, limit, *conversationID)
		} else {
			messages, err = SearchMessagesCombined(ctx, embedding, query, limit)
		}
		if err != nil {
			return nil, fmt.Errorf("combined search failed: %w", err)
		}

	// Case 2: Only embedding available - use vector search
	case embedding != nil:
		if conversationID != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line": line,
			}).Warn("Vector-only search within conversation not implemented, using global search")
		}
		messages, err = GetSimilarMessages(ctx, embedding, 0.4, limit)
		if err != nil {
			return nil, fmt.Errorf("vector search failed: %w", err)
		}

	// Case 3: Only text query available - use text search
	case query != "":
		if conversationID != nil {
			messages, err = SearchMessagesTextInConversation(ctx, query, limit, *conversationID)
		} else {
			messages, err = SearchMessagesText(ctx, query, limit)
		}
		if err != nil {
			return nil, fmt.Errorf("text search failed: %w", err)
		}

	// Case 4: No query or embedding - return error
	default:
		return nil, fmt.Errorf("search requires either a text query or an embedding")
	}

	return messages, nil
}
