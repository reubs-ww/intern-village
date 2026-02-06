-- Projects SQL queries
-- Reference: specs/orchestrator.md §4.2

-- name: CreateProject :one
INSERT INTO projects (
    user_id,
    github_owner,
    github_repo,
    is_fork,
    upstream_owner,
    upstream_repo,
    default_branch,
    clone_path,
    beads_prefix
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetProjectByID :one
SELECT * FROM projects
WHERE id = $1 LIMIT 1;

-- name: GetProjectByOwnerRepo :one
SELECT * FROM projects
WHERE user_id = $1 AND github_owner = $2 AND github_repo = $3
LIMIT 1;

-- name: ListProjectsByUser :many
SELECT * FROM projects
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET github_owner = $2,
    github_repo = $3,
    is_fork = $4,
    upstream_owner = $5,
    upstream_repo = $6,
    default_branch = $7,
    clone_path = $8,
    beads_prefix = $9,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;
