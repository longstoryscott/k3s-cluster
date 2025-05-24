// Package recherche provides shared functionality for web search and content extraction
package recherche

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"proxyllama/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

// SearchResult represents a search result from a web query
type SearchResult struct {
	Query    string   `json:"query"`
	Results  []string `json:"results,omitempty"`
	Contents []string `json:"contents,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// PerformWebSearch performs a web search for the given query and returns URLs
func PerformWebSearch(ctx context.Context, query string, numResults int) ([]string, error) {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":       filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":       line,
		"query":      query,
		"numResults": numResults,
	}).Info("Performing web search")

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

	// Create HTTP client
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
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Warn("No search results found, falling back to default sources")
		urls = append(urls,
			"https://en.wikipedia.org/wiki/"+strings.ReplaceAll(query, " ", "_"),
			"https://www.britannica.com/search?query="+strings.ReplaceAll(query, " ", "+"))
	}

	// Limit to requested number of results
	if len(urls) > numResults {
		urls = urls[:numResults]
	}

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":    filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":    line,
		"query":   query,
		"results": len(urls),
	}).Info("Found search results")
	return urls, nil
}

// ExtractTextFromURL extracts text content from a URL
func ExtractTextFromURL(ctx context.Context, urlString string) (string, error) {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
		"url":  urlString,
	}).Info("Extracting text from URL")

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

// QuickSearch performs a complete search operation: search, extract content, and format results
func QuickSearch(ctx context.Context, query string, maxResults int, includeContents bool) (*SearchResult, error) {
	result := &SearchResult{
		Query: query,
	}

	// Check for URL in query, if so just extract from that URL
	if strings.Contains(query, "http://") || strings.Contains(query, "https://") {
		// Extract URL from query
		words := strings.Fields(query)
		var foundURL string
		for _, word := range words {
			if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
				// Basic URL validation by parsing
				_, err := url.ParseRequestURI(word)
				if err == nil {
					foundURL = word
					break
				}
			}
		}

		if foundURL != "" {
			// Just extract from this specific URL
			result.Results = []string{foundURL}

			if includeContents {
				content, err := ExtractTextFromURL(ctx, foundURL)
				if err != nil {
					result.Error = err.Error()
					return result, err
				}
				result.Contents = []string{content}
			}
			return result, nil
		}
	}

	// If no URL found, perform web search
	urls, err := PerformWebSearch(ctx, query, maxResults)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	result.Results = urls

	// Extract content if requested
	if includeContents && len(urls) > 0 {
		result.Contents = []string{}
		for _, url := range urls {
			content, err := ExtractTextFromURL(ctx, url)
			if err != nil {
				// Just log the error and continue
				_, file, line, _ := runtime.Caller(0)
				logrus.WithFields(logrus.Fields{
					"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
					"line":  line,
					"url":   url,
					"error": err,
				}).Warn("Error extracting text from URL")
				continue
			}
			if content != "" {
				result.Contents = append(result.Contents, content)
			}
		}
	}

	return result, nil
}

// DetectSearchIntent determines if a query likely requires web search
func DetectSearchIntent(query string, userMessages []models.OllamaMessage) bool {
	// Check for explicit web search indicators
	lowerQuery := strings.ToLower(query)
	explicitIndicators := []string{
		"search", "google", "look up", "find information", "search for",
		"what is the latest", "recent news", "current", "today's",
		"latest update", "website", "webpage", "url", "link",
		"http://", "https://", "www.",
	}

	for _, indicator := range explicitIndicators {
		if strings.Contains(lowerQuery, indicator) {
			return true
		}
	}

	// Check for question formats that likely need external information
	questionIndicators := []string{
		"what is", "who is", "where is", "when did", "how does",
		"why does", "can you find", "what are", "is there",
		"tell me about", "explain", "define", "summarize",
	}

	for _, indicator := range questionIndicators {
		if strings.HasPrefix(lowerQuery, indicator) {
			return true
		}
	}

	// Check for date/time-sensitive queries
	timeIndicators := []string{
		"today", "yesterday", "this week", "this month", "this year",
		"latest", "newest", "recent", "current", "update",
	}

	for _, indicator := range timeIndicators {
		if strings.Contains(lowerQuery, indicator) {
			return true
		}
	}

	// Check for URLs in the query
	if strings.Contains(query, "http://") || strings.Contains(query, "https://") {
		return true
	}

	return false
}
