-- name: GetTicketByID :one
SELECT id, tenant_id, subject, status, created_by_user_id, assigned_to_user_id, created_at
FROM tickets
WHERE id = $1
LIMIT 1;

-- name: ListTicketsByTenant :many
SELECT id, tenant_id, subject, status, created_by_user_id, assigned_to_user_id, created_at
FROM tickets
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: CreateTicket :one
INSERT INTO tickets (
	id,
	tenant_id,
	subject,
	status,
	created_by_user_id,
	assigned_to_user_id,
	created_at
) VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7
)
RETURNING id, tenant_id, subject, status, created_by_user_id, assigned_to_user_id, created_at;

-- name: UpdateTicket :one
UPDATE tickets
SET subject = $2,
	status = $3,
	assigned_to_user_id = $4
WHERE id = $1
RETURNING id, tenant_id, subject, status, created_by_user_id, assigned_to_user_id, created_at;