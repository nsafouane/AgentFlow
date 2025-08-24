-- name: CreateUser :one
INSERT INTO users (tenant_id, email, role, hashed_secret)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 AND tenant_id = $2;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE tenant_id = $1 AND email = $2;

-- name: ListUsersByTenant :many
SELECT * FROM users
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET email = $3, role = $4, hashed_secret = $5, updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1 AND tenant_id = $2;