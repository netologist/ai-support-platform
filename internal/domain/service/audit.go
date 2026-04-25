package service

import (
	"context"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
)

type AuditLogger interface {
	Record(ctx context.Context, entry entity.AuditLog) error
}
