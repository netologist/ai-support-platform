package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewProviders_Fake(t *testing.T) {
	t.Parallel()

	providers, err := NewProviders(context.Background(), Config{Provider: "fake"})
	require.NoError(t, err)
	require.NotNil(t, providers.AIProvider)
	require.NotNil(t, providers.EmbeddingProvider)
	require.NoError(t, providers.Close())
}

func TestNewProviders_DefaultsToFake(t *testing.T) {
	t.Parallel()

	providers, err := NewProviders(context.Background(), Config{})
	require.NoError(t, err)
	require.NotNil(t, providers.AIProvider)
	require.NotNil(t, providers.EmbeddingProvider)
	require.NoError(t, providers.Close())
}

func TestNewProviders_EinoRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewProviders(context.Background(), Config{Provider: "eino"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "APP_AI_API_KEY")
}

func TestNewProviders_GenkitAliasRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewProviders(context.Background(), Config{Provider: "genkit"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "APP_AI_API_KEY")
}

func TestNewProviders_Unsupported(t *testing.T) {
	t.Parallel()

	_, err := NewProviders(context.Background(), Config{Provider: "unknown"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported AI provider")
}
