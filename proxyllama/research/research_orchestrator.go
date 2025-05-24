package research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"proxyllama/config"
	pxcx "proxyllama/context"
	"proxyllama/models"
	"proxyllama/recherche"
	"proxyllama/storage"
)

// format is the JSON schema for the research plan
var format map[string]any = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"id": map[string]any{
			"type": "string",
		},
		"question": map[string]any{
			"type": "string",
		},
		"synthesized_answer": map[string]any{
			"type": "string",
		},
		"error_message": map[string]any{
			"type": "string",
		},
	},
}

// PerformDeepResearch is the main orchestrator for research tasks
func PerformDeepResearch(ctx context.Context, taskID, userID, originalQuery string, conversationID *int) {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":   line,
		"taskID": taskID,
		"query":  originalQuery,
	}).Info("Starting research task")
	updateTaskStatus(ctx, taskID, models.ResearchTaskStatusPlanning, nil)

	// 1. Decompose & Plan
	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":   line,
		"taskID": taskID,
	}).Info("Planning phase - Decomposing query")
	plan, err := planResearchTask(ctx, userID, taskID, originalQuery)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"error":  err,
		}).Warn("Error in planning phase")
		errorMsg := fmt.Sprintf("Planning failed: %v", err)
		updateTaskStatus(ctx, taskID, models.ResearchTaskStatusFailed, &errorMsg)
		return
	}

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":         filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":         line,
		"taskID":       taskID,
		"subQuestions": len(plan.SubQuestions),
	}).Info("Plan created with sub-questions")
	updateTaskStatus(ctx, taskID, models.ResearchTaskStatusGathering, nil)

	// Channel for collecting synthesized sub-answers
	subResultsChan := make(chan models.ResearchQuestionResult, len(plan.SubQuestions))
	var wg sync.WaitGroup // To wait for all sub-question processing goroutines

	// 2. Information Gathering & Initial Synthesis per Sub-Question
	for _, sq := range plan.SubQuestions {
		wg.Add(1)
		go func(subQ models.ResearchQuestion) {
			defer wg.Done()
			processSubQuestion(ctx, userID, taskID, subQ, subResultsChan)
		}(sq)
	}

	// Wait for all sub-questions to be processed
	wg.Wait()
	close(subResultsChan)

	// Collect all sub-results
	var finalSubAnswers []models.ResearchQuestionResult
	for res := range subResultsChan {
		finalSubAnswers = append(finalSubAnswers, res)

		// Store individual sub-answers in the database
		if res.Error != nil {
			errorMsg := res.Error.Error()
			_, _, _ = storage.UpdateSubtaskStatus(ctx, taskID, res.ID, string(models.ResearchTaskStatusFailed), &errorMsg)
		} else {
			_, _ = storage.StoreSynthesizedAnswer(ctx, taskID, res.ID, res.SynthesizedAnswer)
			_, _, _ = storage.UpdateSubtaskStatus(ctx, taskID, res.ID, string(models.ResearchTaskStatusCompleted), nil)
		}
	}

	// 3. Consolidate & Final Report
	updateTaskStatus(ctx, taskID, models.ResearchTaskStatusSynthesizing, nil)
	finalReport, err := consolidateResearchResults(ctx, userID, taskID, originalQuery, plan, finalSubAnswers)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"error":  err,
		}).Warn("Error generating final report")
		errorMsg := fmt.Sprintf("Final report generation failed: %v", err)
		updateTaskStatus(ctx, taskID, models.ResearchTaskStatusFailed, &errorMsg)
		return
	}

	// Store final result in database
	_, err = storage.StoreFinalResult(ctx, taskID, finalReport)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"error":  err,
		}).Warn("Error storing final result")
	}

	updateTaskStatus(ctx, taskID, models.ResearchTaskStatusCompleted, nil)
	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":   line,
		"taskID": taskID,
	}).Info("Deep research completed successfully")

	// Optional: If conversationID is provided, add the final report as an assistant message
	if conversationID != nil {
		// Add the final result to the conversation as an assistant message
		err := addResearchResultToConversation(ctx, userID, finalReport, conversationID)
		if err != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":   line,
				"taskID": taskID,
				"error":  err,
			}).Warn("Failed to add result to conversation")
		}
	}
}

