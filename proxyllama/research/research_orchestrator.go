package research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"proxyllama/config"
	pxcx "proxyllama/context"
	"proxyllama/models"
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
func PerformDeepResearch(ctx context.Context, taskID, userID, originalQuery, modelName string, conversationID *int) {
	log.Printf("[ResearchTask %s] Starting for query: %s", taskID, originalQuery)
	updateTaskStatus(ctx, taskID, models.ResearchTaskStatusPlanning, nil)

	// 1. Decompose & Plan
	log.Printf("[ResearchTask %s] Planning phase - Decomposing query...", taskID)
	plan, err := planResearchTask(ctx, taskID, originalQuery, modelName)
	if err != nil {
		log.Printf("[ResearchTask %s] Error in planning phase: %v", taskID, err)
		errorMsg := fmt.Sprintf("Planning failed: %v", err)
		updateTaskStatus(ctx, taskID, models.ResearchTaskStatusFailed, &errorMsg)
		return
	}

	log.Printf("[ResearchTask %s] Plan created with %d sub-questions", taskID, len(plan.SubQuestions))
	updateTaskStatus(ctx, taskID, models.ResearchTaskStatusGathering, nil)

	// Channel for collecting synthesized sub-answers
	subResultsChan := make(chan models.ResearchQuestionResult, len(plan.SubQuestions))
	var wg sync.WaitGroup // To wait for all sub-question processing goroutines

	// 2. Information Gathering & Initial Synthesis per Sub-Question
	for _, sq := range plan.SubQuestions {
		wg.Add(1)
		go func(subQ models.ResearchQuestion) {
			defer wg.Done()
			processSubQuestion(ctx, taskID, modelName, subQ, subResultsChan)
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
			_ = storage.UpdateSubtaskStatus(ctx, taskID, res.ID, string(models.ResearchTaskStatusFailed), &errorMsg)
		} else {
			_ = storage.StoreSubtaskResult(ctx, taskID, res.ID, res.SynthesizedAnswer)
			_ = storage.UpdateSubtaskStatus(ctx, taskID, res.ID, string(models.ResearchTaskStatusCompleted), nil)
		}
	}

	// 3. Consolidate & Final Report
	updateTaskStatus(ctx, taskID, models.ResearchTaskStatusSynthesizing, nil)
	finalReport, err := consolidateResearchResults(ctx, taskID, originalQuery, plan, finalSubAnswers, modelName)
	if err != nil {
		log.Printf("[ResearchTask %s] Error generating final report: %v", taskID, err)
		errorMsg := fmt.Sprintf("Final report generation failed: %v", err)
		updateTaskStatus(ctx, taskID, models.ResearchTaskStatusFailed, &errorMsg)
		return
	}

	// Store final result in database
	err = storage.StoreFinalResearchResult(ctx, taskID, finalReport)
	if err != nil {
		log.Printf("[ResearchTask %s] Error storing final result: %v", taskID, err)
	}

	updateTaskStatus(ctx, taskID, models.ResearchTaskStatusCompleted, nil)
	log.Printf("[ResearchTask %s] Deep research completed successfully", taskID)

	// Optional: If conversationID is provided, add the final report as an assistant message
	if conversationID != nil {
		// Add the final result to the conversation as an assistant message
		err := addResearchResultToConversation(ctx, userID, modelName, finalReport, conversationID)
		if err != nil {
			log.Printf("[ResearchTask %s] Failed to add result to conversation: %v", taskID, err)
		}
	}
}

