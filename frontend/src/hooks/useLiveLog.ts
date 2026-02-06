import { useEffect, useMemo, useRef } from 'react'
import { useProjectEvents } from '@/contexts/ProjectEventsContext'
import { getAgentLogs } from '@/api/agents'
import type { LogLine } from '@/api/events'

interface UseLiveLogResult {
  lines: LogLine[]
  isStreaming: boolean
  error: string | null
}

/**
 * Parse raw log content into LogLine array
 */
function parseLogContent(content: string): LogLine[] {
  const lines = content.split('\n')
  return lines.map((line, index) => {
    // Extract timestamp from log line format: [HH:MM:SS] content
    const timestampMatch = line.match(/^\[(\d{2}:\d{2}:\d{2})\]/)
    return {
      lineNumber: index + 1,
      content: line,
      timestamp: timestampMatch ? timestampMatch[1] : '',
    }
  }).filter(line => line.content.length > 0) // Filter out empty lines
}

/**
 * Hook to subscribe to live log streaming for an agent run.
 * Fetches historical logs first, then streams new logs in real-time.
 *
 * @param runId - The agent run ID to subscribe to, or null if not streaming
 * @returns Object with log lines, streaming status, and error state
 */
export function useLiveLog(runId: string | null): UseLiveLogResult {
  const {
    subscribeToLogs,
    unsubscribeFromLogs,
    getLogsForRun,
    setInitialLogs,
    activeRuns,
    connectionError,
  } = useProjectEvents()

  // Track if we've fetched historical logs for this run
  const fetchedRunIdRef = useRef<string | null>(null)

  // Subscribe/unsubscribe when runId changes
  useEffect(() => {
    if (runId) {
      subscribeToLogs(runId)
    }

    return () => {
      if (runId) {
        unsubscribeFromLogs(runId)
      }
    }
  }, [runId, subscribeToLogs, unsubscribeFromLogs])

  // Fetch historical logs when runId changes
  useEffect(() => {
    if (!runId || fetchedRunIdRef.current === runId) {
      return
    }

    fetchedRunIdRef.current = runId

    // Fetch historical logs
    getAgentLogs(runId)
      .then(({ content }) => {
        if (content) {
          const historicalLogs = parseLogContent(content)
          setInitialLogs(runId, historicalLogs)
        }
      })
      .catch((err) => {
        // Log error but don't fail - real-time streaming may still work
        console.warn('Failed to fetch historical logs:', err)
      })
  }, [runId, setInitialLogs])

  // Reset fetched ref when runId becomes null
  useEffect(() => {
    if (!runId) {
      fetchedRunIdRef.current = null
    }
  }, [runId])

  // Get log lines for this run
  const lines = useMemo(() => {
    if (!runId) return []
    return getLogsForRun(runId)
  }, [runId, getLogsForRun])

  // Check if this run is currently active (streaming)
  const isStreaming = useMemo(() => {
    if (!runId) return false
    return activeRuns.some((run) => run.id === runId)
  }, [runId, activeRuns])

  return {
    lines,
    isStreaming,
    error: connectionError,
  }
}
