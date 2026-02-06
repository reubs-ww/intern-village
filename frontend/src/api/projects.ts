import { api } from './client'
import type { Project, CreateProjectResponse } from '@/types/api'

export const listProjects = () => api.get('projects').json<Project[]>()

export const getProject = (id: string) => api.get(`projects/${id}`).json<Project>()

export const createProject = (repoUrl: string) =>
  api
    .post('projects', {
      json: { repo_url: repoUrl },
      timeout: 180000, // 3 minutes for large repo forks
    })
    .json<CreateProjectResponse>()

export const deleteProject = (id: string) => api.delete(`projects/${id}`)
