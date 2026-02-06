-- Tasks SQL queries
-- Reference: specs/orchestrator.md §4.3

-- name: CreateTask :one
INSERT INTO tasks (
    project_id,
    title,
    description,
    status
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks
WHERE id = $1 LIMIT 1;

-- name: ListTasksByProject :many
SELECT * FROM tasks
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: UpdateTaskStatus :one
UPDATE tasks
SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateTaskBeadsEpicID :one
UPDATE tasks
SET beads_epic_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1;

-- name: GetTasksByStatus :many
SELECT * FROM tasks
WHERE status = $1
ORDER BY created_at DESC;
