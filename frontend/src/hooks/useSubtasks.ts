import { useQueries, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listSubtasks,
  startSubtask,
  markMerged,
  retrySubtask,
  updatePosition,
} from '@/api/subtasks'
import { POLLING_INTERVAL } from '@/lib/constants'
import type { Subtask } from '@/types/api'

interface UseSubtasksOptions {
  /**
   * When true, polling is disabled because SSE is providing real-time updates.
   * When false or undefined, polling continues as fallback.
   */
  isSSEConnected?: boolean
}

export function useSubtasks(taskIds: string[], options: UseSubtasksOptions = {}) {
  const { isSSEConnected = false } = options

  const results = useQueries({
    queries: taskIds.map((taskId) => ({
      queryKey: ['subtasks', taskId],
      queryFn: () => listSubtasks(taskId),
      refetchInterval: (query: { state: { data?: Subtask[] } }) => {
        // When SSE is connected, disable polling entirely
        if (isSSEConnected) {
          return false
        }

        // Fallback: Poll if any subtask is in progress
        const subtasks = query.state.data
        const hasActiveWork = subtasks?.some((s) => s.status === 'IN_PROGRESS')
        return hasActiveWork ? POLLING_INTERVAL : false
      },
    })),
  })

  const allSubtasks = results.flatMap((r) => r.data ?? [])
  const isLoading = results.some((r) => r.isLoading)
  const isError = results.some((r) => r.isError)

  return {
    data: allSubtasks,
    isLoading,
    isError,
    results,
  }
}

export function useStartSubtask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => startSubtask(id),
    onSuccess: (updatedSubtask) => {
      queryClient.setQueryData<Subtask[]>(
        ['subtasks', updatedSubtask.task_id],
        (old) =>
          old
            ? old.map((s) => (s.id === updatedSubtask.id ? updatedSubtask : s))
            : [updatedSubtask]
      )
    },
  })
}

export function useMarkMerged() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => markMerged(id),
    onSuccess: () => {
      // Invalidate all subtasks as dependencies might have changed
      queryClient.invalidateQueries({ queryKey: ['subtasks'] })
    },
  })
}

export function useRetrySubtask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => retrySubtask(id),
    onSuccess: (updatedSubtask) => {
      queryClient.setQueryData<Subtask[]>(
        ['subtasks', updatedSubtask.task_id],
        (old) =>
          old
            ? old.map((s) => (s.id === updatedSubtask.id ? updatedSubtask : s))
            : [updatedSubtask]
      )
    },
  })
}

export function useUpdatePosition() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, position }: { id: string; position: number }) =>
      updatePosition(id, position),
    onSuccess: (updatedSubtask) => {
      queryClient.setQueryData<Subtask[]>(
        ['subtasks', updatedSubtask.task_id],
        (old) =>
          old
            ? old.map((s) => (s.id === updatedSubtask.id ? updatedSubtask : s))
            : [updatedSubtask]
      )
    },
  })
}
