package ai

import (
	"context"
	"fmt"
	"strings"

	einoembedding "github.com/cloudwego/eino-ext/components/embedding/gemini"
	einomodel "github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/schema"
	"google.golang.org/genai"

	"github.com/netologist/ai-support-platform/internal/domain/service"
)

const (
	defaultEinoModel          = "gemini-2.5-flash"
	defaultEinoEmbeddingModel = "text-embedding-004"
)

type EinoConfig struct {
	Model          string
	EmbeddingModel string
	APIKey         string
}

type EinoProvider struct {
	client         *genai.Client
	chatModel      *einomodel.ChatModel
	embedder       *einoembedding.Embedder
	model          string
	embeddingModel string
}

func NewEinoProvider(ctx context.Context, cfg EinoConfig) (*EinoProvider, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("APP_AI_API_KEY is required when APP_AI_PROVIDER=eino")
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultEinoModel
	}

	embeddingModel := strings.TrimSpace(cfg.EmbeddingModel)
	if embeddingModel == "" {
		embeddingModel = defaultEinoEmbeddingModel
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	chatModel, err := einomodel.NewChatModel(ctx, &einomodel.Config{
		Client: client,
		Model:  model,
	})
	if err != nil {
		return nil, fmt.Errorf("create eino chat model: %w", err)
	}

	embedder, err := einoembedding.NewEmbedder(ctx, &einoembedding.EmbeddingConfig{
		Client:   client,
		Model:    embeddingModel,
		TaskType: "RETRIEVAL_QUERY",
	})
	if err != nil {
		return nil, fmt.Errorf("create eino embedder: %w", err)
	}

	return &EinoProvider{
		client:         client,
		chatModel:      chatModel,
		embedder:       embedder,
		model:          model,
		embeddingModel: embeddingModel,
	}, nil
}

func (p *EinoProvider) GenerateReply(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	resp, err := p.chatModel.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: strings.TrimSpace(systemPrompt)},
		{Role: schema.User, Content: strings.TrimSpace(userPrompt)},
	})
	if err != nil {
		return "", fmt.Errorf("eino generate reply: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

func (p *EinoProvider) Summarize(ctx context.Context, text string) (string, error) {
	resp, err := p.chatModel.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: "Summarize support content concisely and accurately."},
		{Role: schema.User, Content: strings.TrimSpace(text)},
	})
	if err != nil {
		return "", fmt.Errorf("eino summarize: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

func (p *EinoProvider) Classify(ctx context.Context, text string, categories []string) (string, error) {
	resp, err := p.chatModel.Generate(ctx, []*schema.Message{
		{
			Role: schema.System,
			Content: fmt.Sprintf(
				"Classify the input into exactly one of these categories and return only the category name: %s",
				strings.Join(categories, ", "),
			),
		},
		{Role: schema.User, Content: strings.TrimSpace(text)},
	})
	if err != nil {
		return "", fmt.Errorf("eino classify: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

func (p *EinoProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := p.embedder.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("eino embed: %w", err)
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("eino embed: no embeddings returned")
	}

	result := make([]float32, len(vectors[0]))
	for index, value := range vectors[0] {
		result[index] = float32(value)
	}

	return result, nil
}

func (p *EinoProvider) ModelName() string {
	return p.embeddingModel
}

func (p *EinoProvider) Close() error {
	return nil
}

var _ service.AIProvider = (*EinoProvider)(nil)
var _ service.EmbeddingProvider = (*EinoProvider)(nil)
