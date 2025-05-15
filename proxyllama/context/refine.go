// Package context provides conversation context management
package context

import (
	"context"
	"fmt"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/storage"
	"strings"
	"time"
)

// getCritiqueForResponse sends a response to Ollama for critique
func (cc *ConversationContext) GetCritiqueForResponse(ctx context.Context, responseToCritique string) (string, error) {
	cfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return "", fmt.Errorf("failed to get user config: %w", err)
	}
	critiqueProfile, err := storage.GetModelProfile(ctx, cfg.ModelProfiles.SelfCritiqueProfileID)
	if err != nil {
		return "", fmt.Errorf("failed to get self-critique profile: %w", err)
	}

	msgs := []models.OllamaMessage{
		{
			Role:    "system",
			Content: critiqueProfile.SystemPrompt,
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
	resp, err := proxy.SendOllamaRequest(timeoutCtx, critiqueProfile.ModelName, msgs, "/api/generate")
	if err != nil {
		return "", fmt.Errorf("failed to get critique: %w", err)
	}

	if resp == "" {
		return "", fmt.Errorf("empty critique response")
	}

	return resp, nil
}

// ImproveResponseWithCritique improves a response based on the critique
func (cc *ConversationContext) ImproveResponseWithCritique(ctx context.Context, originalQuery, originalResponse, critiqueText string) (string, error) {
	cfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		return "", fmt.Errorf("failed to get user config: %w", err)
	}
	improvementProfile, err := storage.GetModelProfile(ctx, cfg.ModelProfiles.ImprovementProfileID)
	if err != nil {
		return "", fmt.Errorf("failed to get self-critique profile: %w", err)
	}

	// Build the request
	msgs := []models.OllamaMessage{
		{
			Role:    "system",
			Content: improvementProfile.SystemPrompt,
		},
		{
			Role: "user",
			Content: fmt.Sprintf("Original query: %s\n\nOriginal response: %s\n\nCritique: %s\n\nPlease provide an improved response addressing the critique:",
				originalQuery, originalResponse, critiqueText),
		},
	}

	// Set a timeout for response improvement
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Send the request to Ollama
	resp, err := proxy.SendOllamaRequest(timeoutCtx, improvementProfile.ModelName, msgs, "/api/generate")
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
