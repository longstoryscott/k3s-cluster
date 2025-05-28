package context

import (
	"context"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/util"
	"time"

	"github.com/sirupsen/logrus"
)

// generateSummarization creates a summary using Ollama
func (cc *ConversationContext) generateSummarization(messages []models.Message, summaryModel *models.ModelProfile) (string, error) {
	// Build Ollama messages
	ollamaMessages := []models.OllamaChatMessage{
		{
			Role:    "system",
			Content: summaryModel.SystemPrompt,
		},
	}

	// Add user messages
	for _, message := range messages {
		ollamaMessages = append(ollamaMessages, models.OllamaChatMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	// Ensure the last message is a user message with the summarization instruction
	if len(ollamaMessages) == 0 || ollamaMessages[len(ollamaMessages)-1].Role != "user" {
		ollamaMessages = append(ollamaMessages, models.OllamaChatMessage{
			Role:    "user",
			Content: summaryModel.SystemPrompt, // Use the system prompt as the summarization instruction
		})
	}

	util.LogInfo("Using model for text generation", logrus.Fields{"model": summaryModel.ModelName})

	// Create a long-lived context for generation
	longCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	resp, err := proxy.StreamOllamaChatRequest(longCtx, summaryModel.ModelName, ollamaMessages)
	util.LogDebug("Ollama summary response", logrus.Fields{"resp": resp, "err": err})
	return resp, err
}

// PrepareOllamaRequest prepares the request for Ollama
func (cc *ConversationContext) PrepareOllamaRequest(ctx context.Context, chatReq models.OllamaChatReq) ([]byte, error) {
	// Get the user message from the request (last message from user)
	var userMessage string
	for i := len(chatReq.Messages) - 1; i >= 0; i-- {
		if chatReq.Messages[i].Role == "user" {
			userMessage = chatReq.Messages[i].Content
			break
		}
	}

	embedding, err := cc.AddUserMessage(ctx, userMessage)
	if err != nil {
		return nil, util.HandleError(err)
	}

	// Get the user's intent to determine if we should perform a web search or retrieve memories
	intent, err := cc.DetectIntent(ctx, userMessage)
	if err != nil {
		util.LogWarning("Error detecting intent", logrus.Fields{"error": err})
		// Non-critical error, we can continue without intent
	} else if intent != nil {
		// If the intent indicates a web search, get the web search results
		if intent.WebSearch {
			// Inject the search results into the conversation context
			if err := cc.SearchAndInjectResults(ctx, userMessage); err != nil {
				util.LogWarning("Error injecting web search results", logrus.Fields{"error": err})
				// Non-critical error, we can continue without web search results
			}
		}

		if intent.Memory {
			// Attempt to retrieve and inject relevant memories for the user's query
			if err := cc.RetrieveAndInjectMemories(ctx, embedding); err != nil {
				util.LogWarning("Error retrieving memories", logrus.Fields{"error": err})
				// Non-critical error, we can continue without memories
			}
		}
	}

	// If embeddings are empty, log a warning and return an empty JSON object
	if len(embedding) == 0 {
		util.LogWarning("Empty embedding vector", logrus.Fields{"userMessage": userMessage})
		return []byte("{}"), nil
	}

	// Convert conversation context to Ollama format (includes summaries)
	return cc.ToJSON()
}
