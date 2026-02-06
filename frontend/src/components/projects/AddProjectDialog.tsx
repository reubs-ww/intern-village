import { useState, useEffect } from 'react'
import { Check, GitFork, Download, Settings, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import type { CreateProjectResponse } from '@/types/api'

interface AddProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (repoUrl: string) => Promise<CreateProjectResponse>
}

const GITHUB_URL_PATTERN = /^(https?:\/\/)?(www\.)?github\.com\/[\w.-]+\/[\w.-]+\/?$/

type StepStatus = 'pending' | 'active' | 'completed' | 'skipped'

interface Step {
  id: string
  label: string
  icon: React.ComponentType<{ className?: string }>
}

const STEPS: Step[] = [
  { id: 'checking', label: 'Checking permissions', icon: Settings },
  { id: 'forking', label: 'Forking repository', icon: GitFork },
  { id: 'cloning', label: 'Cloning repository', icon: Download },
  { id: 'initializing', label: 'Initializing project', icon: Settings },
]

export function AddProjectDialog({
  open,
  onOpenChange,
  onSubmit,
}: AddProjectDialogProps) {
  const [repoUrl, setRepoUrl] = useState('')
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [stepStatuses, setStepStatuses] = useState<Record<string, StepStatus>>({
    checking: 'pending',
    forking: 'pending',
    cloning: 'pending',
    initializing: 'pending',
  })

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setRepoUrl('')
      setError('')
      setIsLoading(false)
      setStepStatuses({
        checking: 'pending',
        forking: 'pending',
        cloning: 'pending',
        initializing: 'pending',
      })
    }
  }, [open])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!repoUrl.trim()) {
      setError('Repository URL is required')
      return
    }

    if (!GITHUB_URL_PATTERN.test(repoUrl.trim())) {
      setError('Please enter a valid GitHub repository URL (e.g., github.com/owner/repo)')
      return
    }

    setIsLoading(true)

    // Simulate step progression while the actual API call runs
    // Step 1: Checking permissions
    setStepStatuses(prev => ({ ...prev, checking: 'active' }))

    try {
      // Start the actual API call
      const resultPromise = onSubmit(repoUrl.trim())

      // After a brief delay, move to checking completed
      await new Promise(resolve => setTimeout(resolve, 800))
      setStepStatuses(prev => ({ ...prev, checking: 'completed', forking: 'active' }))

      // Wait for result to know if we're forking
      const result = await resultPromise

      if (result.was_forked) {
        // Was forked - mark forking as completed, then cloning
        setStepStatuses(prev => ({
          ...prev,
          forking: 'completed',
          cloning: 'completed',
          initializing: 'completed',
        }))
      } else {
        // Direct clone - skip forking step
        setStepStatuses(prev => ({
          ...prev,
          forking: 'skipped',
          cloning: 'completed',
          initializing: 'completed',
        }))
      }

      // Brief delay to show completion before closing
      await new Promise(resolve => setTimeout(resolve, 500))
      onOpenChange(false)
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Failed to add project. Please try again.')
      }
      // Reset steps on error
      setStepStatuses({
        checking: 'pending',
        forking: 'pending',
        cloning: 'pending',
        initializing: 'pending',
      })
    } finally {
      setIsLoading(false)
    }
  }

  const renderStep = (step: Step, status: StepStatus) => {
    const Icon = step.icon

    if (status === 'skipped') {
      return null
    }

    return (
      <div
        key={step.id}
        className={`flex items-center gap-3 py-2 ${
          status === 'pending' ? 'text-muted-foreground' : 'text-foreground'
        }`}
      >
        <div className="relative flex h-6 w-6 items-center justify-center">
          {status === 'completed' ? (
            <Check className="h-5 w-5 text-green-500" />
          ) : status === 'active' ? (
            <Loader2 className="h-5 w-5 animate-spin text-primary" />
          ) : (
            <Icon className="h-5 w-5" />
          )}
        </div>
        <span className={status === 'active' ? 'font-medium' : ''}>
          {step.label}
          {status === 'active' && '...'}
        </span>
      </div>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Project</DialogTitle>
            <DialogDescription>
              Enter a GitHub repository URL to add it as a project.
            </DialogDescription>
          </DialogHeader>

          <div className="py-4">
            <Input
              placeholder="github.com/owner/repo"
              value={repoUrl}
              onChange={(e) => {
                setRepoUrl(e.target.value)
                setError('')
              }}
              disabled={isLoading}
              autoFocus
            />
            {error && (
              <p className="mt-2 text-sm text-destructive">{error}</p>
            )}

            {isLoading && (
              <div className="mt-4 space-y-1 rounded-lg border bg-muted/30 p-4">
                {STEPS.map((step) => renderStep(step, stepStatuses[step.id]))}
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isLoading ? 'Adding...' : 'Add Project'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
