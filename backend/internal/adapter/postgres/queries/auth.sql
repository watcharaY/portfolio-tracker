-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, must_change_password)
VALUES ($1, $2, $3)
RETURNING id;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $1, must_change_password = $2, updated_at = now()
WHERE id = $3
RETURNING id;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, ip, user_agent, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetValidSession :one
SELECT * FROM sessions WHERE token_hash = $1 AND expires_at > $2;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteOtherSessions :exec
DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= $1;
