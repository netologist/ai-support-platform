-- +goose Up
INSERT INTO users (id, email, password_hash)
VALUES (
	'11111111-1111-1111-1111-111111111111',
	'admin+local@example.com',
	'$2a$10$HN3mowtVXnNUpk/vTaELuusU6ZGx0O61qafDpplYVII7gnCdi3sOe' -- bcrypt hash for "password"
)
ON CONFLICT (email) DO UPDATE
SET password_hash = EXCLUDED.password_hash;

-- +goose Down
DELETE FROM users
WHERE email = 'admin+local@example.com';
