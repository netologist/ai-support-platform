package chunking

import (
	"context"
	"strings"
	"unicode/utf8"
)

type SimpleChunker struct{}

func NewSimpleChunker() *SimpleChunker {
	return &SimpleChunker{}
}

func (c *SimpleChunker) Chunk(_ context.Context, content string, maxTokens int) ([]string, error) {
	if maxTokens <= 0 {
		maxTokens = 512
	}

	maxChars := maxTokens * 4

	paragraphs := strings.Split(content, "\n\n")

	var chunks []string
	var current strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		if current.Len()+utf8.RuneCountInString(para)+2 > maxChars && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}

		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, strings.TrimSpace(content))
	}

	return chunks, nil
}
