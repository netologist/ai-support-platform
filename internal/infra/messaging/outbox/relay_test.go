package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/infra/messaging/outbox"
	mockrepository "github.com/netologist/ai-support-platform/internal/mocks/repository"
	mockservice "github.com/netologist/ai-support-platform/internal/mocks/service"
)

func TestRelay_FlushOnce(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockrepository.MockOutboxRepository, *mockservice.MockMessagePublisher)
		wantErr   bool
	}{
		{
			name: "publishes and marks events sent",
			setupMock: func(repo *mockrepository.MockOutboxRepository, pub *mockservice.MockMessagePublisher) {
				eventID := uuid.New()
				aggID := uuid.New()
				repo.EXPECT().GetUnsentEvents(mock.Anything, 100).Return([]*entity.OutboxEvent{
					{ID: eventID, AggregateID: aggID, AggregateType: "ticket", EventType: "ticket.created", Payload: []byte(`{}`), CreatedAt: time.Now()},
				}, nil)
				pub.EXPECT().PublishJSON(mock.Anything, "tickets", aggID.String(), mock.Anything).Return(nil)
				repo.EXPECT().MarkEventSent(mock.Anything, eventID).Return(nil)
			},
		},
		{
			name: "no events is a no-op",
			setupMock: func(repo *mockrepository.MockOutboxRepository, pub *mockservice.MockMessagePublisher) {
				repo.EXPECT().GetUnsentEvents(mock.Anything, 100).Return(nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outboxRepo := mockrepository.NewMockOutboxRepository(t)
			publisher := mockservice.NewMockMessagePublisher(t)

			tt.setupMock(outboxRepo, publisher)

			relay := outbox.NewRelay(outboxRepo, publisher, "tickets", 100, time.Second)
			err := relay.FlushOnce(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
