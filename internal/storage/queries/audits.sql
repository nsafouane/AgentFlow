-- name: CreateAudit :one
INSERT INTO audits (tenant_id, actor_type, actor_id, action, resource_type, resource_id, details, prev_hash, hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetAudit :one
SELECT * FROM audits
WHERE id = $1 AND tenant_id = $2;

-- name: ListAuditsByTenant :many
SELECT * FROM audits
WHERE tenant_id = $1
ORDER BY ts DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditsByActor :many
SELECT * FROM audits
WHERE tenant_id = $1 AND actor_type = $2 AND actor_id = $3
ORDER BY ts DESC
LIMIT $4 OFFSET $5;

-- name: ListAuditsByResource :many
SELECT * FROM audits
WHERE tenant_id = $1 AND resource_type = $2 AND resource_id = $3
ORDER BY ts DESC
LIMIT $4 OFFSET $5;

-- name: GetAuditChain :many
SELECT * FROM audits
WHERE tenant_id = $1
ORDER BY ts ASC;

-- name: GetLatestAudit :one
SELECT * FROM audits
WHERE tenant_id = $1
ORDER BY ts DESC
LIMIT 1;