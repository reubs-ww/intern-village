-- Subtask Dependencies SQL queries
-- Reference: specs/orchestrator.md §4.5

-- name: CreateDependency :one
INSERT INTO subtask_dependencies (
    subtask_id,
    depends_on_id
) VALUES (
    $1, $2
)
ON CONFLICT (subtask_id, depends_on_id) DO NOTHING
RETURNING *;

-- name: GetDependency :one
SELECT * FROM subtask_dependencies
WHERE id = $1 LIMIT 1;

-- name: GetDependenciesForSubtask :many
SELECT sd.*, s.status as dependency_status
FROM subtask_dependencies sd
JOIN subtasks s ON sd.depends_on_id = s.id
WHERE sd.subtask_id = $1
ORDER BY sd.created_at ASC;

-- name: GetDependentsOfSubtask :many
SELECT sd.*, s.status as dependent_status
FROM subtask_dependencies sd
JOIN subtasks s ON sd.subtask_id = s.id
WHERE sd.depends_on_id = $1
ORDER BY sd.created_at ASC;

-- name: DeleteDependency :exec
DELETE FROM subtask_dependencies
WHERE subtask_id = $1 AND depends_on_id = $2;

-- name: DeleteDependenciesForSubtask :exec
DELETE FROM subtask_dependencies
WHERE subtask_id = $1;

-- name: CountUnmergedDependencies :one
SELECT COUNT(*) AS count
FROM subtask_dependencies sd
JOIN subtasks s ON sd.depends_on_id = s.id
WHERE sd.subtask_id = $1 AND s.status != 'MERGED';

-- name: HasBlockingDependencies :one
SELECT EXISTS(
    SELECT 1 FROM subtask_dependencies sd
    JOIN subtasks s ON sd.depends_on_id = s.id
    WHERE sd.subtask_id = $1 AND s.status != 'MERGED'
) AS has_blocking;
