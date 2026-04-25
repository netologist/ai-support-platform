package audit

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/repository"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

type Logger struct {
	repository repository.AuditRepository
	publisher  service.MessagePublisher
	topic      string
}

func NewLogger(repository repository.AuditRepository, publisher service.MessagePublisher, topic string) Logger {
	return Logger{repository: repository, publisher: publisher, topic: topic}
}

func (logger Logger) Record(ctx context.Context, entry entity.AuditLog) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = time.Now().UTC()
	}

	if err := logger.repository.Insert(ctx, entry); err != nil {
		return err
	}

	if logger.publisher != nil && logger.topic != "" {
		if err := logger.publisher.PublishJSON(ctx, logger.topic, entry.ID.String(), entry); err != nil {
			return err
		}
	}

	return nil
}
