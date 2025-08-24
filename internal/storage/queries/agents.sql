-- name: CreateAgent :one
INSERT INTO agents (tenant_id, name, type, role, config_json, policies_json)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAgent :one
SELECT * FROM agents
WHERE id = $1 AND tenant_id = $2;

-- name: GetAgentByName :one
SELECT * FROM agents
WHERE tenant_id = $1 AND name = $2;

-- name: ListAgentsByTenant :many
SELECT * FROM agents
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: ListAgentsByType :many
SELECT * FROM agents
WHERE tenant_id = $1 AND type = $2
ORDER BY created_at DESC;

-- name: UpdateAgent :one
UPDATE agents
SET name = $3, type = $4, role = $5, config_json = $6, policies_json = $7, updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteAgent :exec
DELETE FROM agents
WHERE id = $1 AND tenant_id = $2;