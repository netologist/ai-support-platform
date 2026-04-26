package service

import "context"

type Chunker interface {
	Chunk(ctx context.Context, content string, maxTokens int) ([]string, error)
}
