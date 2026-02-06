import { Loader2, AlertCircle, CheckCircle2, RefreshCw, MoreVertical, Trash2, Terminal, FileText } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { Task, TaskStatus, AgentType } from '@/types/api'
import type { ActiveRun } from '@/api/events'

interface TaskCardProps {
  task: Task
  onRetryPlanning?: () => void
  isRetrying?: boolean
  onDelete?: () => void
  activeRuns?: ActiveRun[]
  onViewLogs?: (runId: string, title: string, agentType: AgentType) => void
  onViewDetails?: () => void
}

const STATUS_CONFIG: Record<TaskStatus, {
  label: string
  variant: 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning' | 'error'
  icon?: React.ReactNode
}> = {
  PLANNING: {
    label: 'Planning...',
    variant: 'secondary',
    icon: <Loader2 className="h-3 w-3 animate-spin" />,
  },
  PLANNING_FAILED: {
    label: 'Planning Failed',
    variant: 'error',
    icon: <AlertCircle className="h-3 w-3" />,
  },
  ACTIVE: {
    label: 'Active',
    variant: 'outline',
  },
  DONE: {
    label: 'Complete',
    variant: 'success',
    icon: <CheckCircle2 className="h-3 w-3" />,
  },
}

export function TaskCard({
  task,
  onRetryPlanning,
  isRetrying,
  onDelete,
  activeRuns = [],
  onViewLogs,
  onViewDetails,
}: TaskCardProps) {
  const config = STATUS_CONFIG[task.status]

  // Find active planner run for this task
  const plannerRun = activeRuns.find(
    (run) => run.task_id === task.id && run.agent_type === 'PLANNER'
  )

  return (
    <div className="rounded-lg border bg-card p-4 h-[140px] flex flex-col">
      <div className="flex items-start justify-between gap-2 flex-1 min-h-0">
        <div className="min-w-0 flex-1">
          <h3 className="font-medium leading-none line-clamp-1">{task.title}</h3>
          <p className="mt-2 line-clamp-2 text-sm text-muted-foreground">
            {task.description}
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Badge variant={config.variant} className="gap-1">
            {config.icon}
            {config.label}
          </Badge>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
                <span className="sr-only">Task options</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={onDelete}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className="mt-3 pt-3 border-t flex gap-2">
        {task.status === 'PLANNING' && plannerRun && onViewLogs ? (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onViewLogs(plannerRun.id, `${task.title} - Planner`, 'PLANNER')}
          >
            <Terminal className="mr-2 h-3 w-3" />
            View Logs
          </Button>
        ) : task.status === 'PLANNING_FAILED' && onRetryPlanning ? (
          <Button
            variant="outline"
            size="sm"
            onClick={onRetryPlanning}
            disabled={isRetrying}
          >
            {isRetrying ? (
              <Loader2 className="mr-2 h-3 w-3 animate-spin" />
            ) : (
              <RefreshCw className="mr-2 h-3 w-3" />
            )}
            Retry Planning
          </Button>
        ) : (task.status === 'ACTIVE' || task.status === 'DONE') && onViewDetails ? (
          <Button
            variant="outline"
            size="sm"
            onClick={onViewDetails}
          >
            <FileText className="mr-2 h-3 w-3" />
            View Details
          </Button>
        ) : (
          <div className="h-8" /> // Placeholder to maintain consistent height
        )}
      </div>
    </div>
  )
}
