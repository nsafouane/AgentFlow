-- name: CreateMessage :one
INSERT INTO messages (id, tenant_id, trace_id, span_id, from_agent, to_agent, type, payload, metadata, cost, ts, envelope_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetMessage :one
SELECT * FROM messages
WHERE id = $1 AND tenant_id = $2;

-- name: ListMessagesByTenant :many
SELECT * FROM messages
WHERE tenant_id = $1
ORDER BY ts DESC
LIMIT $2 OFFSET $3;

-- name: ListMessagesByTrace :many
SELECT * FROM messages
WHERE tenant_id = $1 AND trace_id = $2
ORDER BY ts ASC;

-- name: ListMessagesByAgent :many
SELECT * FROM messages
WHERE tenant_id = $1 AND (from_agent = $2 OR to_agent = $2)
ORDER BY ts DESC
LIMIT $3 OFFSET $4;

-- name: ListMessagesByTimeRange :many
SELECT * FROM messages
WHERE tenant_id = $1 AND ts BETWEEN $2 AND $3
ORDER BY ts DESC
LIMIT $4 OFFSET $5;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = $1 AND tenant_id = $2;