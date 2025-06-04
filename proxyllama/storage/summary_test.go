package storage

import (
	"context"
	"slices"
	"testing"
)

var (
	cid       = 1
	sourceIDs = []int{1, 2, 3, 4, 5}
	content   = `by bundling the inference for both LLMs and image generation into one container (or sidecar) that requests two GPUs, you ensure that model splitting can occur across both devices. This approach meets your requirement while keeping the gRPC interface separate from the inference logic, just as before—but now with full multi-GPU utilization available in a single process.`
	level     = 1
	message1  = "What are some strategies for using multiple GPUs?"
	message2  = "You can split the model across GPUs or use a sidecar container to handle inference."
	message3  = "Can a single model leverage VRAM across multiple GPUs?"
	message4  = "Yes, by bundling inference into one container that requests two GPUs, you can utilize VRAM across both devices."
	message5  = "How can I ensure that my LLM and image generation models can run in parallel on multiple GPUs?"
)

func Test_CreateSummary(t *testing.T) {
	ctx := context.Background()
	Init()

	// Save the summary to the database
	id, err := SummaryStoreInstance.CreateSummary(ctx, cid, content, level, sourceIDs)
	if err != nil {
		t.Fatalf("Failed to create summary: %v", err)
	}

	// Retrieve the summary from the database
	retrievedSummary, err := SummaryStoreInstance.GetSummary(ctx, id)
	if err != nil {
		t.Fatalf("Failed to retrieve summary: %v", err)
	}

	// Check if the retrieved summary matches the original
	if retrievedSummary.Content != content {
		t.Errorf("Expected text '%s', got '%s'", content, retrievedSummary.Content)
	}

	if retrievedSummary.SourceIDs == nil || len(retrievedSummary.SourceIDs) != len(sourceIDs) {
		t.Errorf("Expected source IDs %v, got %v", sourceIDs, retrievedSummary.SourceIDs)
	}
	for i, id := range retrievedSummary.SourceIDs {
		if slices.Contains(sourceIDs, id) {
			continue
		}
		t.Errorf("Expected source ID %d at index %d, but it was not found in %v", id, i, sourceIDs)
	}

	Pool.Exec(ctx, `DELETE from summaries WHERE id = $1`, id)
	if err != nil {
		t.Errorf("failed to delete summary: %s", err.Error())
	}
}
