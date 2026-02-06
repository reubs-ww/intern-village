import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listTasks, createTask, retryPlanning, deleteTask } from '@/api/tasks'
import { POLLING_INTERVAL } from '@/lib/constants'
import type { Task } from '@/types/api'

interface UseTasksOptions {
  /**
   * When true, polling is disabled because SSE is providing real-time updates.
   * When false or undefined, polling continues as fallback.
   */
  isSSEConnected?: boolean
}

export function useTasks(projectId: string, options: UseTasksOptions = {}) {
  const { isSSEConnected = false } = options

  const query = useQuery({
    queryKey: ['tasks', projectId],
    queryFn: () => listTasks(projectId),
    enabled: !!projectId,
    refetchInterval: (query) => {
      // When SSE is connected, disable polling entirely
      if (isSSEConnected) {
        return false
      }

      // Fallback: Poll if any task is in PLANNING status
      const tasks = query.state.data
      const hasActivePlanning = tasks?.some((t) => t.status === 'PLANNING')
      return hasActivePlanning ? POLLING_INTERVAL : false
    },
  })

  return query
}

export function useCreateTask(projectId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ title, description }: { title: string; description: string }) =>
      createTask(projectId, title, description),
    onSuccess: (newTask) => {
      queryClient.setQueryData<Task[]>(['tasks', projectId], (old) =>
        old ? [...old, newTask] : [newTask]
      )
    },
  })
}

export function useRetryPlanning() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (taskId: string) => retryPlanning(taskId),
    onSuccess: (updatedTask) => {
      queryClient.setQueryData<Task[]>(['tasks', updatedTask.project_id], (old) =>
        old ? old.map((t) => (t.id === updatedTask.id ? updatedTask : t)) : [updatedTask]
      )
    },
  })
}

export function useDeleteTask(projectId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (taskId: string) => deleteTask(taskId),
    onSuccess: (_, deletedId) => {
      queryClient.setQueryData<Task[]>(['tasks', projectId], (old) =>
        old ? old.filter((t) => t.id !== deletedId) : []
      )
    },
  })
}
