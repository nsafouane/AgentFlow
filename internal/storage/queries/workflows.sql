-- name: CreateWorkflow :one
INSERT INTO workflows (tenant_id, name, version, config_yaml, planner_type, template_version_constraint)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetWorkflow :one
SELECT * FROM workflows
WHERE id = $1 AND tenant_id = $2;

-- name: GetWorkflowByNameVersion :one
SELECT * FROM workflows
WHERE tenant_id = $1 AND name = $2 AND version = $3;

-- name: ListWorkflowsByTenant :many
SELECT * FROM workflows
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: ListWorkflowsByPlanner :many
SELECT * FROM workflows
WHERE tenant_id = $1 AND planner_type = $2
ORDER BY created_at DESC;

-- name: UpdateWorkflow :one
UPDATE workflows
SET name = $3, version = $4, config_yaml = $5, planner_type = $6, template_version_constraint = $7, updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM workflows
WHERE id = $1 AND tenant_id = $2;