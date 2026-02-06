import { describe, it, expect, vi, beforeEach, afterEach, Mock } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ProjectEventsProvider, useProjectEvents } from './ProjectEventsContext'

// Mock EventSource
class MockEventSource {
  static instances: MockEventSource[] = []
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  listeners: Map<string, ((e: MessageEvent) => void)[]> = new Map()
  url: string
  readyState = 0

  constructor(url: string) {
    this.url = url
    MockEventSource.instances.push(this)
    // Simulate connection after a tick
    setTimeout(() => {
      this.readyState = 1
      this.onopen?.()
    }, 0)
  }

  addEventListener(type: string, handler: (e: MessageEvent) => void) {
    const handlers = this.listeners.get(type) || []
    handlers.push(handler)
    this.listeners.set(type, handlers)
  }

  removeEventListener(type: string, handler: (e: MessageEvent) => void) {
    const handlers = this.listeners.get(type) || []
    this.listeners.set(type, handlers.filter(h => h !== handler))
  }

  close() {
    this.readyState = 2
  }

  emit(type: string, data: unknown) {
    const handlers = this.listeners.get(type) || []
    const event = new MessageEvent(type, { data: JSON.stringify(data) })
    handlers.forEach(h => h(event))
  }
}

// Mock the events module to use our MockEventSource
vi.mock('@/api/events', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/events')>()
  return {
    ...actual,
    createEventSource: (projectId: string, logSubscriptions?: string[]) => {
      const baseUrl = `/api/projects/${projectId}/events`
      const params = new URLSearchParams()
      if (logSubscriptions && logSubscriptions.length > 0) {
        params.set('subscribe_logs', logSubscriptions.join(','))
      }
      const url = params.toString() ? `${baseUrl}?${params}` : baseUrl
      return new MockEventSource(url)
    },
  }
})

beforeEach(() => {
  MockEventSource.instances = []
})

function createWrapper(projectId: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } }
  })

  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ProjectEventsProvider projectId={projectId}>
          {children}
        </ProjectEventsProvider>
      </QueryClientProvider>
    )
  }
}

describe('ProjectEventsContext', () => {
  it('throws error when used outside provider', () => {
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    expect(() => {
      renderHook(() => useProjectEvents())
    }).toThrow('useProjectEvents must be used within a ProjectEventsProvider')

    consoleSpy.mockRestore()
  })

  it('provides initial disconnected state', () => {
    const { result } = renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    expect(result.current.isConnected).toBe(false)
    expect(result.current.connectionError).toBe(null)
    expect(result.current.activeRuns).toEqual([])
  })

  it('creates EventSource on mount', async () => {
    renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    await waitFor(() => {
      expect(MockEventSource.instances.length).toBe(1)
      expect(MockEventSource.instances[0].url).toContain('/api/projects/project-1/events')
    })
  })

  it('becomes connected after EventSource opens', async () => {
    const { result } = renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })
  })

  it('handles connected event with active runs', async () => {
    const { result } = renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    await waitFor(() => {
      expect(MockEventSource.instances.length).toBe(1)
    })

    const eventSource = MockEventSource.instances[0]

    act(() => {
      eventSource.emit('connected', {
        connection_id: 'conn-1',
        active_runs: [
          {
            id: 'run-1',
            subtask_id: 'subtask-1',
            task_id: 'task-1',
            agent_type: 'WORKER',
            status: 'RUNNING',
            log_path: '/logs/run-1.log',
            started_at: '2026-01-01T00:00:00Z',
          },
        ],
      })
    })

    await waitFor(() => {
      expect(result.current.activeRuns.length).toBe(1)
      expect(result.current.activeRuns[0].id).toBe('run-1')
    })
  })

  it('handles agent:started event', async () => {
    const { result } = renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    await waitFor(() => {
      expect(MockEventSource.instances.length).toBe(1)
    })

    const eventSource = MockEventSource.instances[0]

    act(() => {
      eventSource.emit('agent:started', {
        run_id: 'run-2',
        subtask_id: 'subtask-2',
        task_id: 'task-1',
        agent_type: 'PLANNER',
        attempt_number: 1,
        started_at: '2026-01-01T00:00:00Z',
        log_path: '/logs/run-2.log',
      })
    })

    await waitFor(() => {
      expect(result.current.activeRuns.length).toBe(1)
      expect(result.current.activeRuns[0].id).toBe('run-2')
    })
  })

  it('handles agent:completed event', async () => {
    const { result } = renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    await waitFor(() => {
      expect(MockEventSource.instances.length).toBe(1)
    })

    const eventSource = MockEventSource.instances[0]

    // First add a run
    act(() => {
      eventSource.emit('agent:started', {
        run_id: 'run-3',
        subtask_id: 'subtask-3',
        task_id: 'task-1',
        agent_type: 'WORKER',
        attempt_number: 1,
        started_at: '2026-01-01T00:00:00Z',
        log_path: '/logs/run-3.log',
      })
    })

    await waitFor(() => {
      expect(result.current.activeRuns.length).toBe(1)
    })

    // Then complete it
    act(() => {
      eventSource.emit('agent:completed', {
        run_id: 'run-3',
        subtask_id: 'subtask-3',
        task_id: 'task-1',
        agent_type: 'WORKER',
        completed_at: '2026-01-01T00:01:00Z',
        token_usage: 1000,
      })
    })

    await waitFor(() => {
      expect(result.current.activeRuns.length).toBe(0)
    })
  })

  it('subscribes and unsubscribes to logs', async () => {
    const { result } = renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    act(() => {
      result.current.subscribeToLogs('run-1')
    })

    // Verify subscription was added (we can't easily verify the reconnect,
    // but we can verify the internal state via getLogsForRun)
    expect(result.current.getLogsForRun('run-1')).toEqual([])

    act(() => {
      result.current.unsubscribeFromLogs('run-1')
    })
  })

  it('buffers log lines for subscribed runs', async () => {
    const { result } = renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    await waitFor(() => {
      expect(MockEventSource.instances.length).toBe(1)
    })

    const eventSource = MockEventSource.instances[0]

    // Subscribe to logs
    act(() => {
      result.current.subscribeToLogs('run-1')
    })

    // Emit log events
    act(() => {
      eventSource.emit('agent:log', {
        run_id: 'run-1',
        line: 'Log line 1',
        line_number: 1,
        timestamp: '2026-01-01T00:00:01Z',
      })
    })

    act(() => {
      eventSource.emit('agent:log', {
        run_id: 'run-1',
        line: 'Log line 2',
        line_number: 2,
        timestamp: '2026-01-01T00:00:02Z',
      })
    })

    await waitFor(() => {
      const logs = result.current.getLogsForRun('run-1')
      expect(logs.length).toBe(2)
      expect(logs[0].content).toBe('Log line 1')
      expect(logs[1].content).toBe('Log line 2')
    })
  })

  it('closes EventSource on unmount', async () => {
    const { unmount } = renderHook(
      () => useProjectEvents(),
      { wrapper: createWrapper('project-1') }
    )

    await waitFor(() => {
      expect(MockEventSource.instances.length).toBe(1)
    })

    const eventSource = MockEventSource.instances[0]

    unmount()

    expect(eventSource.readyState).toBe(2) // CLOSED
  })
})
