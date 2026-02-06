import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { TaskFilter } from './TaskFilter'
import type { Task } from '@/types/api'

const mockTasks: Task[] = [
  {
    id: 'task-1',
    project_id: 'project-1',
    title: 'First Task',
    description: 'Description 1',
    status: 'ACTIVE',
    created_at: '2026-02-05T00:00:00Z',
  },
  {
    id: 'task-2',
    project_id: 'project-1',
    title: 'Second Task',
    description: 'Description 2',
    status: 'PLANNING',
    created_at: '2026-02-05T00:00:00Z',
  },
]

describe('TaskFilter', () => {
  it('renders filter trigger button', () => {
    render(<TaskFilter tasks={mockTasks} selectedTaskId={null} onSelect={vi.fn()} />)
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('shows "All Tasks" when no task is selected', () => {
    render(<TaskFilter tasks={mockTasks} selectedTaskId={null} onSelect={vi.fn()} />)
    expect(screen.getByText('All Tasks')).toBeInTheDocument()
  })

  it('shows selected task title when a task is selected', () => {
    render(<TaskFilter tasks={mockTasks} selectedTaskId="task-1" onSelect={vi.fn()} />)
    expect(screen.getByText('First Task')).toBeInTheDocument()
  })

  it('renders empty list gracefully', () => {
    render(<TaskFilter tasks={[]} selectedTaskId={null} onSelect={vi.fn()} />)
    expect(screen.getByText('All Tasks')).toBeInTheDocument()
  })

  // Note: Tests for dropdown interaction are skipped due to Radix UI Select
  // component limitations in jsdom. These should be covered by E2E tests.
})