// planResearchTask uses the LLM to decompose a query into sub-questions
func planResearchTask(ctx context.Context, userID, taskID, query string) (*models.ResearchPlan, error) {
	// Call the model for the planning step
	plan, err := CallLLMForResearchPlan(ctx, userID, query)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"error":  err,
		}).Warn("Error parsing plan JSON")
		_, file, line, _ = runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":    filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":    line,
			"taskID":  taskID,
			"rawPlan": plan.RawPlan,
		}).Warn("Raw plan JSON")
		return nil, fmt.Errorf("plan parsing failed: %w", err)
	}

	var subTaskIds []int
	// Create subtasks in database
	for _, sq := range plan.SubQuestions {
		id, err := storage.SaveSubtask(ctx, &models.ResearchSubtask{
			TaskID:     taskID,
			QuestionID: sq.ID,
			Status:     models.ResearchTaskStatusPending,
		})
		if err != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":   line,
				"taskID": taskID,
				"error":  err,
			}).Warn("Error saving subtask")
			continue
		}

		subTaskIds = append(subTaskIds, id)
	}

	_, err = storage.StoreResearchPlan(ctx, taskID, plan)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"error":  err,
		}).Warn("Error storing subtasks")
	}

	return plan, nil
}

// processSubQuestion handles the information gathering and synthesis for a single sub-question
func processSubQuestion(ctx context.Context, userID, taskID string, subQ models.ResearchQuestion, resultChan chan<- models.ResearchQuestionResult) {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":     filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":     line,
		"taskID":   taskID,
		"subQID":   subQ.ID,
		"question": subQ.Question,
	}).Info("Starting sub-question")
	updateSubtaskStatus(ctx, taskID, subQ.ID, models.ResearchTaskStatusGathering, nil)

	var allExtractedTexts []string
	var allSources []string

	// Gather information from web sources using the keywords
	for _, keyword := range subQ.Keywords {
		// Use shared recherche package for web search
		searchResults, err := recherche.PerformWebSearch(ctx, keyword, 3)
		if err != nil {
			_, file, line, _ := runtime.Caller(0)
			logrus.WithFields(logrus.Fields{
				"file":    filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
				"line":    line,
				"taskID":  taskID,
				"subQID":  subQ.ID,
				"keyword": keyword,
				"error":   err,
			}).Warn("Error searching for keyword")
			continue
		}

		for _, resultURL := range searchResults {
			// Extract content from URLs using shared function
			textContent, err := recherche.ExtractTextFromURL(ctx, resultURL)
			if err != nil {
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":   line,
					"taskID": taskID,
					"subQID": subQ.ID,
					"url":    resultURL,
					"error":  err,
				}).Warn("Error extracting from URL")
				continue
			}

			// Add the content and source if successful
			if len(textContent) > 0 {
				allExtractedTexts = append(allExtractedTexts, textContent)
				allSources = append(allSources, resultURL)
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":   line,
					"taskID": taskID,
					"subQID": subQ.ID,
					"url":    resultURL,
					"length": len(textContent),
				}).Info("Added content from URL")

				// Limit the number of sources per question
				if len(allExtractedTexts) >= 5 {
					break
				}
			}
		}

		// If we have enough sources, stop searching with other keywords
		if len(allExtractedTexts) >= 5 {
			break
		}
	}

	// Store gathered information
	_, err := storage.StoreGatheredInfo(ctx, taskID, subQ.ID, allExtractedTexts, allSources)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"subQID": subQ.ID,
			"error":  err,
		}).Warn("Error storing gathered info")
	}

	// Check if we gathered any useful information
	if len(allExtractedTexts) == 0 {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"subQID": subQ.ID,
		}).Warn("No text gathered")
		errorMsg := "No information found for this sub-question"
		updateSubtaskStatus(ctx, taskID, subQ.ID, models.ResearchTaskStatusFailed, &errorMsg)
		resultChan <- models.ResearchQuestionResult{
			ID:                subQ.ID,
			Question:          subQ.Question,
			SynthesizedAnswer: "No information found for this sub-question.",
			Error:             fmt.Errorf("no information found"),
			ErrorMessage:      "No information found for this sub-question.",
		}
		return
	}

	// Synthesize gathered information
	updateSubtaskStatus(ctx, taskID, subQ.ID, models.ResearchTaskStatusProcessing, nil)

	// Combine texts with source citations
	var combinedText strings.Builder
	for i, text := range allExtractedTexts {
		combinedText.WriteString(fmt.Sprintf("Source %d (%s):\n%s\n\n", i+1, allSources[i], text))

		// Break if the text is getting too large for the context window
		if combinedText.Len() > 12000 {
			combinedText.WriteString("...(truncated due to length)...")
			break
		}
	}

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":    filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":    line,
		"taskID":  taskID,
		"subQID":  subQ.ID,
		"sources": len(allExtractedTexts),
	}).Info("Synthesizing information from sources")
	result, err := CallLLMForSubResult(ctx, userID, combinedText.String(), nil)
	if err != nil {
		errorMsg := fmt.Sprintf("Error synthesizing: %v", err)
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"subQID": subQ.ID,
			"error":  errorMsg,
		}).Warn("Error synthesizing")
		updateSubtaskStatus(ctx, taskID, subQ.ID, models.ResearchTaskStatusFailed, &errorMsg)
		result.Error = err
		result.ErrorMessage = errorMsg
		resultChan <- *result
		return
	}

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":   line,
		"taskID": taskID,
		"subQID": subQ.ID,
		"length": len(result.SynthesizedAnswer),
	}).Info("Synthesis complete")
	resultChan <- models.ResearchQuestionResult{
		ID:                subQ.ID,
		Question:          subQ.Question,
		SynthesizedAnswer: result.SynthesizedAnswer,
	}
}

