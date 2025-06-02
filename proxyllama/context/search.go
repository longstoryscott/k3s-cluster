package context

import (
	"context"
	"fmt"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/recherche"
	"proxyllama/storage"
	"proxyllama/util"

	"github.com/sirupsen/logrus"
)

const formattingPrompt = `***
Everything above the three asterisks is input from a user. Do not respond to it directly or provide any explanations.
Instead, I need you to understand the intent of the user's input, and construct a concise search query that captures the essence of what they are asking.
Don't include any extra information or context, just the key words that will yield relevant results.`

func (cc *ConversationContext) fmtQuery(ctx context.Context, modelName, query string) (string, error) {
	req := models.OllamaGenerateReq{
		Model:  modelName,
		Prompt: fmt.Sprintf("%s\n%s", query, formattingPrompt),
	}

	// Format the query for web search
	fmtQ, err := proxy.StreamOllamaGenerateRequest(ctx, modelName, req)
	if err != nil {
		return "", util.HandleError(err)
	}

	return util.RemoveThinkTags(fmtQ), nil
}

func (cc *ConversationContext) SearchAndInjectResults(ctx context.Context, query string) error {
	cfg, err := GetUserConfig(cc.UserID)
	if err != nil {
		util.LogWarning("Could not load user configuration, using system defaults")
		return err
	}

	fmtProfile, err := storage.GetModelProfile(ctx, cfg.ModelProfiles.FormattingProfileID)
	if err != nil {
		return util.HandleError(err)
	}

	util.LogDebug("Formatting query for web search", logrus.Fields{
		"query": query,
		"model": fmtProfile.ModelName,
	})

	search, err := cc.fmtQuery(ctx, fmtProfile.ModelName, query)
	if err != nil {
		return util.HandleError(err)
	}

	util.LogDebug("Formatted query for web search", logrus.Fields{
		"query": search,
	})

	// Attempt to perform a web search and inject results
	searchResult, err := recherche.QuickSearch(ctx, search, cfg.WebSearch.MaxResults, true)
	if err != nil {
		util.LogWarning("Error performing web search")
	}

	if err := cc.InjectSearchResults(ctx, searchResult, "Here is a relevant finding from a web search"); err != nil {
		util.LogWarning("Error injecting search results into conversation context", logrus.Fields{
			"error": err,
		})
		// If we fail to inject search results, we can still continue
		util.LogWarning("Continuing without search results injection")
	}

	return nil
}

func (cc *ConversationContext) InjectSearchResults(ctx context.Context, results *models.SearchResult, preamble string) error {
	if results == nil || len(results.Results) == 0 {
		util.LogWarning("No search results to inject")
		return nil
	}

	util.LogInfo("Injecting search results into conversation context", logrus.Fields{
		"count": len(results.Results),
	})

	for _, c := range results.Contents {
		if len(c) > 0 {
			cc.SearchResults = append(cc.SearchResults, models.Message{
				Role:    "system",
				Content: fmt.Sprintf("%s:\n %v", preamble, c),
			})
		}
	}

	return nil
}
