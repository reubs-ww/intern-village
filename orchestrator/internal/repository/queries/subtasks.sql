-- Subtasks SQL queries
-- Reference: specs/orchestrator.md §4.4

-- name: CreateSubtask :one
INSERT INTO subtasks (
    task_id,
    title,
    spec,
    implementation_plan,
    status,
    position,
    beads_issue_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetSubtaskByID :one
SELECT * FROM subtasks
WHERE id = $1 LIMIT 1;

-- name: GetSubtaskByBeadsID :one
SELECT * FROM subtasks
WHERE beads_issue_id = $1 LIMIT 1;

-- name: ListSubtasksByTask :many
SELECT * FROM subtasks
WHERE task_id = $1
ORDER BY position ASC, created_at ASC;

-- name: UpdateSubtaskStatus :one
UPDATE subtasks
SET status = $2,
    blocked_reason = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSubtaskPosition :one
UPDATE subtasks
SET position = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSubtaskPR :one
UPDATE subtasks
SET pr_url = $2,
    pr_number = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSubtaskBranch :one
UPDATE subtasks
SET branch_name = $2,
    worktree_path = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSubtaskRetryCount :one
UPDATE subtasks
SET retry_count = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSubtaskTokenUsage :one
UPDATE subtasks
SET token_usage = token_usage + $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteSubtask :exec
DELETE FROM subtasks
WHERE id = $1;

-- name: GetSubtasksByStatus :many
SELECT * FROM subtasks
WHERE status = $1
ORDER BY created_at DESC;

-- name: ListInProgressSubtasks :many
SELECT * FROM subtasks
WHERE status = 'IN_PROGRESS'
ORDER BY created_at DESC;

-- name: GetNextPosition :one
SELECT COALESCE(MAX(position), 0) + 1 AS next_position
FROM subtasks
WHERE task_id = $1;