// consolidateResearchResults creates a final coherent report from all sub-question results
func consolidateResearchResults(ctx context.Context, userID, taskID, originalQuery string, plan *models.ResearchPlan, subResults []models.ResearchQuestionResult) (*models.ResearchQuestionResult, error) {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":         filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":         line,
		"taskID":       taskID,
		"subQuestions": len(subResults),
	}).Info("Consolidating results from sub-questions")

	var consolidationInput strings.Builder
	consolidationInput.WriteString(fmt.Sprintf("Original User Research Request: %s\n\n", originalQuery))
	consolidationInput.WriteString("Main Intent: " + plan.MainIntent + "\n\n")
	consolidationInput.WriteString("Synthesized Findings for Sub-Questions:\n\n")

	for _, result := range subResults {
		if result.Error != nil {
			consolidationInput.WriteString(fmt.Sprintf("### Question %d: %s\n\nError: %s\n\n",
				result.ID, result.Question, result.ErrorMessage))
		} else {
			consolidationInput.WriteString(fmt.Sprintf("### Question %d: %s\n\n%s\n\n",
				result.ID, result.Question, result.SynthesizedAnswer))
		}
	}

	finalReport, err := CallLLMForResult(ctx, userID, consolidationInput.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("final report generation failed: %w", err)
	}

	return finalReport, nil
}

