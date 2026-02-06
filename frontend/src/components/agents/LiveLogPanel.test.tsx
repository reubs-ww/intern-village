import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { LiveLogPanel } from './LiveLogPanel'
import type { LogLine } from '@/api/events'

// Mock the useLiveLog hook
const mockLines: LogLine[] = []
const mockIsStreaming = false
const mockError: string | null = null

vi.mock('@/hooks/useLiveLog', () => ({
  useLiveLog: vi.fn(() => ({
    lines: mockLines,
    isStreaming: mockIsStreaming,
    error: mockError,
  })),
}))

// Mock clipboard API
const mockWriteText = vi.fn()
Object.assign(navigator, {
  clipboard: {
    writeText: mockWriteText,
  },
})

import { useLiveLog } from '@/hooks/useLiveLog'

const mockUseLiveLog = useLiveLog as ReturnType<typeof vi.fn>

describe('LiveLogPanel', () => {
  const defaultProps = {
    runId: 'run-123',
    title: 'Test Log Panel',
    agentType: 'WORKER' as const,
    onClose: vi.fn(),
    open: true,
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockUseLiveLog.mockReturnValue({
      lines: [],
      isStreaming: false,
      error: null,
    })
  })

  it('renders the panel when open', () => {
    render(<LiveLogPanel {...defaultProps} />)

    expect(screen.getByText('Test Log Panel')).toBeInTheDocument()
    expect(screen.getByText('WORKER')).toBeInTheDocument()
  })

  it('displays log lines correctly', () => {
    const mockLogs: LogLine[] = [
      { lineNumber: 1, content: '[14:32:05] Starting task...', timestamp: '14:32:05' },
      { lineNumber: 2, content: '[14:32:06] Processing...', timestamp: '14:32:06' },
      { lineNumber: 3, content: 'Plain line without timestamp', timestamp: '' },
    ]

    mockUseLiveLog.mockReturnValue({
      lines: mockLogs,
      isStreaming: false,
      error: null,
    })

    render(<LiveLogPanel {...defaultProps} />)

    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('[14:32:05]')).toBeInTheDocument()
    expect(screen.getByText('[14:32:06]')).toBeInTheDocument()
    expect(screen.getByText('3 lines')).toBeInTheDocument()
  })

  it('shows streaming indicator when active', () => {
    mockUseLiveLog.mockReturnValue({
      lines: [{ lineNumber: 1, content: 'Log line', timestamp: '' }],
      isStreaming: true,
      error: null,
    })

    render(<LiveLogPanel {...defaultProps} />)

    expect(screen.getByText('Streaming')).toBeInTheDocument()
  })

  it('shows complete indicator when finished', () => {
    mockUseLiveLog.mockReturnValue({
      lines: [{ lineNumber: 1, content: 'Log line', timestamp: '' }],
      isStreaming: false,
      error: null,
    })

    render(<LiveLogPanel {...defaultProps} />)

    expect(screen.getByText('Complete')).toBeInTheDocument()
  })

  it('shows error indicator and message when connection fails', () => {
    mockUseLiveLog.mockReturnValue({
      lines: [],
      isStreaming: false,
      error: 'Connection lost',
    })

    render(<LiveLogPanel {...defaultProps} />)

    expect(screen.getByText('Error')).toBeInTheDocument()
    expect(screen.getByText(/Connection error: Connection lost/)).toBeInTheDocument()
  })

  it('shows waiting message when streaming with no logs yet', () => {
    mockUseLiveLog.mockReturnValue({
      lines: [],
      isStreaming: true,
      error: null,
    })

    render(<LiveLogPanel {...defaultProps} />)

    expect(screen.getByText('Waiting for logs...')).toBeInTheDocument()
  })

  it('shows no logs message when complete with no logs', () => {
    mockUseLiveLog.mockReturnValue({
      lines: [],
      isStreaming: false,
      error: null,
    })

    render(<LiveLogPanel {...defaultProps} />)

    expect(screen.getByText('No logs available')).toBeInTheDocument()
  })

  it('copies logs to clipboard', async () => {
    const mockLogs: LogLine[] = [
      { lineNumber: 1, content: 'Line 1', timestamp: '' },
      { lineNumber: 2, content: 'Line 2', timestamp: '' },
    ]

    mockUseLiveLog.mockReturnValue({
      lines: mockLogs,
      isStreaming: false,
      error: null,
    })

    render(<LiveLogPanel {...defaultProps} />)

    const copyButton = screen.getByRole('button', { name: /copy all/i })
    fireEvent.click(copyButton)

    await waitFor(() => {
      expect(mockWriteText).toHaveBeenCalledWith('Line 1\nLine 2')
    })
  })

  it('toggles auto-scroll', () => {
    mockUseLiveLog.mockReturnValue({
      lines: [{ lineNumber: 1, content: 'Log line', timestamp: '' }],
      isStreaming: true,
      error: null,
    })

    render(<LiveLogPanel {...defaultProps} />)

    // Initially auto-scroll is on
    const autoScrollButton = screen.getByRole('button', { name: /auto-scroll on/i })
    expect(autoScrollButton).toBeInTheDocument()

    // Click to toggle off
    fireEvent.click(autoScrollButton)

    expect(screen.getByRole('button', { name: /auto-scroll off/i })).toBeInTheDocument()
  })

  it('calls onClose when sheet is closed', () => {
    const onClose = vi.fn()

    render(<LiveLogPanel {...defaultProps} onClose={onClose} />)

    // Find and click the close button (X button in sheet)
    const closeButton = screen.getByRole('button', { name: /close/i })
    fireEvent.click(closeButton)

    expect(onClose).toHaveBeenCalled()
  })

  it('displays PLANNER badge for planner agents', () => {
    render(<LiveLogPanel {...defaultProps} agentType="PLANNER" />)

    expect(screen.getByText('PLANNER')).toBeInTheDocument()
  })

  it('displays singular "line" text for single log', () => {
    mockUseLiveLog.mockReturnValue({
      lines: [{ lineNumber: 1, content: 'Single line', timestamp: '' }],
      isStreaming: false,
      error: null,
    })

    render(<LiveLogPanel {...defaultProps} />)

    expect(screen.getByText('1 line')).toBeInTheDocument()
  })

  it('does not subscribe when panel is closed', () => {
    render(<LiveLogPanel {...defaultProps} open={false} />)

    // Hook should be called with null when closed
    expect(mockUseLiveLog).toHaveBeenCalledWith(null)
  })

  it('subscribes to runId when panel is open', () => {
    render(<LiveLogPanel {...defaultProps} open={true} runId="run-123" />)

    expect(mockUseLiveLog).toHaveBeenCalledWith('run-123')
  })
})
