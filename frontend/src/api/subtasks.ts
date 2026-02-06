import { api } from './client'
import type { Subtask } from '@/types/api'

export const listSubtasks = (taskId: string) =>
  api.get(`tasks/${taskId}/subtasks`).json<Subtask[]>()

export const startSubtask = (id: string) =>
  api.post(`subtasks/${id}/start`).json<Subtask>()

export const markMerged = (id: string) =>
  api.post(`subtasks/${id}/mark-merged`).json<Subtask>()

export const retrySubtask = (id: string) =>
  api.post(`subtasks/${id}/retry`).json<Subtask>()

export const updatePosition = (id: string, position: number) =>
  api.patch(`subtasks/${id}/position`, { json: { position } }).json<Subtask>()
