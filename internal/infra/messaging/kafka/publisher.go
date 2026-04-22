package kafka

import (
	"context"
	"encoding/json"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Publisher struct {
	client *kgo.Client
}

func NewPublisher(brokers []string) (Publisher, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return Publisher{}, err
	}

	return Publisher{client: client}, nil
}

func (publisher Publisher) PublishJSON(ctx context.Context, topic string, key string, payload any) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	record := &kgo.Record{Topic: topic, Value: message}
	if key != "" {
		record.Key = []byte(key)
	}

	return publisher.client.ProduceSync(ctx, record).FirstErr()
}

func (publisher Publisher) Close() {
	if publisher.client != nil {
		publisher.client.Close()
	}
}
