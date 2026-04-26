package query

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type SuggestReplyQuery struct {
	Principal entity.Principal
	TicketID  uuid.UUID
}

type SuggestReplyResult struct {
	SuggestedReply string
	Sources        []entity.DocumentChunk
}

type SuggestReplyService struct {
	ticketRepo repository.TicketRepository
	chunkRepo  repository.ChunkRepository
	embedder   service.EmbeddingProvider
	aiProvider service.AIProvider
	authorizer service.Authorizer
}

func NewSuggestReplyService(
	ticketRepo repository.TicketRepository,
	chunkRepo repository.ChunkRepository,
	embedder service.EmbeddingProvider,
	aiProvider service.AIProvider,
	authorizer service.Authorizer,
) SuggestReplyService {
	return SuggestReplyService{
		ticketRepo: ticketRepo,
		chunkRepo:  chunkRepo,
		embedder:   embedder,
		aiProvider: aiProvider,
		authorizer: authorizer,
	}
}

func (svc SuggestReplyService) Execute(ctx context.Context, q SuggestReplyQuery) (SuggestReplyResult, error) {
	if err := svc.authorizer.Authorize(ctx, q.Principal, "tickets", "read"); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			return SuggestReplyResult{}, apperrors.ErrForbidden
		}
		return SuggestReplyResult{}, err
	}

	ticket, err := svc.ticketRepo.GetByID(ctx, q.TicketID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return SuggestReplyResult{}, apperrors.ErrNotFound
		}
		return SuggestReplyResult{}, err
	}

	if ticket.TenantID != q.Principal.TenantID {
		return SuggestReplyResult{}, apperrors.ErrForbidden
	}

	embedding, err := svc.embedder.Embed(ctx, ticket.Subject)
	if err != nil {
		return SuggestReplyResult{}, fmt.Errorf("embed ticket subject: %w", err)
	}

	chunks, err := svc.chunkRepo.SearchSimilar(ctx, ticket.TenantID, embedding, 3)
	if err != nil {
		return SuggestReplyResult{}, fmt.Errorf("search similar chunks: %w", err)
	}

	var contextParts []string
	for _, chunk := range chunks {
		contextParts = append(contextParts, chunk.Content)
	}

	systemPrompt := "You are a helpful support agent. Use the following knowledge base context to help answer the customer's question. Be concise and helpful."
	if len(contextParts) > 0 {
		systemPrompt += "\n\nContext:\n" + strings.Join(contextParts, "\n---\n")
	}

	reply, err := svc.aiProvider.GenerateReply(ctx, systemPrompt, ticket.Subject)
	if err != nil {
		return SuggestReplyResult{}, fmt.Errorf("generate reply: %w", err)
	}

	return SuggestReplyResult{
		SuggestedReply: reply,
		Sources:        chunks,
	}, nil
}
