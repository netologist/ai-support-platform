-- +goose Up
ALTER TABLE outbox
    ADD COLUMN attempt_count INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE outbox
    DROP COLUMN attempt_count;
