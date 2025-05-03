package context

import (
	"context"
	"fmt"
	"log"
	"proxyllama/config"
	"proxyllama/models"
	"proxyllama/storage"
)

// shouldSummarize checks if we have enough messages to create a summary
func (cc *ConversationContext) shouldSummarize() bool {
	// We need at least N messages (where N is configurable) to create a summary
	return len(cc.Messages) >= config.GetConfig().Summarization.MessagesBeforeSummary
}

// SummarizeMessages creates a summary of the oldest messages
func (cc *ConversationContext) SummarizeMessages(ctx context.Context) (models.Summary, error) {
	conf := config.GetConfig()

	// Find messages that haven't been summarized yet
	unsummarizedMessages, messageIDs := cc.getUnsummarizedMessages(ctx)

	// Only summarize when we have enough unsummarized messages
	if len(unsummarizedMessages) < conf.Summarization.MessagesBeforeSummary {
		return models.Summary{}, nil // Not enough unsummarized messages to summarize
	}

	// Calculate how many messages to summarize (N/2)
	messagesToSummarize := conf.Summarization.MessagesBeforeSummary / 2
	if messagesToSummarize < 1 {
		messagesToSummarize = 1 // Always summarize at least 1 message
	}

	// Limit to actual number of available unsummarized messages
	if messagesToSummarize > len(unsummarizedMessages) {
		messagesToSummarize = len(unsummarizedMessages)
	}

	// Extract the oldest N/2 unsummarized messages to summarize
	var messagesToSummarizeContent []models.Message
	var messageIDsToSummarize []int

	for i := 0; i < messagesToSummarize; i++ {
		messagesToSummarizeContent = append(messagesToSummarizeContent, unsummarizedMessages[i])
		messageIDsToSummarize = append(messageIDsToSummarize, messageIDs[i])
	}

	log.Printf("Summarizing messages with IDs: %v", messageIDsToSummarize)

	// Generate the summary using the appropriate prompt
	summaryContent, err := cc.generateText(ctx, messagesToSummarizeContent, conf.Summarization.SystemPrompt, config.SummaryModel)
	if err != nil {
		return models.Summary{}, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Create a new summary record in the database
	summaryID, err := storage.CreateSummary(ctx, cc.ConversationID, summaryContent, 1, messageIDsToSummarize)
	if err != nil {
		return models.Summary{}, fmt.Errorf("failed to store summary: %w", err)
	}

	log.Printf("Created summary ID %d for messages %v", summaryID, messageIDsToSummarize)

	// Add the summary to our context
	summary := models.Summary{
		Content: summaryContent,
		Level:   1,
		ID:      summaryID,
	}
	cc.Summaries = append(cc.Summaries, summary)

	// Remove the summarized messages from our in-memory context
	cc.removeMessagesById(messageIDsToSummarize)

	// Check if we need to consolidate summaries at level 1
	if cc.shouldConsolidateLevel(1) {
		if err := cc.consolidateLevel(ctx, 1); err != nil {
			log.Printf("Failed to consolidate level 1 summaries: %v", err)
		}
	}

	return summary, nil
}

// getUnsummarizedMessages returns messages that haven't been summarized yet
func (cc *ConversationContext) getUnsummarizedMessages(ctx context.Context) ([]models.Message, []int) {
	// Create a set to track which message IDs have already been summarized
	summarizedMessageIDs := make(map[int]bool)

	// Get all summaries to determine which messages have already been summarized
	summaries, err := storage.GetSummariesForConversation(ctx, cc.ConversationID)
	if err == nil {
		// Collect all message IDs that have already been included in summaries
		for _, summary := range summaries {
			for _, msgID := range summary.SourceIDs {
				summarizedMessageIDs[msgID] = true
			}
		}
	}

	// Filter messages that haven't been summarized yet
	var unsummarizedMessages []models.Message
	var unsummarizedIDs []int
	for _, msg := range cc.Messages {
		if !summarizedMessageIDs[msg.ID] {
			unsummarizedMessages = append(unsummarizedMessages, msg)
			unsummarizedIDs = append(unsummarizedIDs, msg.ID)
		}
	}

	return unsummarizedMessages, unsummarizedIDs
}

// removeMessagesById removes messages with specific IDs from the conversation
func (cc *ConversationContext) removeMessagesById(ids []int) {
	var remainingMessages []models.Message
	for _, msg := range cc.Messages {
		shouldKeep := true
		for _, id := range ids {
			if msg.ID == id {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			remainingMessages = append(remainingMessages, msg)
		}
	}
	cc.Messages = remainingMessages
}

// shouldConsolidateLevel checks if we have exactly X summaries at a specific level
func (cc *ConversationContext) shouldConsolidateLevel(level int) bool {
	conf := config.GetConfig()
	levelCount := CountSummariesAtLevel(cc.Summaries, level)

	// We only consolidate when we have exactly X summaries at this level
	return levelCount == conf.Summarization.SummariesBeforeConsolidation
}

// consolidateLevel creates a summary of summaries at a specific level
func (cc *ConversationContext) consolidateLevel(ctx context.Context, level int) error {
	conf := config.GetConfig()

	// Get summaries for this level
	var summariesToConsolidate []models.Summary
	var summaryIDs []int

	// Filter summaries by level
	for _, summary := range cc.Summaries {
		if summary.Level == level {
			summariesToConsolidate = append(summariesToConsolidate, summary)
			summaryIDs = append(summaryIDs, summary.ID)
		}
	}

	// We only consolidate when we have EXACTLY X summaries
	if len(summariesToConsolidate) != conf.Summarization.SummariesBeforeConsolidation {
		return nil // Not exactly the right number of summaries to consolidate
	}

	log.Printf("Consolidating %d summaries at level %d", len(summariesToConsolidate), level)

	// Convert summaries to messages for the summary generator
	var messagesToSummarize []models.Message
	for _, summary := range summariesToConsolidate {
		messagesToSummarize = append(messagesToSummarize, models.Message{
			Role:    "system",
			Content: summary.Content,
			ID:      summary.ID,
		})
	}

	// Generate the next level summary
	nextLevel := level + 1
	summaryContent, err := cc.generateText(ctx, messagesToSummarize,
		fmt.Sprintf("Create a comprehensive summary of these level %d conversation summaries:", level), config.SummaryModel)
	if err != nil {
		return fmt.Errorf("failed to generate level %d summary: %w", nextLevel, err)
	}

	// Store the new summary in the database
	summaryID, err := storage.CreateSummary(ctx, cc.ConversationID, summaryContent, nextLevel, summaryIDs)
	if err != nil {
		return fmt.Errorf("failed to store level %d summary: %w", nextLevel, err)
	}

	log.Printf("Created level %d summary ID %d from %d level %d summaries",
		nextLevel, summaryID, len(summariesToConsolidate), level)

	// Add the new summary to our context
	newSummary := models.Summary{
		Content: summaryContent,
		Level:   nextLevel,
		ID:      summaryID,
	}
	cc.Summaries = append(cc.Summaries, newSummary)

	// Remove consolidated summaries from our in-memory context
	cc.removeSummariesByIdAndLevel(summaryIDs, level)

	// Check if we need to create/update a master summary
	maxSummaryLevel := conf.Summarization.MaxSummaryLevels
	if nextLevel == maxSummaryLevel {
		levelMaxCount := CountSummariesAtLevel(cc.Summaries, maxSummaryLevel)

		// If we have exactly X summaries at max level, create/update master summary
		if levelMaxCount == conf.Summarization.SummariesBeforeConsolidation {
			log.Printf("We have %d summaries at level %d, creating/updating master summary",
				levelMaxCount, maxSummaryLevel)

			if cc.MasterSummary == nil {
				if err := cc.createMasterSummary(ctx); err != nil {
					log.Printf("Failed to create master summary: %v", err)
				}
			} else {
				// Update the existing master summary
				if err := cc.updateMasterSummary(ctx); err != nil {
					log.Printf("Failed to update master summary: %v", err)
				}
			}
		} else {
			log.Printf("We have %d/%d summaries at level %d, not creating master summary yet",
				levelMaxCount, conf.Summarization.SummariesBeforeConsolidation, maxSummaryLevel)
		}
	}

	return nil
}

// removeSummariesByIdAndLevel removes summaries with specific IDs and level from the conversation
func (cc *ConversationContext) removeSummariesByIdAndLevel(ids []int, level int) {
	var updatedSummaries []models.Summary
	for _, s := range cc.Summaries {
		shouldKeep := true
		for _, id := range ids {
			if s.ID == id && s.Level == level {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			updatedSummaries = append(updatedSummaries, s)
		}
	}
	cc.Summaries = updatedSummaries
}

// createMasterSummary generates a weighted summary of all summaries
func (cc *ConversationContext) createMasterSummary(ctx context.Context) error {
	conf := config.GetConfig()

	// Create messages for the master summary
	// Add summaries from each level with appropriate weighting
	messagesToSummarize := cc.prepareMasterSummaryMessages()

	// Generate the master summary
	masterSummaryContent, err := cc.generateText(ctx, messagesToSummarize, conf.Summarization.MasterSummaryPrompt, config.SummaryModel)
	if err != nil {
		return fmt.Errorf("failed to generate master summary: %w", err)
	}

	// Get IDs of all summaries
	var summaryIDs []int
	for _, summary := range cc.Summaries {
		summaryIDs = append(summaryIDs, summary.ID)
	}

	// Store the master summary with a special level (0 for master)
	masterLevel := 0 // Special level for master summary
	masterSummaryID, err := storage.CreateSummary(ctx, cc.ConversationID, masterSummaryContent, masterLevel, summaryIDs)
	if err != nil {
		return fmt.Errorf("failed to store master summary: %w", err)
	}

	log.Printf("Created master summary ID %d incorporating %d summaries", masterSummaryID, len(summaryIDs))

	// Store the master summary in our context
	cc.MasterSummary = &models.Summary{
		Content: masterSummaryContent,
		Level:   masterLevel,
		ID:      masterSummaryID,
	}

	return nil
}

// updateMasterSummary updates the existing master summary with new information
func (cc *ConversationContext) updateMasterSummary(ctx context.Context) error {
	if cc.MasterSummary == nil {
		return cc.createMasterSummary(ctx)
	}

	// Create messages for updating the master summary
	var messagesToSummarize []models.Message

	// Start with the existing master summary as context
	messagesToSummarize = append(messagesToSummarize, models.Message{
		Role:    "system",
		Content: fmt.Sprintf("Current master summary: %s", cc.MasterSummary.Content),
		ID:      cc.MasterSummary.ID,
	})

	// Find new summaries that weren't part of the original master summary
	newSummaryMessages := cc.getNewSummariesForMasterUpdate(ctx)

	// Skip if no new summaries to integrate
	if len(newSummaryMessages) == 0 {
		log.Printf("No new summaries to integrate into master summary")
		return nil
	}

	// Add new summaries to the messages list
	messagesToSummarize = append(messagesToSummarize, newSummaryMessages...)

	// Generate the updated master summary
	updatedMasterPrompt := "Update the master summary with new information, maintaining the most important points while integrating new context."
	masterSummaryContent, err := cc.generateText(ctx, messagesToSummarize, updatedMasterPrompt, config.SummaryModel)
	if err != nil {
		return fmt.Errorf("failed to update master summary: %w", err)
	}

	// Get IDs of all summaries
	var summaryIDs []int
	for _, summary := range cc.Summaries {
		summaryIDs = append(summaryIDs, summary.ID)
	}

	// Add the old master summary ID
	summaryIDs = append(summaryIDs, cc.MasterSummary.ID)

	// Store the updated master summary
	masterLevel := 0 // Special level for master summary
	masterSummaryID, err := storage.CreateSummary(ctx, cc.ConversationID, masterSummaryContent, masterLevel, summaryIDs)
	if err != nil {
		return fmt.Errorf("failed to store updated master summary: %w", err)
	}

	log.Printf("Updated master summary with ID %d", masterSummaryID)

	// Update the master summary in our context
	cc.MasterSummary = &models.Summary{
		Content: masterSummaryContent,
		Level:   masterLevel,
		ID:      masterSummaryID,
	}

	return nil
}

// prepareMasterSummaryMessages creates weighted messages for the master summary
func (cc *ConversationContext) prepareMasterSummaryMessages() []models.Message {
	conf := config.GetConfig()
	var messagesToSummarize []models.Message

	// Start with a system message describing the importance of weighting
	messagesToSummarize = append(messagesToSummarize, models.Message{
		Role:    "system",
		Content: conf.Summarization.MasterSummaryPrompt,
	})

	// Group summaries by level
	summariesByLevel := GroupSummariesByLevel(cc.Summaries)

	// Find the max level
	maxLevel := FindMaxLevel(cc.Summaries)

	// Add summaries from each level, with decreasing importance for higher levels
	for level := 1; level <= maxLevel; level++ {
		if summaries, ok := summariesByLevel[level]; ok {
			// Calculate weight for this level
			weight := 1.0
			for i := 1; i < level; i++ {
				weight *= conf.Summarization.SummaryWeightCoefficient
			}

			// Add summaries with level information and weight
			for _, summary := range summaries {
				weightedPrompt := fmt.Sprintf("Level %d summary (importance weight: %.2f): %s",
					level, weight, summary.Content)

				messagesToSummarize = append(messagesToSummarize, models.Message{
					Role:    "system",
					Content: weightedPrompt,
					ID:      summary.ID,
				})
			}
		}
	}

	return messagesToSummarize
}

// getNewSummariesForMasterUpdate returns messages for summaries that aren't in the master summary
func (cc *ConversationContext) getNewSummariesForMasterUpdate(ctx context.Context) []models.Message {
	conf := config.GetConfig()

	// Get summary IDs already included in the master summary
	masterSummaryIDs := make(map[int]bool)
	masterSummary, err := storage.GetSummary(ctx, cc.MasterSummary.ID)
	if err == nil {
		for _, id := range masterSummary.SourceIDs {
			masterSummaryIDs[id] = true
		}
	}

	// Group summaries by level that aren't already part of the master summary
	summariesByLevel := make(map[int][]models.Summary)
	for _, summary := range cc.Summaries {
		if !masterSummaryIDs[summary.ID] {
			summariesByLevel[summary.Level] = append(summariesByLevel[summary.Level], summary)
		}
	}

	// Find the max level
	maxLevel := 0
	for level := range summariesByLevel {
		if level > maxLevel {
			maxLevel = level
		}
	}

	var newSummaryMessages []models.Message

	// Add new summaries with appropriate weighting
	for level := 1; level <= maxLevel; level++ {
		if summaries, ok := summariesByLevel[level]; ok {
			// Calculate weight for this level
			weight := 1.0
			for i := 1; i < level; i++ {
				weight *= conf.Summarization.SummaryWeightCoefficient
			}

			// Add summaries with level information and weight
			for _, summary := range summaries {
				weightedPrompt := fmt.Sprintf("New level %d summary (importance weight: %.2f): %s",
					level, weight, summary.Content)

				newSummaryMessages = append(newSummaryMessages, models.Message{
					Role:    "system",
					Content: weightedPrompt,
					ID:      summary.ID,
				})
			}
		}
	}

	return newSummaryMessages
}
