package service

import "context"

type AIProvider interface {
	GenerateReply(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
	Summarize(ctx context.Context, text string) (string, error)
	Classify(ctx context.Context, text string, categories []string) (string, error)
}
