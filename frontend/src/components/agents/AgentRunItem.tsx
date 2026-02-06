import { useState } from 'react'
import {
  Loader2,
  CheckCircle2,
  XCircle,
  ChevronDown,
  ChevronUp,
  Terminal,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import type { AgentRun } from '@/types/api'

interface AgentRunItemProps {
  run: AgentRun
  onViewLogs: () => void
}

function formatDuration(start: string, end: string | null): string {
  const startDate = new Date(start)
  const endDate = end ? new Date(end) : new Date()
  const durationMs = endDate.getTime() - startDate.getTime()

  const seconds = Math.floor(durationMs / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  }
  return `${seconds}s`
}

function formatTokens(tokens: number | null): string {
  if (!tokens) return '-'
  if (tokens >= 1000000) {
    return `${(tokens / 1000000).toFixed(1)}M`
  }
  if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}K`
  }
  return tokens.toString()
}

export function AgentRunItem({ run, onViewLogs }: AgentRunItemProps) {
  const [isOpen, setIsOpen] = useState(false)

  const statusIcon = {
    RUNNING: <Loader2 className="h-4 w-4 animate-spin text-blue-400" />,
    SUCCEEDED: <CheckCircle2 className="h-4 w-4 text-green-400" />,
    FAILED: <XCircle className="h-4 w-4 text-red-400" />,
  }[run.status]

  const statusVariant = {
    RUNNING: 'secondary',
    SUCCEEDED: 'success',
    FAILED: 'error',
  }[run.status] as 'secondary' | 'success' | 'error'

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger asChild>
        <button className="flex w-full items-center justify-between rounded-lg border bg-card p-3 text-left transition-colors hover:bg-accent">
          <div className="flex items-center gap-3">
            {statusIcon}
            <div>
              <div className="flex items-center gap-2">
                <span className="font-medium">
                  Attempt {run.attempt_number}
                </span>
                <Badge variant={statusVariant} className="text-xs">
                  {run.status.toLowerCase()}
                </Badge>
              </div>
              <div className="mt-1 flex items-center gap-4 text-xs text-muted-foreground">
                <span>{run.agent_type}</span>
                <span>{formatDuration(run.started_at, run.ended_at)}</span>
                <span>{formatTokens(run.token_usage)} tokens</span>
              </div>
            </div>
          </div>
          {isOpen ? (
            <ChevronUp className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          )}
        </button>
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="mt-2 rounded-lg border bg-muted/30 p-3">
          {run.error_message && (
            <div className="mb-3">
              <p className="text-sm font-medium text-destructive">Error:</p>
              <p className="mt-1 text-sm text-muted-foreground">
                {run.error_message}
              </p>
            </div>
          )}

          <Button variant="outline" size="sm" onClick={onViewLogs}>
            <Terminal className="mr-2 h-4 w-4" />
            View Full Logs
          </Button>
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
