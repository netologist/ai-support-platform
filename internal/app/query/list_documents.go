package query

import (
	"context"
	"errors"

	apperrors "github.com/netologist/ai-support-platform/internal/app/errors"
	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type ListDocumentsQuery struct {
	Principal entity.Principal
}

type ListDocumentsService struct {
	documentRepository repository.DocumentRepository
	authorizer         service.Authorizer
	auditLogger        service.AuditLogger
}

func NewListDocumentsService(documentRepository repository.DocumentRepository, authorizer service.Authorizer, auditLogger service.AuditLogger) ListDocumentsService {
	return ListDocumentsService{documentRepository: documentRepository, authorizer: authorizer, auditLogger: auditLogger}
}

func (svc ListDocumentsService) Execute(ctx context.Context, query ListDocumentsQuery) ([]entity.KnowledgeDocument, error) {
	if err := svc.authorizer.Authorize(ctx, query.Principal, "documents", "read"); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			return nil, apperrors.ErrForbidden
		}

		return nil, err
	}

	documents, err := svc.documentRepository.ListDocumentsByTenant(ctx, query.Principal.TenantID)
	if err != nil {
		return nil, err
	}

	if svc.auditLogger != nil {
		_ = svc.auditLogger.Record(ctx, entity.AuditLog{
			EventType: "document.list",
			Action:    "list",
			Outcome:   "success",
			TenantID:  &query.Principal.TenantID,
			UserID:    &query.Principal.UserID,
			Resource:  "document",
			Metadata:  map[string]any{"count": len(documents)},
		})
	}

	return documents, nil
}
