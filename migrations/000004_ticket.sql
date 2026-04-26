-- +goose Up
CREATE TABLE tickets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    assigned_to_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    subject TEXT NOT NULL,
    status TEXT NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX tickets_tenant_id_idx ON tickets (tenant_id);

-- +goose Down
DROP INDEX IF EXISTS tickets_tenant_id_idx;
DROP TABLE IF EXISTS tickets;
