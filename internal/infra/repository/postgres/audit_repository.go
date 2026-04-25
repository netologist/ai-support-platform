package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	generated "github.com/netologist/ai-support-platform/internal/infra/repository/sqlc"
)

type AuditRepository struct {
	queries *generated.Queries
}

func NewAuditRepository(queries *generated.Queries) AuditRepository {
	return AuditRepository{queries: queries}
}

func (repository AuditRepository) Insert(ctx context.Context, auditLog entity.AuditLog) error {
	metadata := auditLog.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	occurredAt := auditLog.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	identifier := auditLog.ID
	if identifier == uuid.Nil {
		identifier = uuid.New()
	}

	return repository.queries.InsertAuditLog(ctx, generated.InsertAuditLogParams{
		ID:         toPGUUID(identifier),
		OccurredAt: pgtype.Timestamptz{Time: occurredAt, Valid: true},
		EventType:  auditLog.EventType,
		Action:     auditLog.Action,
		Outcome:    auditLog.Outcome,
		TenantID:   toNullablePGUUID(auditLog.TenantID),
		UserID:     toNullablePGUUID(auditLog.UserID),
		Resource:   auditLog.Resource,
		ResourceID: auditLog.ResourceID,
		Metadata:   payload,
	})
}
