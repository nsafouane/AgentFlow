-- name: GetMigrationBaseline :one
SELECT id, created_at, description
FROM migration_baseline
WHERE id = $1;

-- name: ListMigrationBaselines :many
SELECT id, created_at, description
FROM migration_baseline
ORDER BY created_at DESC;

-- name: CreateMigrationBaseline :one
INSERT INTO migration_baseline (description)
VALUES ($1)
RETURNING id, created_at, description;