// Package context provides conversation context management
package context

import (
	"context"
	"fmt"
	"path/filepath"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/storage"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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

	// Ensure the last message is a user message with the critique instruction
	if len(msgs) == 0 || msgs[len(msgs)-1].Role != "user" {
		msgs = append(msgs, models.OllamaMessage{
			Role:    "user",
			Content: "Please critique the above AI response.",
		})
	}

	// Set a timeout for critique generation
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Send the request to Ollama
	resp, err := proxy.StreamOllamaRequest(timeoutCtx, critiqueProfile.ModelName, msgs, "/api/generate")
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
	resp, err := proxy.StreamOllamaRequest(timeoutCtx, improvementProfile.ModelName, msgs, "/api/generate")
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

func RefineResponse(response, userMessage, userID string, conversationID int) string {
	cfg, err := GetUserConfig(userID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Error("Failed to get user config for response refinement")
		return ""
	}

	cc, err := GetCachedConversation(userID, conversationID)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Error("Failed to get cached conversation for response refinement")
		return ""
	}
	// Create a new background context for this goroutine
	// This is crucial to avoid using the request context which may be canceled
	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, time.Minute*10)
	defer cancel()

	refinedRes := response // Start with the original response

	// Step 1: Apply basic filtering if enabled
	if cfg.Summarization.EnableResponseFiltering {
		refinedRes = FilterResponseText(refinedRes)
	}

	// Step 2: Apply self-critique if enabled
	if cfg.Summarization.EnableResponseCritique {
		// Use the fresh background context and conversation context
		critique, err := cc.GetCritiqueForResponse(ctx, refinedRes)
		if err == nil && critique != "" {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":   line,
				"length": len(critique),
			}).Info("Got critique for response")
			improvedRes, err := cc.ImproveResponseWithCritique(ctx, userMessage, refinedRes, critique)
			if err != nil {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"error": err,
				}).Error("Failed to improve response with critique")
			} else if improvedRes != "" {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line": line,
				}).Info("Applied critique improvements to response")
				refinedRes = improvedRes
			}
		} else if err != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":  line,
				"error": err,
			}).Error("Failed to get critique")
		}
	}

	// Store the refined response
	var finalRes string
	if refinedRes != response {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Response was refined/improved before storing")
		finalRes = refinedRes
	} else {
		finalRes = response
	}

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":   line,
		"length": len(finalRes),
	}).Info("Storing assistant response")

	if err := cc.AddAssistantMessage(ctx, finalRes); err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Error("Error storing assistant response")
	} else {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":           line,
			"conversationId": cc.ConversationID,
		}).Info("Successfully stored assistant message in conversation")
	}

	return finalRes
}
