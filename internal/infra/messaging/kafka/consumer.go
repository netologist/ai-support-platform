package kafka

import (
	"context"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

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

			if err := handler(ctx, record.Topic, record.Key, record.Value); err != nil {
				slog.Error("handler failed",
					slog.String("topic", record.Topic),
					slog.String("key", string(record.Key)),
					slog.Any("error", err),
				)
				return
			}
		})

		if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
			slog.Error("commit offsets failed", slog.Any("error", err))
		}
	}
}

func (c *Consumer) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
