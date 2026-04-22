package provider

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTPAddress            string
	DatabaseURL            string
	RedisAddress           string
	RedisPassword          string
	RedisDatabase          int
	AIProvider             string
	AIModel                string
	AIEmbeddingModel       string
	AIAPIKey               string
	KafkaBrokers           []string
	KafkaAuditTopic        string
	KafkaTicketTopic       string
	JWTIssuer              string
	JWTSecret              string
	JWTTTL                 time.Duration
	TicketCacheTTL         time.Duration
	RateLimitWindow        time.Duration
	PublicRateLimit        int64
	AuthenticatedRateLimit int64
}

type rawConfig struct {
	HTTPAddress            string        `envconfig:"APP_HTTP_ADDRESS" default:":8080"`
	DatabaseURL            string        `envconfig:"APP_DATABASE_URL"`
	RedisAddress           string        `envconfig:"APP_REDIS_ADDRESS" default:"localhost:6379"`
	RedisPassword          string        `envconfig:"APP_REDIS_PASSWORD"`
	RedisDatabase          int           `envconfig:"APP_REDIS_DB" default:"0"`
	AIProvider             string        `envconfig:"APP_AI_PROVIDER" default:"fake"`
	AIModel                string        `envconfig:"APP_AI_MODEL" default:"gemini-2.5-flash"`
	AIEmbeddingModel       string        `envconfig:"APP_AI_EMBEDDING_MODEL" default:"text-embedding-004"`
	AIAPIKey               string        `envconfig:"APP_AI_API_KEY"`
	GoogleAPIKey           string        `envconfig:"GOOGLE_API_KEY"`
	GeminiAPIKey           string        `envconfig:"GEMINI_API_KEY"`
	KafkaBrokers           []string      `envconfig:"APP_KAFKA_BROKERS" default:"localhost:9094"`
	KafkaAuditTopic        string        `envconfig:"APP_KAFKA_AUDIT_TOPIC" default:"audit.logged.v1"`
	KafkaTicketTopic       string        `envconfig:"APP_KAFKA_TICKET_TOPIC" default:"ticket.created.v1"`
	JWTIssuer              string        `envconfig:"APP_JWT_ISSUER" default:"ai-support-platform"`
	JWTSecret              string        `envconfig:"APP_JWT_SECRET"`
	JWTTTL                 time.Duration `envconfig:"APP_JWT_TTL" default:"15m"`
	TicketCacheTTL         time.Duration `envconfig:"APP_TICKET_CACHE_TTL" default:"1m"`
	RateLimitWindow        time.Duration `envconfig:"APP_RATE_LIMIT_WINDOW" default:"1m"`
	PublicRateLimit        int64         `envconfig:"APP_PUBLIC_RATE_LIMIT" default:"5"`
	AuthenticatedRateLimit int64         `envconfig:"APP_AUTHENTICATED_RATE_LIMIT" default:"60"`
}

func LoadConfig() (Config, error) {
	var raw rawConfig
	if err := envconfig.Process("", &raw); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}

	config := Config{
		HTTPAddress:            valueOrDefault(raw.HTTPAddress, ":8080"),
		DatabaseURL:            raw.DatabaseURL,
		RedisAddress:           valueOrDefault(raw.RedisAddress, "localhost:6379"),
		RedisPassword:          raw.RedisPassword,
		RedisDatabase:          raw.RedisDatabase,
		AIProvider:             valueOrDefault(raw.AIProvider, "fake"),
		AIModel:                valueOrDefault(raw.AIModel, "gemini-2.5-flash"),
		AIEmbeddingModel:       valueOrDefault(raw.AIEmbeddingModel, "text-embedding-004"),
		AIAPIKey:               firstNonEmpty(raw.AIAPIKey, raw.GoogleAPIKey, raw.GeminiAPIKey),
		KafkaBrokers:           valuesOrDefault(trimNonEmpty(raw.KafkaBrokers), []string{"localhost:9094"}),
		KafkaAuditTopic:        valueOrDefault(raw.KafkaAuditTopic, "audit.logged.v1"),
		KafkaTicketTopic:       valueOrDefault(raw.KafkaTicketTopic, "ticket.created.v1"),
		JWTIssuer:              valueOrDefault(raw.JWTIssuer, "ai-support-platform"),
		JWTSecret:              raw.JWTSecret,
		JWTTTL:                 raw.JWTTTL,
		TicketCacheTTL:         raw.TicketCacheTTL,
		RateLimitWindow:        raw.RateLimitWindow,
		PublicRateLimit:        raw.PublicRateLimit,
		AuthenticatedRateLimit: raw.AuthenticatedRateLimit,
	}

	if config.DatabaseURL == "" {
		return Config{}, fmt.Errorf("APP_DATABASE_URL is required")
	}

	if config.JWTSecret == "" {
		return Config{}, fmt.Errorf("APP_JWT_SECRET is required")
	}

	if (strings.EqualFold(config.AIProvider, "eino") || strings.EqualFold(config.AIProvider, "genkit")) && config.AIAPIKey == "" {
		return Config{}, fmt.Errorf("APP_AI_API_KEY or GOOGLE_API_KEY is required when APP_AI_PROVIDER=eino")
	}

	return config, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func trimNonEmpty(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		cleaned := strings.TrimSpace(value)
		if cleaned != "" {
			trimmed = append(trimmed, cleaned)
		}
	}

	return trimmed
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func valuesOrDefault(values []string, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}

	return values
}
