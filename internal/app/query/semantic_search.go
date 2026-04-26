package query

import (
	"context"
	"errors"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
)

type SemanticSearchQuery struct {
	Principal entity.Principal
	Text      string
	Limit     int
}

type SemanticSearchResult struct {
	Chunks []entity.DocumentChunk
}

type SemanticSearchService struct {
	chunkRepo  repository.ChunkRepository
	embedder   service.EmbeddingProvider
	authorizer service.Authorizer
}

func NewSemanticSearchService(
	chunkRepo repository.ChunkRepository,
	embedder service.EmbeddingProvider,
	authorizer service.Authorizer,
) SemanticSearchService {
	return SemanticSearchService{
		chunkRepo:  chunkRepo,
		embedder:   embedder,
		authorizer: authorizer,
	}
}

func (svc SemanticSearchService) Execute(ctx context.Context, q SemanticSearchQuery) (SemanticSearchResult, error) {
	if err := svc.authorizer.Authorize(ctx, q.Principal, "documents", "read"); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			return SemanticSearchResult{}, apperrors.ErrForbidden
		}
		return SemanticSearchResult{}, err
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 5
	}

	embedding, err := svc.embedder.Embed(ctx, q.Text)
	if err != nil {
		return SemanticSearchResult{}, err
	}

	chunks, err := svc.chunkRepo.SearchSimilar(ctx, q.Principal.TenantID, embedding, limit)
	if err != nil {
		return SemanticSearchResult{}, err
	}

	return SemanticSearchResult{Chunks: chunks}, nil
}
