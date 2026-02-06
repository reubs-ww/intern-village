// Event types for real-time SSE events
// Reference: specs/realtime-events.md §3.2

import type { AgentType, SubtaskStatus, TaskStatus, BlockedReason } from '@/types/api'

// Log line from agent output
export interface LogLine {
  lineNumber: number
  content: string
  timestamp: string
}

// Active agent run
export interface ActiveRun {
  id: string
  subtask_id: string
  task_id: string
  agent_type: AgentType
  status: 'RUNNING'
  log_path: string
  started_at: string
}

// Event data types
export interface AgentStartedData {
  run_id: string
  subtask_id: string
  task_id: string
  agent_type: AgentType
  attempt_number: number
  started_at: string
  log_path: string
}

export interface AgentLogData {
  run_id: string
  line: string
  line_number: number
  timestamp: string
}

export interface AgentCompletedData {
  run_id: string
  subtask_id: string
  task_id: string
  agent_type: AgentType
  completed_at: string
  token_usage: number
  pr_url?: string
}

export interface AgentFailedData {
  run_id: string
  subtask_id: string
  task_id: string
  agent_type: AgentType
  failed_at: string
  error_message: string
  will_retry: boolean
  next_attempt_at?: string
}

export interface TaskStatusChangedData {
  task_id: string
  old_status: TaskStatus | ''
  new_status: TaskStatus
}

export interface SubtaskStatusChangedData {
  subtask_id: string
  task_id: string
  old_status: SubtaskStatus
  new_status: SubtaskStatus
  blocked_reason?: BlockedReason
  pr_url?: string
  pr_number?: number
}

export interface SubtaskUnblockedData {
  subtask_id: string
  task_id: string
  unblocked_by_id: string
}

export interface ConnectedData {
  connection_id: string
  active_runs: ActiveRun[]
}

export interface HeartbeatData {
  time: string
}

// All event types
export type ProjectEvent =
  | { type: 'connected'; data: ConnectedData }
  | { type: 'heartbeat'; data: HeartbeatData }
  | { type: 'agent:started'; data: AgentStartedData }
  | { type: 'agent:log'; data: AgentLogData }
  | { type: 'agent:completed'; data: AgentCompletedData }
  | { type: 'agent:failed'; data: AgentFailedData }
  | { type: 'task:status_changed'; data: TaskStatusChangedData }
  | { type: 'subtask:status_changed'; data: SubtaskStatusChangedData }
  | { type: 'subtask:unblocked'; data: SubtaskUnblockedData }

// Parse SSE message event into typed event
export function parseSSEEvent(event: MessageEvent): ProjectEvent | null {
  try {
    const eventType = (event as any).type || 'message'
    const data = JSON.parse(event.data)

    // Handle different event types
    switch (eventType) {
      case 'connected':
        return { type: 'connected', data: data as ConnectedData }
      case 'heartbeat':
        return { type: 'heartbeat', data: data as HeartbeatData }
      case 'agent:started':
        return { type: 'agent:started', data: data as AgentStartedData }
      case 'agent:log':
        return { type: 'agent:log', data: data as AgentLogData }
      case 'agent:completed':
        return { type: 'agent:completed', data: data as AgentCompletedData }
      case 'agent:failed':
        return { type: 'agent:failed', data: data as AgentFailedData }
      case 'task:status_changed':
        return { type: 'task:status_changed', data: data as TaskStatusChangedData }
      case 'subtask:status_changed':
        return { type: 'subtask:status_changed', data: data as SubtaskStatusChangedData }
      case 'subtask:unblocked':
        return { type: 'subtask:unblocked', data: data as SubtaskUnblockedData }
      default:
        console.warn('Unknown SSE event type:', eventType)
        return null
    }
  } catch (err) {
    console.error('Failed to parse SSE event:', err)
    return null
  }
}

// Create EventSource for project events
export function createEventSource(
  projectId: string,
  logSubscriptions?: string[]
): EventSource {
  const baseUrl = `/api/projects/${projectId}/events`
  const params = new URLSearchParams()

  if (logSubscriptions && logSubscriptions.length > 0) {
    params.set('subscribe_logs', logSubscriptions.join(','))
  }

  const url = params.toString() ? `${baseUrl}?${params}` : baseUrl
  return new EventSource(url, { withCredentials: true })
}
