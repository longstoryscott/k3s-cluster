// Package context provides conversation context management
package context

import (
	"context"
	"fmt"
	"proxyllama/config"
	"proxyllama/models"
	"proxyllama/proxy"
	"strings"
	"time"
)

// getCritiqueForResponse sends a response to Ollama for critique
func GetCritiqueForResponse(ctx context.Context, responseToCritique, modelName string) (string, error) {
	conf := config.GetConfig()
	// Use a more compact model for critiquing if available
	critiqueModel := conf.Summarization.CritiqueModel
	if critiqueModel == "" {
		critiqueModel = modelName // Fall back to the same model
	}

	// Create the critique system prompt
	critiquePrompt := "You are an expert critique assistant. Your task is to analyze the following AI response and identify:" +
		"\n1. Factual inaccuracies or potential errors" +
		"\n2. Areas where clarity could be improved" +
		"\n3. Opportunities to make the response more helpful or comprehensive" +
		"\n4. Any redundancies or unnecessary content" +
		"\nBe concise and focus on actionable feedback that can improve the response."

	msgs := []models.OllamaMessage{
		{
			Role:    "system",
			Content: critiquePrompt,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Please critique this AI response: \n\n%s", responseToCritique),
		},
	}

	// Set a timeout for critique generation
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Send the request to Ollama
	resp, err := proxy.SendOllamaRequest(timeoutCtx, critiqueModel, msgs, "/api/generate")
	if err != nil {
		return "", fmt.Errorf("failed to get critique: %w", err)
	}

	if resp == "" {
		return "", fmt.Errorf("empty critique response")
	}

	return resp, nil
}

// ImproveResponseWithCritique improves a response based on the critique
func ImproveResponseWithCritique(ctx context.Context, originalQuery, originalResponse, critiqueText, modelName string) (string, error) {
	conf := config.GetConfig()
	// Use a more compact model for critiquing if available
	critiqueModel := conf.Summarization.CritiqueModel
	if critiqueModel == "" {
		critiqueModel = modelName // Fall back to the same model
	}
	// Use the same model for improvement
	// Build the improvement system prompt
	improvementPrompt := "Your task is to improve the original AI response based on the critique provided. " +
		"Maintain the overall structure and intent of the original response, but address the issues identified in the critique. " +
		"The improved response should be clear, accurate, concise, and directly answer the user's original query."

	// Build the request
	msgs := []models.OllamaMessage{
		{
			Role:    "system",
			Content: improvementPrompt,
		},
		{
			Role: "user",
			Content: fmt.Sprintf("Original query: %s\n\nOriginal response: %s\n\nCritique: %s\n\nPlease provide an improved response addressing the critique:",
				originalQuery, originalResponse, critiqueText),
		},
	}

	// Set a timeout for response improvement
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Send the request to Ollama
	resp, err := proxy.SendOllamaRequest(timeoutCtx, critiqueModel, msgs, "/api/generate")
	if err != nil {
		return "", fmt.Errorf("failed to improve response: %w", err)
	}

	if resp == "" {
		return originalResponse, nil // Fall back to original if improvement failed
	}

	return resp, nil
}

// FilterResponseText applies basic filtering rules to clean up the response text
func FilterResponseText(text string) string {
	// Basic filtering - remove repetitive phrases and other cleanup
	replacements := map[string]string{
		"I am unable to browse URLs. ":                      "",
		"I don't have the ability to browse the internet. ": "",
		"As an AI language model, ":                         "",
		"As an AI assistant, ":                              "",
		"I'm sorry, but ":                                   "Sorry, ",
		"I apologize, but ":                                 "Sorry, ",
		"\n\n\n":                                            "\n\n", // Cleanup excessive line breaks
	}

	result := text
	for phrase, replacement := range replacements {
		result = strings.ReplaceAll(result, phrase, replacement)
	}

	return result
}
