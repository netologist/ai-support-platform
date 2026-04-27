package worker_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netologist/ai-support-platform/internal/infra/messaging/worker"
	"github.com/netologist/ai-support-platform/internal/infra/messaging/worker/consumers"
	mockworker "github.com/netologist/ai-support-platform/internal/mocks/worker"
)

func TestTicketEventHandler_Handle(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     map[string]any
		setupMock func(*mockworker.MockIdempotencyChecker)
		wantErr   bool
	}{
		{
			name:  "processes ticket.created event",
			key:   "ticket-123",
			value: map[string]any{"event_type": "ticket.created"},
			setupMock: func(m *mockworker.MockIdempotencyChecker) {
				m.EXPECT().IsProcessed(context.Background(), "tickets:ticket-123").Return(false, nil)
				m.EXPECT().MarkProcessed(context.Background(), "tickets:ticket-123").Return(nil)
			},
		},
		{
			name:  "skips already processed event",
			key:   "ticket-456",
			value: map[string]any{"event_type": "ticket.updated"},
			setupMock: func(m *mockworker.MockIdempotencyChecker) {
				m.EXPECT().IsProcessed(context.Background(), "tickets:ticket-456").Return(true, nil)
			},
		},
		{
			name:      "fails on invalid json",
			key:       "ticket-789",
			value:     nil,
			setupMock: func(m *mockworker.MockIdempotencyChecker) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idempotency := mockworker.NewMockIdempotencyChecker(t)
			tt.setupMock(idempotency)

			handler := consumers.NewTicketEventHandler(idempotency)

			var valueBytes []byte
			if tt.value != nil {
				valueBytes, _ = json.Marshal(tt.value)
			} else {
				valueBytes = []byte("invalid json")
			}

			err := handler.Handle(context.Background(), "tickets", []byte(tt.key), valueBytes)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestPool_StartAndStop(t *testing.T) {
	pool := worker.NewPool(2, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)

	executed := make(chan struct{}, 1)
	err := pool.Submit(ctx, func(ctx context.Context) error {
		executed <- struct{}{}
		return nil
	})
	require.NoError(t, err)

	<-executed
	pool.Stop()

	assert.True(t, true, "pool started and stopped without deadlock")
}
