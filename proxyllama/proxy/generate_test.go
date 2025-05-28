package proxy

import (
	"context"
	"proxyllama/config"
	"proxyllama/models"
	"testing"
)

func Test_StreamOllamaGenerateRequest(t *testing.T) {
	confFile := "testdata/.config.yaml"
	config.GetConfig(&confFile)

	reqBody := models.OllamaGenerateReq{
		Model:  config.DefaultPrimaryProfile.ModelName,
		Prompt: "Tell me a joke about llamas.",
	}
	content, err := StreamOllamaGenerateRequest(context.Background(), config.DefaultFormattingProfile.ModelName, reqBody)
	if err != nil {
		t.Fatalf("Failed to get generated response: %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("Received empty generate for text: %s", content)
	}
	t.Logf("Generate for '%s': %v", reqBody.Prompt, content)
}
