import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useLiveLog } from './useLiveLog'

// Mock the agents API
vi.mock('@/api/agents', () => ({
  getAgentLogs: vi.fn(() => Promise.resolve({ content: '' })),
}))

// Mock the ProjectEventsContext
const mockSubscribeToLogs = vi.fn()
const mockUnsubscribeFromLogs = vi.fn()
const mockGetLogsForRun = vi.fn()
const mockSetInitialLogs = vi.fn()
const mockActiveRuns = [{ id: 'run-1', subtask_id: 'subtask-1', task_id: 'task-1', agent_type: 'WORKER' as const, status: 'RUNNING' as const, log_path: '/logs/run-1.log', started_at: '2026-01-01T00:00:00Z' }]

vi.mock('@/contexts/ProjectEventsContext', () => ({
  useProjectEvents: () => ({
    subscribeToLogs: mockSubscribeToLogs,
    unsubscribeFromLogs: mockUnsubscribeFromLogs,
    getLogsForRun: mockGetLogsForRun,
    setInitialLogs: mockSetInitialLogs,
    activeRuns: mockActiveRuns,
    connectionError: null,
  }),
}))

describe('useLiveLog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetLogsForRun.mockReturnValue([])
  })

  it('returns empty lines when runId is null', () => {
    const { result } = renderHook(() => useLiveLog(null))

    expect(result.current.lines).toEqual([])
    expect(result.current.isStreaming).toBe(false)
    expect(result.current.error).toBe(null)
  })

  it('subscribes to logs when runId is provided', () => {
    renderHook(() => useLiveLog('run-1'))

    expect(mockSubscribeToLogs).toHaveBeenCalledWith('run-1')
  })

  it('unsubscribes from logs on unmount', () => {
    const { unmount } = renderHook(() => useLiveLog('run-1'))

    unmount()

    expect(mockUnsubscribeFromLogs).toHaveBeenCalledWith('run-1')
  })

  it('returns logs for the run', () => {
    const mockLogs = [
      { lineNumber: 1, content: 'Line 1', timestamp: '2026-01-01T00:00:00Z' },
      { lineNumber: 2, content: 'Line 2', timestamp: '2026-01-01T00:00:01Z' },
    ]
    mockGetLogsForRun.mockReturnValue(mockLogs)

    const { result } = renderHook(() => useLiveLog('run-1'))

    expect(result.current.lines).toEqual(mockLogs)
  })

  it('indicates streaming when run is active', () => {
    const { result } = renderHook(() => useLiveLog('run-1'))

    expect(result.current.isStreaming).toBe(true)
  })

  it('indicates not streaming when run is not active', () => {
    const { result } = renderHook(() => useLiveLog('run-2'))

    expect(result.current.isStreaming).toBe(false)
  })

  it('resubscribes when runId changes', () => {
    const { rerender } = renderHook(
      ({ runId }) => useLiveLog(runId),
      { initialProps: { runId: 'run-1' as string | null } }
    )

    expect(mockSubscribeToLogs).toHaveBeenCalledWith('run-1')

    rerender({ runId: 'run-2' })

    expect(mockUnsubscribeFromLogs).toHaveBeenCalledWith('run-1')
    expect(mockSubscribeToLogs).toHaveBeenCalledWith('run-2')
  })
})
