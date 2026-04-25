package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig_DefaultsToFakeProvider(t *testing.T) {
	t.Setenv("APP_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ai_support_platform?sslmode=disable")
	t.Setenv("APP_JWT_SECRET", "secret")
	t.Setenv("APP_AI_PROVIDER", "")
	t.Setenv("APP_AI_API_KEY", "")
    t.Setenv("APP_AI_MODEL", "gemini-2.5-flash")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	config, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "fake", config.AIProvider)
	require.Empty(t, config.AIAPIKey)
	require.Equal(t, "gemini-2.5-flash", config.AIModel)
	require.Equal(t, "text-embedding-004", config.AIEmbeddingModel)
}

func TestLoadConfig_EinoRequiresAPIKey(t *testing.T) {
	t.Setenv("APP_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ai_support_platform?sslmode=disable")
	t.Setenv("APP_JWT_SECRET", "secret")
	t.Setenv("APP_AI_PROVIDER", "eino")
	t.Setenv("APP_AI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	_, err := LoadConfig()
	require.Error(t, err)
	require.Contains(t, err.Error(), "APP_AI_API_KEY")
}

func TestLoadConfig_EinoUsesGoogleAPIKeyFallback(t *testing.T) {
	t.Setenv("APP_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ai_support_platform?sslmode=disable")
	t.Setenv("APP_JWT_SECRET", "secret")
	t.Setenv("APP_AI_PROVIDER", "eino")
	t.Setenv("APP_AI_API_KEY", "")
    t.Setenv("APP_AI_MODEL", "gemini-2.5-flash")
	t.Setenv("GOOGLE_API_KEY", "google-key")
	t.Setenv("GEMINI_API_KEY", "")

	config, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "eino", config.AIProvider)
	require.Equal(t, "google-key", config.AIAPIKey)
	require.Equal(t, "gemini-2.5-flash", config.AIModel)
	require.Equal(t, "text-embedding-004", config.AIEmbeddingModel)
}

func TestLoadConfig_TrimsKafkaBrokers(t *testing.T) {
	t.Setenv("APP_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ai_support_platform?sslmode=disable")
	t.Setenv("APP_JWT_SECRET", "secret")
	t.Setenv("APP_KAFKA_BROKERS", " broker-1:9092, broker-2:9092 , ,broker-3:9092 ")

	config, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, []string{"broker-1:9092", "broker-2:9092", "broker-3:9092"}, config.KafkaBrokers)
}
