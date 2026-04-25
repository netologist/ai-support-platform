-- +goose Up
INSERT INTO users (id, email, password_hash)
VALUES (
	'11111111-1111-1111-1111-111111111111',
	'admin+local@example.com',
	'$2a$10$HN3mowtVXnNUpk/vTaELuusU6ZGx0O61qafDpplYVII7gnCdi3sOe' -- bcrypt hash for "password"
)
ON CONFLICT (email) DO UPDATE
SET password_hash = EXCLUDED.password_hash;

INSERT INTO tenants (id, name)
VALUES (
	'22222222-2222-2222-2222-222222222222',
	'Default Tenant'
)
ON CONFLICT (name) DO NOTHING;

INSERT INTO memberships (tenant_id, user_id, role)
VALUES (
	'22222222-2222-2222-2222-222222222222',
	'11111111-1111-1111-1111-111111111111',
	'admin'
)
ON CONFLICT (tenant_id, user_id) DO NOTHING;

-- +goose Down
DELETE FROM memberships
WHERE tenant_id = '22222222-2222-2222-2222-222222222222' AND user_id = '11111111-1111-1111-1111-111111111111';
DELETE FROM tenants
WHERE id = '22222222-2222-2222-2222-222222222222';
DELETE FROM users
WHERE email = 'admin+local@example.com';
