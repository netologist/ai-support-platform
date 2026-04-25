-- name: InsertAuditLog :exec
INSERT INTO audit_logs (
    id,
    occurred_at,
    event_type,
    action,
    outcome,
    tenant_id,
    user_id,
    resource,
    resource_id,
    metadata
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10
);