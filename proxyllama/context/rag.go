package context

import (
	"context"
	"errors"
	"fmt"
	"proxyllama/models"
	"proxyllama/storage"
	"proxyllama/util"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// RetrievedMemory represents a message that was retrieved based on vector similarity
type RetrievedMemory struct {
	Message    models.Memory
	Similarity float32
}

// EnhanceRequestWithRAG adds relevant memories to the request based on the latest user query
func (cc *ConversationContext) EnhanceRequestWithRAG(ctx context.Context, req *models.OllamaChatReq) error {
	// Find the latest user message to use as query
	var latestUserMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			latestUserMessage = req.Messages[i].Content
			break
		}
	}

	if latestUserMessage == "" {
		return fmt.Errorf("no user message found in request")
	}

	if len(cc.RetrievedMemories) == 0 {
		return nil // No relevant memories found, continue with original request
	}

	// Create a new request with memories inserted at the right position
	var enhancedMessages []models.OllamaChatMessage
	var relevantMemories []models.OllamaChatMessage
	for _, mem := range cc.RetrievedMemories {
		relevantMemories = append(relevantMemories, models.OllamaChatMessage{
			Role:    mem.Role,
			Content: mem.Content,
		})
	}

	// First add any system messages (these should come first)
	systemMessagesCount := 0
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			enhancedMessages = append(enhancedMessages, msg)
			systemMessagesCount++
		}
	}

	// Add a system message to explain the memories
	enhancedMessages = append(enhancedMessages, models.OllamaChatMessage{
		Role:    "system",
		Content: fmt.Sprintf("Here are %d relevant memories from previous conversations that may help answer the current query:", len(relevantMemories)),
	})

	// Add the retrieved memories
	enhancedMessages = append(enhancedMessages, relevantMemories...)

	// Add another system message to separate memories from the current conversation
	enhancedMessages = append(enhancedMessages, models.OllamaChatMessage{
		Role:    "system",
		Content: "Now continuing with the current conversation:",
	})

	// Add the non-system messages from the original request
	for i, msg := range req.Messages {
		if i >= systemMessagesCount {
			enhancedMessages = append(enhancedMessages, msg)
		}
	}

	// Replace messages in the request
	req.Messages = enhancedMessages

	util.LogInfo("Enhanced request with relevant memories", logrus.Fields{"count": len(relevantMemories)})
	return nil
}

// RetrieveAndInjectMemories retrieves relevant memories based on the current user query
func (cc *ConversationContext) RetrieveAndInjectMemories(ctx context.Context, queryEmbeddings [][]float32) error {
	// Clear any previous memories
	cc.RetrievedMemories = nil

	// Get user-specific configuration
	userConfig, err := GetUserConfig(cc.UserID)
	if err != nil {
		return util.HandleError(err)
	}

	// Set a search limit from user config
	limit := userConfig.Retrieval.Limit
	if limit <= 0 {
		limit = 5
	}

	// First try vector similarity search if RAG is enabled
	if userConfig.Summarization.EnableRAG {
		util.LogInfo("Performing semantic search for memories")
		if len(queryEmbeddings) > 0 {
			var wg sync.WaitGroup

			// Channel for current conversation results
			currentConvResults := make(chan []models.Message, 1)
			currentConvErrors := make(chan error, 1)
			defer close(currentConvResults)
			defer close(currentConvErrors)

			// Only create cross-conv channels if needed
			var crossConvResults chan []models.Message
			var crossConvErrors chan error
			crossConvEnabled := userConfig.Retrieval.EnableCrossConversation
			if crossConvEnabled {
				crossConvResults = make(chan []models.Message, 1)
				crossConvErrors = make(chan error, 1)
				defer close(crossConvResults)
				defer close(crossConvErrors)
			}

			// Start search in current conversation
			wg.Add(1)
			go func(cid, limit int, embeddings [][]float32) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
				defer cancel()
				ccr := []models.Message{}

				var errorStr string
				var memoryError error
				for _, emb := range embeddings {
					similarMessages, err := storage.SearchMessagesBySimilarity(ctx, cid, emb, limit)
					if err != nil {
						errorStr += err.Error() + " "
					}
					for _, msg := range similarMessages {
						ccr = append(ccr, models.Message{
							Role:    msg.Role,
							Content: msg.Content,
							ID:      msg.ID,
						})
						if len(ccr) >= limit {
							break
						}
					}
					if errorStr != "" {
						memoryError = errors.New(errorStr)
					}
				}
				currentConvResults <- ccr
				currentConvErrors <- memoryError
			}(cc.ConversationID, limit, queryEmbeddings)

			threshold := userConfig.Retrieval.SimilarityThreshold
			// Start cross-conversation search if enabled
			if crossConvEnabled {
				wg.Add(1)
				go func(threshold float64, limit int, embeddings [][]float32) {
					defer wg.Done()
					ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
					defer cancel()
					if threshold <= 0 {
						threshold = 0.7 // Default threshold
					}

					ccr := []models.Message{}

					var errorStr string
					var memoryError error
					for _, embedding := range embeddings {
						similarMessages, err := storage.SearchAllMessagesBySimilarity(ctx, embedding, threshold, limit)
						if err != nil {
							errorStr += err.Error() + " "
						}

						for _, msg := range similarMessages {
							ccr = append(ccr, models.Message{
								Role:    msg.Role,
								Content: msg.Content,
								ID:      msg.ID,
							})
							if len(ccr) >= limit {
								break
							}
						}
					}
					if errorStr != "" {
						memoryError = errors.New(errorStr)
					}
					crossConvResults <- ccr
					crossConvErrors <- memoryError
				}(threshold, limit, queryEmbeddings)
			}

			// Wait for all started goroutines
			wg.Wait()

			// Collect results
			currentMsgs := <-currentConvResults
			currentErr := <-currentConvErrors

			if currentErr != nil {
				return util.HandleError(currentErr)
			}

			var crossMsgs []models.Message
			var crossErr error
			if crossConvEnabled {
				crossMsgs = <-crossConvResults
				crossErr = <-crossConvErrors
				if crossErr != nil {
					return util.HandleError(crossErr)
				}
			}

			util.LogInfo(fmt.Sprintf("Found %v semantically similar messages in current conversation", len(currentMsgs)))
			// 1. First add current conversation messages
			if len(currentMsgs) > 0 {
				cc.appendMemoriesToContext(currentMsgs)
			}

			util.LogInfo(fmt.Sprintf("Found %v semantically similar messages across conversations", len(crossMsgs)))
			// 2. Then add cross-conversation messages
			if crossConvEnabled && len(crossMsgs) > 0 {
				cc.appendMemoriesToContext(crossMsgs)
			}

			if len(cc.RetrievedMemories) > 0 {
				return nil
			}
		}
	}

	return nil
}

func (cc *ConversationContext) appendMemoriesToContext(memories []models.Message) {
	// Add retrieved memories to the context
	for _, mem := range memories {
		formattedContent := fmt.Sprintf(
			"From conversation #%d (%s): %s",
			cc.ConversationID,
			mem.Role,
			mem.Content,
		)
		cc.RetrievedMemories = append(cc.RetrievedMemories, models.Memory{
			Role:    mem.Role,
			Content: formattedContent,
			ID:      mem.ID,
		})
	}
}
