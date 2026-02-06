-- Migration: 002_agent_runs_task_id
-- Description: Add task_id to agent_runs table for Planner agents
-- Reference: Planner agents run at task level, before subtasks exist

-- +goose Up

-- Add task_id column (nullable for backward compatibility)
ALTER TABLE agent_runs ADD COLUMN task_id UUID REFERENCES tasks(id) ON DELETE CASCADE;

-- Make subtask_id nullable (Planner runs don't have a subtask)
ALTER TABLE agent_runs ALTER COLUMN subtask_id DROP NOT NULL;

-- Add constraint: either subtask_id or task_id must be set
ALTER TABLE agent_runs ADD CONSTRAINT check_agent_run_parent CHECK (
    (subtask_id IS NOT NULL) OR (task_id IS NOT NULL)
);

-- Add index for task_id lookups
CREATE INDEX idx_agent_runs_task_id ON agent_runs(task_id);

-- +goose Down
DROP INDEX IF EXISTS idx_agent_runs_task_id;
ALTER TABLE agent_runs DROP CONSTRAINT IF EXISTS check_agent_run_parent;
ALTER TABLE agent_runs ALTER COLUMN subtask_id SET NOT NULL;
ALTER TABLE agent_runs DROP COLUMN IF EXISTS task_id;
