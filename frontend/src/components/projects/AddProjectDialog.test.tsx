import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import { AddProjectDialog } from './AddProjectDialog'
import type { CreateProjectResponse } from '@/types/api'

const mockProjectResponse: CreateProjectResponse = {
  id: 'test-id',
  github_owner: 'owner',
  github_repo: 'repo',
  is_fork: false,
  default_branch: 'main',
  created_at: new Date().toISOString(),
  was_forked: false,
}

describe('AddProjectDialog', () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(mockProjectResponse),
  }

  it('renders dialog when open', () => {
    render(<AddProjectDialog {...defaultProps} />)
    expect(screen.getByRole('heading', { name: /add project/i })).toBeInTheDocument()
  })

  it('does not render dialog content when closed', () => {
    render(<AddProjectDialog {...defaultProps} open={false} />)
    expect(screen.queryByRole('heading', { name: /add project/i })).not.toBeInTheDocument()
  })

  it('shows input for repo URL', () => {
    render(<AddProjectDialog {...defaultProps} />)
    expect(screen.getByPlaceholderText(/github\.com\/owner\/repo/i)).toBeInTheDocument()
  })

  describe('URL validation', () => {
    it('shows error for empty URL on submit', async () => {
      render(<AddProjectDialog {...defaultProps} />)
      fireEvent.click(screen.getByRole('button', { name: /add project/i }))
      await waitFor(() => {
        expect(screen.getByText(/repository url is required/i)).toBeInTheDocument()
      })
    })

    it('shows error for invalid GitHub URL', async () => {
      const user = userEvent.setup()
      render(<AddProjectDialog {...defaultProps} />)

      const input = screen.getByPlaceholderText(/github\.com\/owner\/repo/i)
      await user.type(input, 'not-a-valid-url')
      fireEvent.click(screen.getByRole('button', { name: /add project/i }))

      await waitFor(() => {
        expect(screen.getByText(/valid github repository url/i)).toBeInTheDocument()
      })
    })

    it('accepts valid GitHub URL without protocol', async () => {
      const user = userEvent.setup()
      const onSubmit = vi.fn().mockResolvedValue(mockProjectResponse)
      render(<AddProjectDialog {...defaultProps} onSubmit={onSubmit} />)

      const input = screen.getByPlaceholderText(/github\.com\/owner\/repo/i)
      await user.type(input, 'github.com/owner/repo')
      fireEvent.click(screen.getByRole('button', { name: /add project/i }))

      await waitFor(() => {
        expect(onSubmit).toHaveBeenCalledWith('github.com/owner/repo')
      })
    })

    it('accepts valid GitHub URL with https', async () => {
      const user = userEvent.setup()
      const onSubmit = vi.fn().mockResolvedValue(mockProjectResponse)
      render(<AddProjectDialog {...defaultProps} onSubmit={onSubmit} />)

      const input = screen.getByPlaceholderText(/github\.com\/owner\/repo/i)
      await user.type(input, 'https://github.com/owner/repo')
      fireEvent.click(screen.getByRole('button', { name: /add project/i }))

      await waitFor(() => {
        expect(onSubmit).toHaveBeenCalledWith('https://github.com/owner/repo')
      })
    })

    it('accepts valid GitHub URL with www', async () => {
      const user = userEvent.setup()
      const onSubmit = vi.fn().mockResolvedValue(mockProjectResponse)
      render(<AddProjectDialog {...defaultProps} onSubmit={onSubmit} />)

      const input = screen.getByPlaceholderText(/github\.com\/owner\/repo/i)
      await user.type(input, 'https://www.github.com/owner/repo')
      fireEvent.click(screen.getByRole('button', { name: /add project/i }))

      await waitFor(() => {
        expect(onSubmit).toHaveBeenCalledWith('https://www.github.com/owner/repo')
      })
    })
  })

  describe('submission behavior', () => {
    it('shows loading state during submission', async () => {
      const user = userEvent.setup()
      const onSubmit = vi.fn(() => new Promise(() => {})) // Never resolves
      render(<AddProjectDialog {...defaultProps} onSubmit={onSubmit} />)

      const input = screen.getByPlaceholderText(/github\.com\/owner\/repo/i)
      await user.type(input, 'github.com/owner/repo')
      fireEvent.click(screen.getByRole('button', { name: /add project/i }))

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /adding/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /adding/i })).toBeDisabled()
      })
    })

    it('closes dialog on successful submission', async () => {
      const user = userEvent.setup()
      const onOpenChange = vi.fn()
      const onSubmit = vi.fn().mockResolvedValue(mockProjectResponse)
      render(<AddProjectDialog {...defaultProps} onOpenChange={onOpenChange} onSubmit={onSubmit} />)

      const input = screen.getByPlaceholderText(/github\.com\/owner\/repo/i)
      await user.type(input, 'github.com/owner/repo')
      fireEvent.click(screen.getByRole('button', { name: /add project/i }))

      // Wait longer due to step animation delays (800ms + 500ms)
      await waitFor(() => {
        expect(onOpenChange).toHaveBeenCalledWith(false)
      }, { timeout: 3000 })
    })

    it('shows error message on submission failure', async () => {
      const user = userEvent.setup()
      const onSubmit = vi.fn().mockRejectedValue(new Error('Repository not found'))
      render(<AddProjectDialog {...defaultProps} onSubmit={onSubmit} />)

      const input = screen.getByPlaceholderText(/github\.com\/owner\/repo/i)
      await user.type(input, 'github.com/owner/repo')
      fireEvent.click(screen.getByRole('button', { name: /add project/i }))

      // Wait longer due to step animation delays before error shows
      await waitFor(() => {
        expect(screen.getByText(/repository not found/i)).toBeInTheDocument()
      }, { timeout: 3000 })
    })
  })

  it('calls onOpenChange when Cancel is clicked', () => {
    const onOpenChange = vi.fn()
    render(<AddProjectDialog {...defaultProps} onOpenChange={onOpenChange} />)
    fireEvent.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
