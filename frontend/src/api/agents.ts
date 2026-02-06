import { api } from './client'
import type { AgentRun } from '@/types/api'

export const listAgentRuns = (subtaskId: string) =>
  api.get(`subtasks/${subtaskId}/runs`).json<AgentRun[]>()

export const getAgentLogs = (runId: string) =>
  api.get(`runs/${runId}/logs`).json<{ content: string }>()
