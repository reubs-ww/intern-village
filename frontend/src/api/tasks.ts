import { api } from './client'
import type { Task } from '@/types/api'

export const listTasks = (projectId: string) =>
  api.get(`projects/${projectId}/tasks`).json<Task[]>()

export const createTask = (projectId: string, title: string, description: string) =>
  api.post(`projects/${projectId}/tasks`, { json: { title, description } }).json<Task>()

export const retryPlanning = (taskId: string) =>
  api.post(`tasks/${taskId}/retry-planning`).json<Task>()

export const deleteTask = (taskId: string) => api.delete(`tasks/${taskId}`)
