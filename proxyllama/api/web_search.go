package api

import (
	"fmt"
	"path/filepath"
	"proxyllama/context"
	"proxyllama/recherche"
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// RegisterWebSearchRoutes adds web search endpoints
func RegisterWebSearchRoutes(app *fiber.App) {
	app.Post("/api/websearch", PerformWebSearch)
	app.Post("/api/websearch/detect", DetectWebSearchIntent)
}

// DetectWebSearchIntent determines if a query likely needs a web search
func DetectWebSearchIntent(c *fiber.Ctx) error {
	var req struct {
		Query     string      `json:"query"`
		UserID    string      `json:"user_id,omitempty"`
		OllamaReq interface{} `json:"ollama_req,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return handleError(err, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Query == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Query is required")
	}

	needsWebSearch := recherche.DetectSearchIntent(req.Query, nil)

	// Check user config if available
	if req.UserID != "" {
		cfg, err := context.GetUserConfig(req.UserID)
		if err == nil && cfg.WebSearch != nil {
			// Only enable auto-detect if web search is enabled
			if !cfg.WebSearch.Enabled {
				needsWebSearch = false
			} else {
				// Only auto-detect if that feature is enabled
				if !cfg.WebSearch.AutoDetect {
					needsWebSearch = false
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"needs_web_search": needsWebSearch,
	})
}

// PerformWebSearch handles a web search request
func PerformWebSearch(c *fiber.Ctx) error {
	var req struct {
		Query          string `json:"query"`
		UserID         string `json:"user_id,omitempty"`
		MaxResults     *int   `json:"max_results,omitempty"`
		IncludeContent *bool  `json:"include_content,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return handleError(err, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Query == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Query is required")
	}

	maxResults := 3
	includeContent := true

	// Override defaults with request values if provided
	if req.MaxResults != nil {
		maxResults = *req.MaxResults
	}
	if req.IncludeContent != nil {
		includeContent = *req.IncludeContent
	}

	// Check user config if available
	if req.UserID != "" {
		cfg, err := context.GetUserConfig(req.UserID)
		if err == nil && cfg.WebSearch != nil {
			if cfg.WebSearch.MaxResults > 0 {
				maxResults = cfg.WebSearch.MaxResults
			}
			includeContent = cfg.WebSearch.IncludeResults
		}
	}

	// Cap max results at 5 to prevent abuse
	if maxResults > 5 {
		maxResults = 5
	}

	// Log the search request
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":           filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":           line,
		"query":          req.Query,
		"maxResults":     maxResults,
		"includeContent": includeContent,
	}).Info("Web search request")

	// Perform the web search
	results, err := recherche.QuickSearch(c.Context(), req.Query, maxResults, includeContent)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, fmt.Sprintf("Web search failed: %v", err))
	}

	return c.JSON(results)
}
