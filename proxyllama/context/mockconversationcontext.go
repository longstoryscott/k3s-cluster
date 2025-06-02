package context

import (
	"proxyllama/models"
	"time"
)

var MockConversationContext = ConversationContext{
	UserID:         "CgNsc20SBGxkYXA",
	ConversationID: 0,
	Title:          "Test Conversation",
	MasterSummary:  nil,
	Summaries: []models.Summary{
		{
			CreatedAt:      time.Date(2023, 10, 01, 12, 00, 00, 0, time.UTC),
			ConversationID: 0,
			Content:        `by bundling the inference for both LLMs and image generation into one container (or sidecar) that requests two GPUs, you ensure that model splitting can occur across both devices. This approach meets your requirement while keeping the gRPC interface separate from the inference logic, just as before—but now with full multi-GPU utilization available in a single process.`,
			SourceIDs:      []int{1, 2, 3},
		},
	},
	RetrievedMemories: []models.Memory{},
	Messages: []models.Message{
		{
			ID:      1,
			Role:    "user",
			Content: "What are some strategies for using multiple GPUs?",
		},
		{
			ID:      2,
			Role:    "assistant",
			Content: "You can split the model across GPUs or use a sidecar container to handle inference.",
		},
		{
			ID:      3,
			Role:    "user",
			Content: "Can a single model leverage VRAM across multiple GPUs?",
		},
		{
			ID:      4,
			Role:    "assistant",
			Content: "Yes, by bundling inference into one container that requests two GPUs, you can utilize VRAM across both devices.",
		},
	},
	SearchResults: []models.Message{},
}
