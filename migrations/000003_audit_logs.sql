-- +goose Up
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL,
    event_type TEXT NOT NULL,
    action TEXT NOT NULL,
    outcome TEXT NOT NULL,
    tenant_id UUID REFERENCES tenants (id) ON DELETE SET NULL,
    user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    resource TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX audit_logs_occurred_at_idx ON audit_logs (occurred_at DESC);
CREATE INDEX audit_logs_tenant_id_idx ON audit_logs (tenant_id);
CREATE INDEX audit_logs_user_id_idx ON audit_logs (user_id);

-- +goose Down
DROP INDEX IF EXISTS audit_logs_user_id_idx;
DROP INDEX IF EXISTS audit_logs_tenant_id_idx;
DROP INDEX IF EXISTS audit_logs_occurred_at_idx;
DROP TABLE IF EXISTS audit_logs;
