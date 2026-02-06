import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  Play,
  Loader2,
  CheckCircle2,
  CheckCheck,
  Clock,
  AlertCircle,
  ExternalLink,
  RotateCcw,
  GripVertical,
  Terminal,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { Subtask, SubtaskStatus, BlockedReason, AgentType } from '@/types/api'
import type { ActiveRun } from '@/api/events'

interface SubtaskCardProps {
  subtask: Subtask
  taskTitle?: string
  onStart?: () => void
  onMarkMerged?: () => void
  onRetry?: () => void
  onClick?: () => void
  isStarting?: boolean
  isMarkingMerged?: boolean
  isRetrying?: boolean
  activeRuns?: ActiveRun[]
  onViewLogs?: (runId: string, title: string, agentType: AgentType) => void
}

const STATUS_CONFIG: Record<
  SubtaskStatus,
  {
    bgClass: string
    icon?: React.ReactNode
  }
> = {
  PENDING: { bgClass: '' },
  READY: { bgClass: '' },
  IN_PROGRESS: {
    bgClass: 'border-blue-500/50 bg-blue-500/5',
    icon: <Loader2 className="h-4 w-4 animate-spin text-blue-400" />,
  },
  COMPLETED: {
    bgClass: 'border-green-500/50 bg-green-500/5',
    icon: <CheckCircle2 className="h-4 w-4 text-green-400" />,
  },
  MERGED: {
    bgClass: 'border-muted bg-muted/30',
    icon: <CheckCheck className="h-4 w-4 text-muted-foreground" />,
  },
  BLOCKED: {
    bgClass: 'border-yellow-500/50 bg-yellow-500/5',
    icon: <Clock className="h-4 w-4 text-yellow-400" />,
  },
}

function getBlockedConfig(reason: BlockedReason) {
  if (reason === 'FAILURE') {
    return {
      bgClass: 'border-red-500/50 bg-red-500/5',
      icon: <AlertCircle className="h-4 w-4 text-red-400" />,
      label: 'Failed',
      variant: 'error' as const,
    }
  }
  return {
    bgClass: 'border-yellow-500/50 bg-yellow-500/5',
    icon: <Clock className="h-4 w-4 text-yellow-400" />,
    label: 'Waiting on dependency',
    variant: 'warning' as const,
  }
}

export function SubtaskCard({
  subtask,
  taskTitle,
  onStart,
  onMarkMerged,
  onRetry,
  onClick,
  isStarting,
  isMarkingMerged,
  isRetrying,
  activeRuns = [],
  onViewLogs,
}: SubtaskCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: subtask.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  const config =
    subtask.status === 'BLOCKED'
      ? getBlockedConfig(subtask.blocked_reason)
      : STATUS_CONFIG[subtask.status]

  const isBlocked = subtask.status === 'BLOCKED'
  const isFailure = isBlocked && subtask.blocked_reason === 'FAILURE'

  // Find active worker run for this subtask
  const workerRun = activeRuns.find(
    (run) => run.subtask_id === subtask.id && run.agent_type === 'WORKER'
  )

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        'group rounded-lg border bg-card p-3 shadow-sm transition-all',
        config.bgClass,
        isDragging && 'opacity-50 shadow-lg',
        onClick && 'cursor-pointer hover:border-accent'
      )}
      onClick={onClick}
    >
      <div className="flex items-start gap-2">
        <button
          {...attributes}
          {...listeners}
          className="mt-0.5 cursor-grab opacity-0 transition-opacity group-hover:opacity-100 active:cursor-grabbing"
          onClick={(e) => e.stopPropagation()}
        >
          <GripVertical className="h-4 w-4 text-muted-foreground" />
        </button>

        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <h4 className="font-medium leading-tight">{subtask.title}</h4>
            {config.icon}
          </div>

          {taskTitle && (
            <Badge variant="outline" className="mt-2 text-xs">
              {taskTitle}
            </Badge>
          )}

          {isBlocked && (
            <Badge
              variant={'variant' in config ? config.variant : 'secondary'}
              className="mt-2"
            >
              {'label' in config ? config.label : ''}
            </Badge>
          )}

          <div className="mt-3 flex flex-wrap gap-2">
            {subtask.status === 'READY' && onStart && (
              <Button
                size="sm"
                onClick={(e) => {
                  e.stopPropagation()
                  onStart()
                }}
                disabled={isStarting}
              >
                {isStarting ? (
                  <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                ) : (
                  <Play className="mr-1 h-3 w-3" />
                )}
                Start
              </Button>
            )}

            {subtask.status === 'IN_PROGRESS' && workerRun && onViewLogs && (
              <Button
                variant="outline"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation()
                  onViewLogs(workerRun.id, `${subtask.title} - Worker`, 'WORKER')
                }}
              >
                <Terminal className="mr-1 h-3 w-3" />
                View Logs
              </Button>
            )}

            {subtask.status === 'COMPLETED' && (
              <>
                {subtask.pr_url && (
                  <Button
                    variant="outline"
                    size="sm"
                    asChild
                    onClick={(e) => e.stopPropagation()}
                  >
                    <a
                      href={subtask.pr_url}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      <ExternalLink className="mr-1 h-3 w-3" />
                      View PR
                    </a>
                  </Button>
                )}
                {onMarkMerged && (
                  <Button
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation()
                      onMarkMerged()
                    }}
                    disabled={isMarkingMerged}
                  >
                    {isMarkingMerged ? (
                      <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                    ) : (
                      <CheckCheck className="mr-1 h-3 w-3" />
                    )}
                    Mark Merged
                  </Button>
                )}
              </>
            )}

            {subtask.status === 'MERGED' && subtask.pr_url && (
              <Button
                variant="ghost"
                size="sm"
                asChild
                onClick={(e) => e.stopPropagation()}
              >
                <a
                  href={subtask.pr_url}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <ExternalLink className="mr-1 h-3 w-3" />
                  View PR
                </a>
              </Button>
            )}

            {isFailure && onRetry && (
              <Button
                variant="outline"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation()
                  onRetry()
                }}
                disabled={isRetrying}
              >
                {isRetrying ? (
                  <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                ) : (
                  <RotateCcw className="mr-1 h-3 w-3" />
                )}
                Retry
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
