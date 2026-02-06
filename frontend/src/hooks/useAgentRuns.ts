import { useQuery } from '@tanstack/react-query'
import { listAgentRuns, getAgentLogs } from '@/api/agents'
import { POLLING_INTERVAL } from '@/lib/constants'

interface UseAgentRunsOptions {
  /**
   * When true, polling is disabled because SSE is providing real-time updates.
   * When false or undefined, polling continues as fallback.
   */
  isSSEConnected?: boolean
}

export function useAgentRuns(subtaskId: string | null, options: UseAgentRunsOptions = {}) {
  const { isSSEConnected = false } = options

  return useQuery({
    queryKey: ['agent-runs', subtaskId],
    queryFn: () => listAgentRuns(subtaskId!),
    enabled: !!subtaskId,
    refetchInterval: (query) => {
      // When SSE is connected, disable polling entirely
      if (isSSEConnected) {
        return false
      }

      // Fallback: Poll if any run is still running
      const runs = query.state.data
      const hasRunning = runs?.some((r) => r.status === 'RUNNING')
      return hasRunning ? POLLING_INTERVAL : false
    },
  })
}

export function useAgentLogs(runId: string | null) {
  return useQuery({
    queryKey: ['agent-logs', runId],
    queryFn: () => getAgentLogs(runId!),
    enabled: !!runId,
  })
}
