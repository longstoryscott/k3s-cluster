package context

import (
	"context"
	"log"
	"proxyllama/config"
	"proxyllama/models"
	"proxyllama/proxy"
	"time"
)

// generateText creates a summary using Ollama
func (cc *ConversationContext) generateText(ctx context.Context, messages []models.Message, systemPrompt, summaryModel string) (string, error) {
	// If summaryModel is empty, use the global SummaryModel variable
	if summaryModel == "" {
		summaryModel = config.SummaryModel
	}

	// As a fallback, check the configuration
	if summaryModel == "" {
		conf := config.GetConfig()
		if conf.Summarization.SummaryModel != "" {
			summaryModel = conf.Summarization.SummaryModel
		} else {
			// Default to qwen3:0.6b if no model is specified
			summaryModel = "qwen3:0.6b"
			log.Printf("No summary model specified, using default: %s", summaryModel)
		}
	}

	// Build Ollama messages
	ollamaMessages := []models.OllamaMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Add user messages
	for _, message := range messages {
		ollamaMessages = append(ollamaMessages, models.OllamaMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	// debug log the content of each message in the request
	for _, msg := range ollamaMessages {
		log.Printf("Message: %s", msg.Content)
	}

	log.Printf("Using model %s for text generation", summaryModel)

	// Create a long-lived context for generation
	longCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	return proxy.SendOllamaRequest(longCtx, summaryModel, ollamaMessages, "/api/chat")
}
