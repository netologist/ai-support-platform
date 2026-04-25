package repository

import (
	"context"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type AuditRepository interface {
	Insert(ctx context.Context, auditLog entity.AuditLog) error
}
