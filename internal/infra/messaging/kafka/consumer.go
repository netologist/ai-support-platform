package kafka

import (
	"context"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

// maxHandlerRetries is the number of times a handler is retried before a
// record is dead-lettered (committed and skipped). This prevents a persistently
// failing message from blocking the consumer indefinitely.
const maxHandlerRetries = 3

type Handler func(ctx context.Context, topic string, key []byte, value []byte) error

type Consumer struct {
	client   *kgo.Client
	handlers map[string]Handler
}

func NewConsumer(brokers []string, group string, topics []string) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		client:   client,
		handlers: make(map[string]Handler),
	}, nil
}

func (c *Consumer) RegisterHandler(topic string, handler Handler) {
	c.handlers[topic] = handler
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				slog.Error("kafka fetch error", slog.String("topic", e.Topic), slog.Any("error", e.Err))
			}
		}

		fetches.EachRecord(func(record *kgo.Record) {
			handler, ok := c.handlers[record.Topic]
			if !ok {
				slog.Warn("no handler for topic", slog.String("topic", record.Topic))
				return
			}

			c.processWithRetry(ctx, handler, record)
		})

		if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
			slog.Error("commit offsets failed", slog.Any("error", err))
		}
	}
}

// processWithRetry calls the handler up to maxHandlerRetries times. On
// persistent failure the record is dead-lettered: logged at ERROR level so an
// alerting rule can fire, and the offset is committed on the next
// CommitUncommittedOffsets call so the consumer does not loop forever.
func (c *Consumer) processWithRetry(ctx context.Context, handler Handler, record *kgo.Record) {
	var lastErr error
	for attempt := 1; attempt <= maxHandlerRetries; attempt++ {
		if lastErr = handler(ctx, record.Topic, record.Key, record.Value); lastErr == nil {
			return
		}
		slog.Warn("handler failed, will retry",
			slog.String("topic", record.Topic),
			slog.String("key", string(record.Key)),
			slog.Int64("offset", record.Offset),
			slog.Int("attempt", attempt),
			slog.Int("max_retries", maxHandlerRetries),
			slog.Any("error", lastErr),
		)
	}

	// All retries exhausted — dead-letter: log the full record context so an
	// operator can inspect and replay from the Kafka topic if needed. The offset
	// is committed by the surrounding CommitUncommittedOffsets so the consumer
	// advances past this record and does not stall.
	slog.Error("handler failed after max retries, dead-lettering record",
		slog.String("topic", record.Topic),
		slog.String("key", string(record.Key)),
		slog.Int64("partition", int64(record.Partition)),
		slog.Int64("offset", record.Offset),
		slog.Any("error", lastErr),
	)
}

func (c *Consumer) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