// CallLLMForResult calls the LLM with the given prompt for research steps
func CallLLMForResult(ctx context.Context, userID, consolidatedInput string, userMessages []models.OllamaMessage) (*models.ResearchQuestionResult, error) {
	var ollamaMessages []models.OllamaMessage

	cfg, err := pxcx.GetUserConfig(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	profile, err := storage.GetModelProfile(ctx, cfg.ModelProfiles.ResearchConsolidationProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model profile: %w", err)
	}
	ollamaMessages = append(ollamaMessages, models.OllamaMessage{
		Role:    "system",
		Content: fmt.Sprintf("%s \n\n%s", profile.SystemPrompt, consolidatedInput),
	})

	// Add any additional user messages
	ollamaMessages = append(ollamaMessages, userMessages...)

	// Return the assistant's message content
	return doResearch[models.ResearchQuestionResult](ctx, profile, ollamaMessages)
}

// CallLLMForResult calls the LLM with the given prompt for research steps
func CallLLMForSubResult(ctx context.Context, userID, consolidatedInput string, userMessages []models.OllamaMessage) (*models.ResearchQuestionResult, error) {
	var ollamaMessages []models.OllamaMessage

	cfg, err := pxcx.GetUserConfig(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	profile, err := storage.GetModelProfile(ctx, cfg.ModelProfiles.ResearchAnalysisProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model profile: %w", err)
	}
	ollamaMessages = append(ollamaMessages, models.OllamaMessage{
		Role:    "system",
		Content: fmt.Sprintf("%s \n\n%s", profile.SystemPrompt, consolidatedInput),
	})

	// Add any additional user messages
	ollamaMessages = append(ollamaMessages, userMessages...)

	// Return the assistant's message content
	return doResearch[models.ResearchQuestionResult](ctx, profile, ollamaMessages)
}

// CallLLMForResearchPlan calls the LLM with the given prompt for research steps
func CallLLMForResearchPlan(ctx context.Context, userID, query string) (*models.ResearchPlan, error) {
	var ollamaMessages []models.OllamaMessage

	cfg, err := pxcx.GetUserConfig(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	systemPrompt := fmt.Sprintf("%s User Query: %s", cfg.ModelProfiles.ResearchPlanProfileID, query)
	ollamaMessages = append(ollamaMessages, models.OllamaMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	profile, err := storage.GetModelProfile(ctx, cfg.ModelProfiles.ResearchPlanProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model profile: %w", err)
	}

	return doResearch[models.ResearchPlan](ctx, profile, ollamaMessages)
}

func doResearch[T any](ctx context.Context, profile *models.ModelProfile, ollamaMessages []models.OllamaMessage) (*T, error) {
	// Create a non-streaming request to get the full response at once
	ollamaReq := models.OllamaReq{
		Model:    profile.ModelName,
		Messages: ollamaMessages,
		Format:   format,
		Stream:   false, // We want the complete response at once
	}

	// Convert to JSON
	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Get Ollama URL from config
	conf := config.GetConfig()
	url := conf.Ollama.BaseURL + "/api/chat"

	// Send request to Ollama
	resp, err := http.Post(url, "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var ollamaResp models.OllamaResp
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result T
	if err := json.Unmarshal([]byte(ollamaResp.Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal research format: %w", err)
	}

	// Return the assistant's message content
	return &result, nil
}

// Helper functions

// updateTaskStatus updates the status of a research task in the database
func updateTaskStatus(ctx context.Context, taskID string, status models.ResearchTaskStatus, errorMsg *string) {
	_, err := storage.UpdateTaskStatus(ctx, taskID, string(status), errorMsg)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"error":  err,
		}).Warn("Error updating task status")
	}
}

// updateSubtaskStatus updates the status of a research subtask in the database
func updateSubtaskStatus(ctx context.Context, taskID string, questionID int, status models.ResearchTaskStatus, errorMsg *string) {
	_, _, err := storage.UpdateSubtaskStatus(ctx, taskID, questionID, string(status), errorMsg)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":   line,
			"taskID": taskID,
			"subQID": questionID,
			"error":  err,
		}).Warn("Error updating subtask status")
	}
}

// extractJSON tries to extract a JSON object from a string that might contain additional text
func extractJSON(input string) string {
	// Find the start and end of a JSON object
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")

	if start >= 0 && end > start {
		return input[start : end+1]
	}

	return input // Return original if no JSON found
}

// addResearchResultToConversation adds the research result to a conversation as an assistant message
func addResearchResultToConversation(ctx context.Context, userID string, finalResult *models.ResearchQuestionResult, conversationID *int) error {
	if conversationID == nil {
		return fmt.Errorf("no conversation ID provided")
	}

	// Get the conversation context
	convCtx, err := pxcx.GetOrCreateConversation(ctx, userID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation context: %w", err)
	}

	// Add the research result as an assistant message
	message := "Research Results:\n\n" + finalResult.SynthesizedAnswer
	if err := convCtx.AddAssistantMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to add assistant message: %w", err)
	}

	return nil
}
