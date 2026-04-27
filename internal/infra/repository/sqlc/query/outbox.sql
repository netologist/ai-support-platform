-- name: InsertOutboxEvent :exec
INSERT INTO outbox (
	id,
	aggregate_type,
	aggregate_id,
	event_type,
	payload
) VALUES (
	$1, $2, $3, $4, $5
);

-- name: GetUnsentOutboxEvents :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, attempt_count
FROM outbox
WHERE sent_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkOutboxEventSent :exec
UPDATE outbox SET sent_at = NOW() WHERE id = $1;

-- name: IncrementOutboxEventAttempt :exec
UPDATE outbox SET attempt_count = attempt_count + 1 WHERE id = $1;
