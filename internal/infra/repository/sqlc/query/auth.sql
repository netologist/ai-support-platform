-- name: GetUserByEmail :one
SELECT id, email, password_hash, created_at
FROM users
WHERE email = $1
LIMIT 1;

-- name: GetMembershipByUserAndTenant :one
SELECT tenant_id, user_id, role, created_at
FROM memberships
WHERE user_id = $1 AND tenant_id = $2
LIMIT 1;