package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type Config struct {
	Provider       string
	Model          string
	EmbeddingModel string
	APIKey         string
}

type Providers struct {
	AIProvider        service.AIProvider
	EmbeddingProvider service.EmbeddingProvider
	Close             func() error
}

func NewProviders(ctx context.Context, cfg Config) (Providers, error) {
	providerName := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if providerName == "" {
		providerName = "fake"
	}

	switch providerName {
	case "fake":
		fakeProvider := NewFakeProvider()
		return Providers{
			AIProvider:        fakeProvider,
			EmbeddingProvider: fakeProvider,
			Close:             func() error { return nil },
		}, nil
	case "eino", "genkit":
		provider, err := NewEinoProvider(ctx, EinoConfig{
			Model:          cfg.Model,
			EmbeddingModel: cfg.EmbeddingModel,
			APIKey:         cfg.APIKey,
		})
		if err != nil {
			return Providers{}, err
		}

		return Providers{
			AIProvider:        provider,
			EmbeddingProvider: provider,
			Close:             provider.Close,
		}, nil
	default:
		return Providers{}, fmt.Errorf("unsupported AI provider %q", cfg.Provider)
	}
}
