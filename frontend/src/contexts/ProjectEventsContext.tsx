import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  useRef,
  useCallback,
} from 'react'
import { useQueryClient } from '@tanstack/react-query'
import {
  createEventSource,
  type ActiveRun,
  type LogLine,
  type ProjectEvent,
} from '@/api/events'
import type { Task, Subtask } from '@/types/api'

interface ProjectEventsContextValue {
  isConnected: boolean
  connectionError: string | null
  subscribeToLogs: (runId: string) => void
  unsubscribeFromLogs: (runId: string) => void
  getLogsForRun: (runId: string) => LogLine[]
  setInitialLogs: (runId: string, logs: LogLine[]) => void
  activeRuns: ActiveRun[]
}

const ProjectEventsContext = createContext<ProjectEventsContextValue | null>(null)

// Reconnection backoff delays (in ms)
const BACKOFF_DELAYS = [1000, 2000, 4000, 8000, 16000, 30000]

interface ProjectEventsProviderProps {
  projectId: string
  children: React.ReactNode
}

export function ProjectEventsProvider({
  projectId,
  children,
}: ProjectEventsProviderProps) {
  const queryClient = useQueryClient()
  const [isConnected, setIsConnected] = useState(false)
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const [activeRuns, setActiveRuns] = useState<ActiveRun[]>([])
  const [logSubscriptions, setLogSubscriptions] = useState<Set<string>>(new Set())
  const [logBuffers, setLogBuffers] = useState<Map<string, LogLine[]>>(new Map())

  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimeoutRef = useRef<number | null>(null)

  // Subscribe to logs for a specific run
  const subscribeToLogs = useCallback((runId: string) => {
    setLogSubscriptions((prev) => {
      const newSet = new Set(prev)
      newSet.add(runId)
      return newSet
    })
    // Initialize empty buffer if not exists
    setLogBuffers((prev) => {
      if (!prev.has(runId)) {
        const newMap = new Map(prev)
        newMap.set(runId, [])
        return newMap
      }
      return prev
    })
  }, [])

  // Unsubscribe from logs for a specific run
  const unsubscribeFromLogs = useCallback((runId: string) => {
    setLogSubscriptions((prev) => {
      const newSet = new Set(prev)
      newSet.delete(runId)
      return newSet
    })
  }, [])

  // Get logs for a specific run
  const getLogsForRun = useCallback(
    (runId: string): LogLine[] => {
      return logBuffers.get(runId) || []
    },
    [logBuffers]
  )

  // Set initial logs for a run (used when fetching historical logs)
  const setInitialLogs = useCallback((runId: string, logs: LogLine[]) => {
    setLogBuffers((prev) => {
      const existing = prev.get(runId) || []
      // Only set if we have more logs than what's already buffered
      // This prevents overwriting real-time logs with stale historical data
      if (logs.length > existing.length) {
        const newMap = new Map(prev)
        newMap.set(runId, logs)
        return newMap
      }
      return prev
    })
  }, [])

  // Handle incoming events
  const handleEvent = useCallback(
    (event: ProjectEvent) => {
      switch (event.type) {
        case 'connected':
          setActiveRuns(event.data.active_runs || [])
          reconnectAttemptRef.current = 0
          break

        case 'heartbeat':
          // Keep connection alive, nothing to do
          break

        case 'agent:started':
          setActiveRuns((prev) => [
            ...prev,
            {
              id: event.data.run_id,
              subtask_id: event.data.subtask_id,
              task_id: event.data.task_id,
              agent_type: event.data.agent_type,
              status: 'RUNNING',
              log_path: event.data.log_path,
              started_at: event.data.started_at,
            },
          ])
          break

        case 'agent:log':
          // Append log line to buffer
          setLogBuffers((prev) => {
            const runId = event.data.run_id
            const existing = prev.get(runId) || []
            const newLine: LogLine = {
              lineNumber: event.data.line_number,
              content: event.data.line,
              timestamp: event.data.timestamp,
            }
            const newMap = new Map(prev)
            newMap.set(runId, [...existing, newLine])
            return newMap
          })
          break

        case 'agent:completed':
          // Remove from active runs
          setActiveRuns((prev) => prev.filter((r) => r.id !== event.data.run_id))
          // Invalidate agent runs query
          queryClient.invalidateQueries({
            queryKey: ['agentRuns', event.data.subtask_id],
          })
          break

        case 'agent:failed':
          // Remove from active runs
          setActiveRuns((prev) => prev.filter((r) => r.id !== event.data.run_id))
          // Invalidate agent runs query
          queryClient.invalidateQueries({
            queryKey: ['agentRuns', event.data.subtask_id],
          })
          break

        case 'task:status_changed':
          // Update task in cache
          queryClient.setQueriesData<Task[]>(
            { queryKey: ['tasks', projectId] },
            (old) =>
              old?.map((t) =>
                t.id === event.data.task_id
                  ? { ...t, status: event.data.new_status }
                  : t
              )
          )
          break

        case 'subtask:status_changed':
          // Update subtask in cache
          queryClient.setQueriesData<Subtask[]>(
            { queryKey: ['subtasks', event.data.task_id] },
            (old) =>
              old?.map((s) =>
                s.id === event.data.subtask_id
                  ? {
                      ...s,
                      status: event.data.new_status,
                      blocked_reason: event.data.blocked_reason ?? null,
                      pr_url: event.data.pr_url ?? s.pr_url,
                      pr_number: event.data.pr_number ?? s.pr_number,
                    }
                  : s
              )
          )
          break

        case 'subtask:unblocked':
          // Update subtask in cache
          queryClient.setQueriesData<Subtask[]>(
            { queryKey: ['subtasks', event.data.task_id] },
            (old) =>
              old?.map((s) =>
                s.id === event.data.subtask_id
                  ? { ...s, status: 'READY' as const, blocked_reason: null }
                  : s
              )
          )
          break
      }
    },
    [projectId, queryClient]
  )

  // Connect to SSE
  const connect = useCallback(() => {
    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    const logSubsArray = Array.from(logSubscriptions)
    const eventSource = createEventSource(projectId, logSubsArray)
    eventSourceRef.current = eventSource

    // Register event handlers
    const eventTypes = [
      'connected',
      'heartbeat',
      'agent:started',
      'agent:log',
      'agent:completed',
      'agent:failed',
      'task:status_changed',
      'subtask:status_changed',
      'subtask:unblocked',
    ]

    eventTypes.forEach((type) => {
      eventSource.addEventListener(type, (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data)
          handleEvent({ type, data } as ProjectEvent)
        } catch (err) {
          console.error('Failed to parse event:', err)
        }
      })
    })

    eventSource.onopen = () => {
      setIsConnected(true)
      setConnectionError(null)
    }

    eventSource.onerror = () => {
      setIsConnected(false)
      eventSource.close()

      // Schedule reconnection with backoff
      const attempt = reconnectAttemptRef.current
      const delay = BACKOFF_DELAYS[Math.min(attempt, BACKOFF_DELAYS.length - 1)]
      setConnectionError(`Connection lost. Reconnecting in ${delay / 1000}s...`)

      reconnectTimeoutRef.current = window.setTimeout(() => {
        reconnectAttemptRef.current++
        connect()
      }, delay)
    }
  }, [projectId, logSubscriptions, handleEvent])

  // Effect to manage connection lifecycle
  useEffect(() => {
    if (!projectId) return

    connect()

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
    }
  }, [projectId, connect])

  // Reconnect when log subscriptions change
  useEffect(() => {
    if (isConnected && eventSourceRef.current) {
      // Reconnect with new subscriptions
      connect()
    }
  }, [logSubscriptions, connect, isConnected])

  const value: ProjectEventsContextValue = {
    isConnected,
    connectionError,
    subscribeToLogs,
    unsubscribeFromLogs,
    getLogsForRun,
    setInitialLogs,
    activeRuns,
  }

  return (
    <ProjectEventsContext.Provider value={value}>
      {children}
    </ProjectEventsContext.Provider>
  )
}

export function useProjectEvents(): ProjectEventsContextValue {
  const context = useContext(ProjectEventsContext)
  if (!context) {
    throw new Error('useProjectEvents must be used within a ProjectEventsProvider')
  }
  return context
}
