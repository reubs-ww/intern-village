-- Agent Runs SQL queries
-- Reference: specs/orchestrator.md §4.6

-- name: CreateAgentRun :one
-- For Worker agents (subtask-level runs)
INSERT INTO agent_runs (
    subtask_id,
    agent_type,
    attempt_number,
    status,
    log_path,
    prompt_text
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: CreateAgentRunForTask :one
-- For Planner agents (task-level runs)
INSERT INTO agent_runs (
    task_id,
    agent_type,
    attempt_number,
    status,
    log_path,
    prompt_text
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetAgentRunByID :one
SELECT * FROM agent_runs
WHERE id = $1 LIMIT 1;

-- name: ListAgentRunsBySubtask :many
SELECT * FROM agent_runs
WHERE subtask_id = $1
ORDER BY attempt_number DESC;

-- name: UpdateAgentRunStatus :one
UPDATE agent_runs
SET status = $2,
    ended_at = $3,
    error_message = $4
WHERE id = $1
RETURNING *;

-- name: UpdateAgentRunTokenUsage :one
UPDATE agent_runs
SET token_usage = $2
WHERE id = $1
RETURNING *;

-- name: GetRunningAgentRuns :many
SELECT * FROM agent_runs
WHERE status = 'RUNNING'
ORDER BY started_at ASC;

-- name: GetLatestAgentRun :one
SELECT * FROM agent_runs
WHERE subtask_id = $1
ORDER BY attempt_number DESC
LIMIT 1;

-- name: CountAgentRunsForSubtask :one
SELECT COUNT(*) AS count
FROM agent_runs
WHERE subtask_id = $1;

-- name: ListAgentRunsByTask :many
-- For Planner agent runs (task-level)
SELECT * FROM agent_runs
WHERE task_id = $1
ORDER BY attempt_number DESC;

-- name: GetLatestAgentRunForTask :one
-- Get most recent Planner run for a task
SELECT * FROM agent_runs
WHERE task_id = $1
ORDER BY attempt_number DESC
LIMIT 1;

-- name: CountAgentRunsForTask :one
SELECT COUNT(*) AS count
FROM agent_runs
WHERE task_id = $1;

-- name: MarkStaleAgentRunsFailed :exec
UPDATE agent_runs
SET status = 'FAILED',
    ended_at = NOW(),
    error_message = 'Orchestrator restart - process orphaned'
WHERE status = 'RUNNING'
AND started_at < $1;

-- name: ListActiveAgentRunsByProject :many
-- Returns both Planner runs (task-level) and Worker runs (subtask-level)
SELECT
    ar.id,
    ar.subtask_id,
    COALESCE(ar.task_id, s.task_id) AS task_id,
    ar.agent_type,
    ar.status,
    ar.log_path,
    ar.started_at
FROM agent_runs ar
LEFT JOIN subtasks s ON ar.subtask_id = s.id
LEFT JOIN tasks t ON ar.task_id = t.id OR s.task_id = t.id
WHERE (t.project_id = $1 OR (
    ar.task_id IS NOT NULL AND ar.task_id IN (SELECT id FROM tasks WHERE project_id = $1)
))
AND ar.status = 'RUNNING'
ORDER BY ar.started_at ASC;
