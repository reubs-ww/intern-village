import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listProjects, createProject, deleteProject, getProject } from '@/api/projects'
import type { Project, CreateProjectResponse } from '@/types/api'

export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: listProjects,
  })
}

export function useProject(id: string) {
  return useQuery({
    queryKey: ['projects', id],
    queryFn: () => getProject(id),
    enabled: !!id,
  })
}

export function useCreateProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (repoUrl: string) => createProject(repoUrl),
    onSuccess: (response: CreateProjectResponse) => {
      // Strip was_forked when updating cache (not part of Project type)
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { was_forked, ...project } = response
      queryClient.setQueryData<Project[]>(['projects'], (old) =>
        old ? [...old, project] : [project]
      )
    },
  })
}

export function useDeleteProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteProject(id),
    onSuccess: (_, deletedId) => {
      queryClient.setQueryData<Project[]>(['projects'], (old) =>
        old ? old.filter((p) => p.id !== deletedId) : []
      )
    },
  })
}
