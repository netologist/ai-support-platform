package service

import "context"

type MessagePublisher interface {
	PublishJSON(ctx context.Context, topic string, key string, payload any) error
}
