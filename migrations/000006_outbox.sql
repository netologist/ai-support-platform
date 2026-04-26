-- +goose Up
CREATE TABLE outbox (
    id UUID PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ
);
CREATE INDEX outbox_aggregate_idx ON outbox (aggregate_type, aggregate_id);
CREATE INDEX outbox_event_type_idx ON outbox (event_type);

-- +goose Down
DROP INDEX IF EXISTS outbox_event_type_idx;
DROP INDEX IF EXISTS outbox_aggregate_idx;
DROP TABLE IF EXISTS outbox;
