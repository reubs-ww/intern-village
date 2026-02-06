-- Migration: 001_initial
-- Description: Initial schema for Intern Village Orchestrator
-- Reference: specs/orchestrator.md §6

-- +goose Up

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id BIGINT NOT NULL UNIQUE,
    github_username TEXT NOT NULL,
    github_token TEXT NOT NULL,  -- Encrypted with AES-256-GCM
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_github_id ON users(github_id);

-- Projects table
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    github_owner TEXT NOT NULL,
    github_repo TEXT NOT NULL,
    is_fork BOOLEAN NOT NULL DEFAULT FALSE,
    upstream_owner TEXT,              -- Original repo owner (only for forks)
    upstream_repo TEXT,               -- Original repo name (only for forks)
    default_branch TEXT NOT NULL DEFAULT 'main',
    clone_path TEXT NOT NULL,
    beads_prefix TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, github_owner, github_repo),
    CONSTRAINT check_upstream_fields CHECK (
        (upstream_owner IS NULL AND upstream_repo IS NULL) OR
        (upstream_owner IS NOT NULL AND upstream_repo IS NOT NULL)
    )
);

CREATE INDEX idx_projects_user_id ON projects(user_id);

-- Tasks table
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PLANNING',
    beads_epic_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_status ON tasks(status);

-- Subtasks table
CREATE TABLE subtasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    spec TEXT,
    implementation_plan TEXT,
    status TEXT NOT NULL DEFAULT 'PENDING',
    blocked_reason TEXT,
    branch_name TEXT,
    pr_url TEXT,
    pr_number INTEGER,
    retry_count INTEGER NOT NULL DEFAULT 0,
    token_usage INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    beads_issue_id TEXT,
    worktree_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subtasks_task_id ON subtasks(task_id);
CREATE INDEX idx_subtasks_status ON subtasks(status);
CREATE INDEX idx_subtasks_beads_issue_id ON subtasks(beads_issue_id);

-- Subtask Dependencies table
CREATE TABLE subtask_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subtask_id UUID NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    depends_on_id UUID NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(subtask_id, depends_on_id)
);

CREATE INDEX idx_subtask_deps_subtask_id ON subtask_dependencies(subtask_id);
CREATE INDEX idx_subtask_deps_depends_on_id ON subtask_dependencies(depends_on_id);

-- Agent Runs table
CREATE TABLE agent_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subtask_id UUID NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    agent_type TEXT NOT NULL,
    attempt_number INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'RUNNING',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    token_usage INTEGER,
    error_message TEXT,
    log_path TEXT NOT NULL,
    prompt_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_runs_subtask_id ON agent_runs(subtask_id);
CREATE INDEX idx_agent_runs_status ON agent_runs(status);

-- +goose Down
DROP TABLE IF EXISTS agent_runs;
DROP TABLE IF EXISTS subtask_dependencies;
DROP TABLE IF EXISTS subtasks;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
