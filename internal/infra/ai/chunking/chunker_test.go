package chunking_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/infra/ai/chunking"
)

func TestSimpleChunker_Chunk(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		maxTokens int
		wantLen   int
	}{
		{
			name:      "splits paragraphs into chunks",
			content:   "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
			maxTokens: 512,
			wantLen:   1, // all fit in one chunk at 512*4 chars
		},
		{
			name:      "single paragraph",
			content:   "Just one paragraph here.",
			maxTokens: 512,
			wantLen:   1,
		},
		{
			name:      "empty content returns empty",
			content:   "",
			maxTokens: 512,
			wantLen:   0,
		},
		{
			name:      "whitespace only returns empty",
			content:   "   \n\n   ",
			maxTokens: 512,
			wantLen:   0,
		},
		{
			name:      "forces split on small max tokens",
			content:   "Paragraph one.\n\nParagraph two.",
			maxTokens: 1, // 4 chars max
			wantLen:   2,
		},
		{
			name:      "default max tokens when zero",
			content:   "Some content.",
			maxTokens: 0,
			wantLen:   1,
		},
		{
			name:      "large document splits into multiple chunks",
			content:   strings.Repeat("Word. ", 1000) + "\n\n" + strings.Repeat("Another. ", 1000),
			maxTokens: 10,
			wantLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker := chunking.NewSimpleChunker()
			chunks, err := chunker.Chunk(context.Background(), tt.content, tt.maxTokens)

			require.NoError(t, err)
			assert.Len(t, chunks, tt.wantLen)

			for _, chunk := range chunks {
				assert.NotEmpty(t, strings.TrimSpace(chunk))
			}
		})
	}
}
