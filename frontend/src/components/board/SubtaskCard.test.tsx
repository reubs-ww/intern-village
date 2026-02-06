import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test/test-utils'
import { SubtaskCard } from './SubtaskCard'
import type { Subtask } from '@/types/api'

// Mock @dnd-kit/sortable
vi.mock('@dnd-kit/sortable', () => ({
  useSortable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    transition: null,
    isDragging: false,
  }),
}))

vi.mock('@dnd-kit/utilities', () => ({
  CSS: {
    Transform: {
      toString: () => undefined,
    },
  },
}))

const baseSubtask: Subtask = {
  id: '1',
  task_id: 'task-1',
  title: 'Test Subtask',
  spec: null,
  implementation_plan: null,
  status: 'READY',
  blocked_reason: null,
  branch_name: null,
  pr_url: null,
  pr_number: null,
  retry_count: 0,
  token_usage: 0,
  position: 1,
  created_at: '2026-02-05T00:00:00Z',
}

describe('SubtaskCard', () => {
  it('renders subtask title', () => {
    render(<SubtaskCard subtask={baseSubtask} />)
    expect(screen.getByText('Test Subtask')).toBeInTheDocument()
  })

  it('shows task title badge when provided', () => {
    render(<SubtaskCard subtask={baseSubtask} taskTitle="Parent Task" />)
    expect(screen.getByText('Parent Task')).toBeInTheDocument()
  })

  describe('READY status', () => {
    it('shows Start button', () => {
      const onStart = vi.fn()
      render(<SubtaskCard subtask={baseSubtask} onStart={onStart} />)
      expect(screen.getByRole('button', { name: /start/i })).toBeInTheDocument()
    })

    it('calls onStart when Start button clicked', () => {
      const onStart = vi.fn()
      render(<SubtaskCard subtask={baseSubtask} onStart={onStart} />)
      fireEvent.click(screen.getByRole('button', { name: /start/i }))
      expect(onStart).toHaveBeenCalledTimes(1)
    })

    it('disables Start button when isStarting is true', () => {
      render(<SubtaskCard subtask={baseSubtask} onStart={vi.fn()} isStarting />)
      expect(screen.getByRole('button', { name: /start/i })).toBeDisabled()
    })
  })

  describe('IN_PROGRESS status', () => {
    const inProgressSubtask: Subtask = { ...baseSubtask, status: 'IN_PROGRESS' }

    it('does not show Start button', () => {
      render(<SubtaskCard subtask={inProgressSubtask} />)
      expect(screen.queryByRole('button', { name: /start/i })).not.toBeInTheDocument()
    })
  })

  describe('COMPLETED status', () => {
    const completedSubtask: Subtask = {
      ...baseSubtask,
      status: 'COMPLETED',
      pr_url: 'https://github.com/owner/repo/pull/123',
      pr_number: 123,
    }

    it('shows View PR link', () => {
      render(<SubtaskCard subtask={completedSubtask} />)
      const prLink = screen.getByRole('link', { name: /view pr/i })
      expect(prLink).toBeInTheDocument()
      expect(prLink).toHaveAttribute('href', 'https://github.com/owner/repo/pull/123')
      expect(prLink).toHaveAttribute('target', '_blank')
      expect(prLink).toHaveAttribute('rel', 'noopener noreferrer')
    })

    it('shows Mark Merged button', () => {
      const onMarkMerged = vi.fn()
      render(<SubtaskCard subtask={completedSubtask} onMarkMerged={onMarkMerged} />)
      expect(screen.getByRole('button', { name: /mark merged/i })).toBeInTheDocument()
    })

    it('calls onMarkMerged when button clicked', () => {
      const onMarkMerged = vi.fn()
      render(<SubtaskCard subtask={completedSubtask} onMarkMerged={onMarkMerged} />)
      fireEvent.click(screen.getByRole('button', { name: /mark merged/i }))
      expect(onMarkMerged).toHaveBeenCalledTimes(1)
    })
  })

  describe('MERGED status', () => {
    const mergedSubtask: Subtask = {
      ...baseSubtask,
      status: 'MERGED',
      pr_url: 'https://github.com/owner/repo/pull/123',
      pr_number: 123,
    }

    it('shows View PR link', () => {
      render(<SubtaskCard subtask={mergedSubtask} />)
      expect(screen.getByRole('link', { name: /view pr/i })).toBeInTheDocument()
    })

    it('does not show Mark Merged button', () => {
      render(<SubtaskCard subtask={mergedSubtask} onMarkMerged={vi.fn()} />)
      expect(screen.queryByRole('button', { name: /mark merged/i })).not.toBeInTheDocument()
    })
  })

  describe('BLOCKED status with DEPENDENCY reason', () => {
    const blockedDependencySubtask: Subtask = {
      ...baseSubtask,
      status: 'BLOCKED',
      blocked_reason: 'DEPENDENCY',
    }

    it('shows waiting on dependency badge', () => {
      render(<SubtaskCard subtask={blockedDependencySubtask} />)
      expect(screen.getByText(/waiting on dependency/i)).toBeInTheDocument()
    })

    it('does not show Retry button', () => {
      render(<SubtaskCard subtask={blockedDependencySubtask} onRetry={vi.fn()} />)
      expect(screen.queryByRole('button', { name: /retry/i })).not.toBeInTheDocument()
    })
  })

  describe('BLOCKED status with FAILURE reason', () => {
    const blockedFailureSubtask: Subtask = {
      ...baseSubtask,
      status: 'BLOCKED',
      blocked_reason: 'FAILURE',
    }

    it('shows Failed badge', () => {
      render(<SubtaskCard subtask={blockedFailureSubtask} />)
      expect(screen.getByText(/failed/i)).toBeInTheDocument()
    })

    it('shows Retry button', () => {
      const onRetry = vi.fn()
      render(<SubtaskCard subtask={blockedFailureSubtask} onRetry={onRetry} />)
      expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
    })

    it('calls onRetry when Retry button clicked', () => {
      const onRetry = vi.fn()
      render(<SubtaskCard subtask={blockedFailureSubtask} onRetry={onRetry} />)
      fireEvent.click(screen.getByRole('button', { name: /retry/i }))
      expect(onRetry).toHaveBeenCalledTimes(1)
    })
  })

  it('calls onClick when card is clicked', () => {
    const onClick = vi.fn()
    render(<SubtaskCard subtask={baseSubtask} onClick={onClick} />)
    fireEvent.click(screen.getByText('Test Subtask'))
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})