// planResearchTask uses the LLM to decompose a query into sub-questions
func planResearchTask(ctx context.Context, taskID, query, modelName string) (*models.ResearchPlan, error) {
	planningPrompt := fmt.Sprintf(
		"You are a research planning assistant. Analyze the following user request. "+
			"1. Clarify the core intent and scope. "+
			"2. Break down the request into 3-5 key research questions or sub-topics. "+
			"3. For each sub-topic, suggest 1-3 initial search engine query keywords. "+
			"User Query: %s", query)

	// Call the model for the planning step
	plan, err := CallLLMForResearchPlan(ctx, modelName, planningPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	if err != nil {
		log.Printf("[ResearchTask %s] Error parsing plan JSON: %v", taskID, err)
		log.Printf("[ResearchTask %s] Raw plan JSON: %s", taskID, plan.RawPlan)
		return nil, fmt.Errorf("plan parsing failed: %w", err)
	}

	// Store plan in database
	err = storage.StoreResearchPlan(ctx, taskID, plan)
	if err != nil {
		log.Printf("[ResearchTask %s] Error storing plan: %v", taskID, err)
		// Non-fatal error, continue with the process
	}

	// Create subtasks in database
	for _, sq := range plan.SubQuestions {
		_, err := storage.CreateResearchSubtask(ctx, taskID, sq.ID, sq.Question)
		if err != nil {
			log.Printf("[ResearchTask %s] Error creating subtask for question %d: %v", taskID, sq.ID, err)
			// Non-fatal error, continue with other subtasks
		}
	}

	return plan, nil
}

// processSubQuestion handles the information gathering and synthesis for a single sub-question
func processSubQuestion(ctx context.Context, taskID, modelName string, subQ models.ResearchQuestion, resultChan chan<- models.ResearchQuestionResult) {
	log.Printf("[ResearchTask %s - SubQ %d] Starting: %s", taskID, subQ.ID, subQ.Question)
	updateSubtaskStatus(ctx, taskID, subQ.ID, models.ResearchTaskStatusGathering, nil)

	var allExtractedTexts []string
	var allSources []string

	// Gather information from web sources using the keywords
	for _, keyword := range subQ.Keywords {
		// Use ExternalSearch interface that can be implemented with real search providers
		searchResults, err := performWebSearch(ctx, keyword, 3)
		if err != nil {
			log.Printf("[ResearchTask %s - SubQ %d] Error searching for '%s': %v", taskID, subQ.ID, keyword, err)
			continue
		}

		for _, resultURL := range searchResults {
			// Extract content from URLs
			textContent, err := extractTextFromURL(ctx, resultURL)
			if err != nil {
				log.Printf("[ResearchTask %s - SubQ %d] Error extracting from '%s': %v", taskID, subQ.ID, resultURL, err)
				continue
			}

			// Add the content and source if successful
			if len(textContent) > 0 {
				allExtractedTexts = append(allExtractedTexts, textContent)
				allSources = append(allSources, resultURL)
				log.Printf("[ResearchTask %s - SubQ %d] Added content from %s (length: %d)",
					taskID, subQ.ID, resultURL, len(textContent))

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
	err := storage.StoreSubtaskGatheredInfo(ctx, taskID, subQ.ID, allExtractedTexts, allSources)
	if err != nil {
		log.Printf("[ResearchTask %s - SubQ %d] Error storing gathered info: %v", taskID, subQ.ID, err)
	}

	// Check if we gathered any useful information
	if len(allExtractedTexts) == 0 {
		log.Printf("[ResearchTask %s - SubQ %d] No text gathered", taskID, subQ.ID)
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

	synthesisPrompt := fmt.Sprintf(
		"You are a research analyst. Based ONLY on the provided text snippets, answer the following research question concisely. "+
			"Synthesize the information and extract key findings. If the text doesn't answer the question, say so explicitly. "+
			"Include references to the sources in your answer when appropriate. "+
			"Research Question: %s\n\nProvided Text:\n%s", subQ.Question, combinedText.String())

	log.Printf("[ResearchTask %s - SubQ %d] Synthesizing information from %d sources", taskID, subQ.ID, len(allExtractedTexts))
	result, err := CallLLMForResult(ctx, modelName, synthesisPrompt, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("Error synthesizing: %v", err)
		log.Printf("[ResearchTask %s - SubQ %d] %s", taskID, subQ.ID, errorMsg)
		updateSubtaskStatus(ctx, taskID, subQ.ID, models.ResearchTaskStatusFailed, &errorMsg)
		result.Error = err
		result.ErrorMessage = errorMsg
		resultChan <- *result
		return
	}

	log.Printf("[ResearchTask %s - SubQ %d] Synthesis complete (%d characters)", taskID, subQ.ID, len(result.SynthesizedAnswer))
	resultChan <- models.ResearchQuestionResult{
		ID:                subQ.ID,
		Question:          subQ.Question,
		SynthesizedAnswer: result.SynthesizedAnswer,
	}
}

// consolidateResearchResults creates a final coherent report from all sub-question results
func consolidateResearchResults(ctx context.Context, taskID, originalQuery string, plan *models.ResearchPlan, subResults []models.ResearchQuestionResult, modelName string) (string, error) {
	log.Printf("[ResearchTask %s] Consolidating results from %d sub-questions", taskID, len(subResults))

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

	reportPrompt := fmt.Sprintf(
		"You are a research report writer. You have been provided with findings for sub-topics of a larger research request. "+
			"Combine these findings into a coherent, well-structured report that directly addresses the original user request. "+
			"Start with a brief executive summary, then elaborate on the findings for each sub-question. "+
			"If some sub-questions had errors or insufficient info, acknowledge that in your report. "+
			"Format the report with proper sections, highlighting key points. "+
			"Do not invent information not present in the input. \n\n%s", consolidationInput.String())

	finalReport, err := CallLLMForResult(ctx, modelName, reportPrompt, nil)
	if err != nil {
		return "", fmt.Errorf("final report generation failed: %w", err)
	}

	return finalReport.SynthesizedAnswer, nil
}

// CallLLMForResult calls the LLM with the given prompt for research steps
func CallLLMForResult(ctx context.Context, modelName, systemPrompt string, userMessages []models.OllamaMessage) (*models.ResearchQuestionResult, error) {
	var ollamaMessages []models.OllamaMessage

	// Add system prompt if provided
	if systemPrompt != "" {
		ollamaMessages = append(ollamaMessages, models.OllamaMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add any additional user messages
	ollamaMessages = append(ollamaMessages, userMessages...)

	// Create a non-streaming request to get the full response at once
	ollamaReq := models.OllamaReq{
		Model:    modelName,
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

	var result models.ResearchQuestionResult
	if err := json.Unmarshal([]byte(ollamaResp.Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal research format: %w", err)
	}

	// Return the assistant's message content
	return &result, nil
}

// CallLLMForResearchPlan calls the LLM with the given prompt for research steps
func CallLLMForResearchPlan(ctx context.Context, modelName, systemPrompt string, userMessages []models.OllamaMessage) (*models.ResearchPlan, error) {
	var ollamaMessages []models.OllamaMessage

	// Add system prompt if provided
	if systemPrompt != "" {
		ollamaMessages = append(ollamaMessages, models.OllamaMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add any additional user messages
	ollamaMessages = append(ollamaMessages, userMessages...)

	// Create a non-streaming request to get the full response at once
	ollamaReq := models.OllamaReq{
		Model:    modelName,
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

	var result models.ResearchPlan
	if err := json.Unmarshal([]byte(ollamaResp.Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal research format: %w", err)
	}

	// Return the assistant's message content
	return &result, nil
}

// Helper functions

// updateTaskStatus updates the status of a research task in the database
func updateTaskStatus(ctx context.Context, taskID string, status models.ResearchTaskStatus, errorMsg *string) {
	err := storage.UpdateResearchTaskStatus(ctx, taskID, string(status), errorMsg)
	if err != nil {
		log.Printf("[ResearchTask %s] Error updating task status: %v", taskID, err)
	}
}

// updateSubtaskStatus updates the status of a research subtask in the database
func updateSubtaskStatus(ctx context.Context, taskID string, questionID int, status models.ResearchTaskStatus, errorMsg *string) {
	err := storage.UpdateSubtaskStatus(ctx, taskID, questionID, string(status), errorMsg)
	if err != nil {
		log.Printf("[ResearchTask %s - SubQ %d] Error updating subtask status: %v", taskID, questionID, err)
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
func addResearchResultToConversation(ctx context.Context, userID, modelName, finalResult string, conversationID *int) error {
	if conversationID == nil {
		return fmt.Errorf("no conversation ID provided")
	}

	// Get the conversation context
	convCtx, err := pxcx.GetOrCreateConversation(ctx, userID, modelName, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation context: %w", err)
	}

	// Add the research result as an assistant message
	message := "Research Results:\n\n" + finalResult
	if err := convCtx.AddAssistantMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to add assistant message: %w", err)
	}

	return nil
}

// performWebSearch performs a web search for the given query using DuckDuckGo API
func performWebSearch(ctx context.Context, query string, numResults int) ([]string, error) {
	log.Printf("Performing web search for: %s (limit: %d results)", query, numResults)

	// Create a DuckDuckGo API request URL
	searchURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&pretty=1&no_html=1&skip_disambig=1", url.QueryEscape(query))

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	// Set a user agent to avoid potential blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Use the same client creation approach as in the proxy.Stream function
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			DisableKeepAlives:     false,
			ResponseHeaderTimeout: 5 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
		},
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned non-OK status: %d", resp.StatusCode)
	}

	// Read and parse the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %w", err)
	}

	var searchResult struct {
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			FirstURL string `json:"FirstURL"`
			Text     string `json:"Text"`
		} `json:"RelatedTopics"`
		Results []struct {
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	// Collect URLs from the response
	var urls []string

	// Add the abstract URL if available
	if searchResult.AbstractURL != "" {
		urls = append(urls, searchResult.AbstractURL)
	}

	// Add related topic URLs
	for _, topic := range searchResult.RelatedTopics {
		if topic.FirstURL != "" {
			urls = append(urls, topic.FirstURL)

			// Stop if we have enough results
			if len(urls) >= numResults {
				break
			}
		}
	}

	// Add results URLs if we still need more
	if len(urls) < numResults {
		for _, result := range searchResult.Results {
			if result.FirstURL != "" {
				urls = append(urls, result.FirstURL)

				// Stop if we have enough results
				if len(urls) >= numResults {
					break
				}
			}
		}
	}

	// If we didn't get any results, fall back to Wikipedia and Britannica
	if len(urls) == 0 {
		log.Printf("No search results found, falling back to default sources")
		urls = append(urls,
			"https://en.wikipedia.org/wiki/"+strings.ReplaceAll(query, " ", "_"),
			"https://www.britannica.com/search?query="+strings.ReplaceAll(query, " ", "+"))
	}

	// Limit to requested number of results
	if len(urls) > numResults {
		urls = urls[:numResults]
	}

	log.Printf("Found %d search results for query: %s", len(urls), query)
	return urls, nil
}

// extractTextFromURL extracts text content from a URL
func extractTextFromURL(ctx context.Context, urlString string) (string, error) {
	log.Printf("Extracting text from: %s", urlString)

	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", urlString, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add a user agent to simulate a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed with status %d", resp.StatusCode)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract text from relevant HTML elements
	var textBuilder strings.Builder
	doc.Find("p, h1, h2, h3, h4, h5, article, section").Each(func(i int, s *goquery.Selection) {
		// Skip empty elements or elements with very little content
		text := strings.TrimSpace(s.Text())
		if len(text) > 15 { // Only include elements with meaningful content
			textBuilder.WriteString(text)
			textBuilder.WriteString("\n\n")
		}
	})

	// If no content was found with the main selectors, try a more generic approach
	if textBuilder.Len() < 100 {
		// Get all text from the body
		bodyText := strings.TrimSpace(doc.Find("body").Text())
		if len(bodyText) > 0 {
			// Clean up the text (remove excessive whitespace)
			bodyText = strings.Join(strings.Fields(bodyText), " ")
			textBuilder.WriteString(bodyText)
		}
	}

	result := textBuilder.String()
	if len(result) == 0 {
		return "", fmt.Errorf("no text content extracted")
	}

	// Truncate if too long
	maxLen := 5000
	if len(result) > maxLen {
		result = result[:maxLen] + "...(text truncated)..."
	}

	return result, nil
}
