package context

import (
	"context"
	"path/filepath"
	"proxyllama/models"
	"proxyllama/proxy"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// generateText creates a summary using Ollama
func (cc *ConversationContext) generateText(ctx context.Context, messages []models.Message, summaryModel *models.ModelProfile) (string, error) {
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

	// debug log the content of each message in the request
	for _, msg := range ollamaMessages {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Debugf("Message: %s", msg.Content)
	}

	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"model": summaryModel.ModelName,
	}).Info("Using model for text generation")

	// Create a long-lived context for generation
	longCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	return proxy.SendOllamaRequest(longCtx, summaryModel.ModelName, ollamaMessages, "/api/chat")
}
