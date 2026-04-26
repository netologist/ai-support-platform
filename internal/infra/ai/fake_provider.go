package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type FakeProvider struct{}

func NewFakeProvider() *FakeProvider {
	return &FakeProvider{}
}

func (p *FakeProvider) GenerateReply(_ context.Context, _ string, userPrompt string) (string, error) {
	return fmt.Sprintf("Thank you for reaching out. Regarding: %q — we will look into this shortly.", truncate(userPrompt, 100)), nil
}

func (p *FakeProvider) Summarize(_ context.Context, text string) (string, error) {
	return fmt.Sprintf("Summary of %d characters of text.", len(text)), nil
}

func (p *FakeProvider) Classify(_ context.Context, _ string, categories []string) (string, error) {
	if len(categories) > 0 {
		return categories[0], nil
	}
	return "general", nil
}

func (p *FakeProvider) Embed(_ context.Context, _ string) ([]float32, error) {
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = 0.01
	}
	return embedding, nil
}

func (p *FakeProvider) ModelName() string {
	return "fake-embedding-v1"
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

var _ service.AIProvider = (*FakeProvider)(nil)
var _ service.EmbeddingProvider = (*FakeProvider)(nil)
