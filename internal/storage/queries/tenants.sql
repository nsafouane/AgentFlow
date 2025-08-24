-- name: CreateTenant :one
INSERT INTO tenants (name, tier, settings)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants
WHERE id = $1;

-- name: GetTenantByName :one
SELECT * FROM tenants
WHERE name = $1;

-- name: ListTenants :many
SELECT * FROM tenants
ORDER BY created_at DESC;

-- name: UpdateTenant :one
UPDATE tenants
SET name = $2, tier = $3, settings = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTenant :exec
DELETE FROM tenants
WHERE id = $1;