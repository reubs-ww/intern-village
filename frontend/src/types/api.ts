export interface User {
  id: string
  github_username: string
  created_at: string
}

export interface Project {
  id: string
  github_owner: string
  github_repo: string
  is_fork: boolean
  default_branch: string
  created_at: string
}

export interface CreateProjectResponse extends Project {
  was_forked: boolean
}

export type TaskStatus = 'PLANNING' | 'PLANNING_FAILED' | 'ACTIVE' | 'DONE'

export interface Task {
  id: string
  project_id: string
  title: string
  description: string
  status: TaskStatus
  created_at: string
}

export type SubtaskStatus =
  | 'PENDING'
  | 'READY'
  | 'BLOCKED'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'MERGED'

export type BlockedReason = 'DEPENDENCY' | 'FAILURE' | null

export interface Subtask {
  id: string
  task_id: string
  title: string
  spec: string | null
  implementation_plan: string | null
  status: SubtaskStatus
  blocked_reason: BlockedReason
  branch_name: string | null
  pr_url: string | null
  pr_number: number | null
  retry_count: number
  token_usage: number
  position: number
  created_at: string
}

export type AgentType = 'PLANNER' | 'WORKER'
export type AgentRunStatus = 'RUNNING' | 'SUCCEEDED' | 'FAILED'

export interface AgentRun {
  id: string
  subtask_id: string
  agent_type: AgentType
  attempt_number: number
  status: AgentRunStatus
  started_at: string
  ended_at: string | null
  token_usage: number | null
  error_message: string | null
}

export interface ApiError {
  error: {
    code: string
    message: string
  }
}
