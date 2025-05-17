package context

import (
	"context"
	"path/filepath"
	"proxyllama/models"
	"proxyllama/proxy"
	"runtime"
	"time"

	"encoding/json"

	"github.com/sirupsen/logrus"
)

// generateSummarization creates a summary using Ollama
func (cc *ConversationContext) generateSummarization(messages []models.Message, summaryModel *models.ModelProfile) (string, error) {
	// Build Ollama messages
	ollamaMessages := []models.OllamaMessage{
		{
			Role:    "system",
			Content: summaryModel.SystemPrompt,
		},
	}

	// Add user messages
	for _, message := range messages {
		ollamaMessages = append(ollamaMessages, models.OllamaMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	// Ensure the last message is a user message with the summarization instruction
	if len(ollamaMessages) == 0 || ollamaMessages[len(ollamaMessages)-1].Role != "user" {
		ollamaMessages = append(ollamaMessages, models.OllamaMessage{
			Role:    "user",
			Content: summaryModel.SystemPrompt, // Use the system prompt as the summarization instruction
		})
	}

	// Debug: log the full request payload as JSON
	reqObj := models.OllamaReq{
		Model:    summaryModel.ModelName,
		Messages: ollamaMessages,
		Stream:   true,
	}
	if reqJson, err := json.MarshalIndent(reqObj, "", "  "); err == nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Debugf("Ollama summary request payload: %s", string(reqJson))
	}

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"model": summaryModel.ModelName,
	}).Info("Using model for text generation")

	// Create a long-lived context for generation
	longCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	resp, err := proxy.StreamOllamaRequest(longCtx, summaryModel.ModelName, ollamaMessages, "/api/chat")
	// Debug: log the raw response from Ollama
	_, file2, line2, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file2)), filepath.Base(file2)),
		"line": line2,
		"resp": resp,
		"err":  err,
	}).Debug("Ollama summary response")
	return resp, err
}
