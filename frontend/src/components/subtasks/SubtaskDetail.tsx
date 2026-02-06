import { useState } from 'react'
import ReactMarkdown from 'react-markdown'
import {
  Play,
  Loader2,
  CheckCheck,
  RotateCcw,
  ExternalLink,
  GitBranch,
  FileText,
  ListTree,
  History,
  Terminal,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { AgentRunList } from '@/components/agents/AgentRunList'
import { LogViewer } from '@/components/agents/LogViewer'
import { useAgentRuns, useAgentLogs } from '@/hooks/useAgentRuns'
import type { Subtask, SubtaskStatus, Task, AgentType } from '@/types/api'
import type { ActiveRun } from '@/api/events'

interface SubtaskDetailProps {
  subtask: Subtask | null
  task?: Task
  open: boolean
  onOpenChange: (open: boolean) => void
  onStart: () => void
  onMarkMerged: () => void
  onRetry: () => void
  isStarting?: boolean
  isMarkingMerged?: boolean
  isRetrying?: boolean
  onViewLogs?: (runId: string, title: string, agentType: AgentType) => void
  activeRuns?: ActiveRun[]
  isSSEConnected?: boolean
}

const STATUS_LABELS: Record<SubtaskStatus, { label: string; variant: 'default' | 'secondary' | 'success' | 'warning' | 'error' | 'outline' }> = {
  PENDING: { label: 'Pending', variant: 'secondary' },
  READY: { label: 'Ready', variant: 'outline' },
  IN_PROGRESS: { label: 'In Progress', variant: 'secondary' },
  COMPLETED: { label: 'Completed', variant: 'success' },
  MERGED: { label: 'Merged', variant: 'success' },
  BLOCKED: { label: 'Blocked', variant: 'warning' },
}

export function SubtaskDetail({
  subtask,
  task,
  open,
  onOpenChange,
  onStart,
  onMarkMerged,
  onRetry,
  isStarting,
  isMarkingMerged,
  isRetrying,
  onViewLogs,
  activeRuns = [],
  isSSEConnected = false,
}: SubtaskDetailProps) {
  const [specOpen, setSpecOpen] = useState(true)
  const [planOpen, setPlanOpen] = useState(true)
  const [runsOpen, setRunsOpen] = useState(true)
  const [logViewerOpen, setLogViewerOpen] = useState(false)
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)

  const { data: runs, isLoading: runsLoading } = useAgentRuns(
    open && subtask ? subtask.id : null,
    { isSSEConnected }
  )
  const { data: logsData, isLoading: logsLoading } = useAgentLogs(selectedRunId)

  if (!subtask) return null

  const statusConfig = STATUS_LABELS[subtask.status]
  const isBlocked = subtask.status === 'BLOCKED'
  const isFailure = isBlocked && subtask.blocked_reason === 'FAILURE'

  // Find active worker run for this subtask
  const workerRun = activeRuns.find(
    (run) => run.subtask_id === subtask.id && run.agent_type === 'WORKER'
  )

  const handleViewLogs = (runId: string) => {
    setSelectedRunId(runId)
    setLogViewerOpen(true)
  }

  const handleViewLiveLogs = () => {
    if (workerRun && onViewLogs) {
      onViewLogs(workerRun.id, `${subtask.title} - Worker`, 'WORKER')
    }
  }

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent className="w-full sm:max-w-lg overflow-hidden flex flex-col">
          <SheetHeader>
            <SheetTitle className="pr-8">{subtask.title}</SheetTitle>
            <SheetDescription className="flex items-center gap-2">
              {task && <span>{task.title}</span>}
              <Badge variant={statusConfig.variant}>{statusConfig.label}</Badge>
              {isBlocked && subtask.blocked_reason && (
                <Badge variant={isFailure ? 'error' : 'warning'}>
                  {subtask.blocked_reason === 'FAILURE'
                    ? 'Failed'
                    : 'Waiting on dependency'}
                </Badge>
              )}
            </SheetDescription>
          </SheetHeader>

          <ScrollArea className="flex-1 -mx-6 px-6">
            <div className="space-y-6 pb-6">
              {/* Actions */}
              <div className="flex flex-wrap gap-2">
                {subtask.status === 'READY' && (
                  <Button onClick={onStart} disabled={isStarting}>
                    {isStarting ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Play className="mr-2 h-4 w-4" />
                    )}
                    Start
                  </Button>
                )}

                {subtask.status === 'IN_PROGRESS' && workerRun && onViewLogs && (
                  <Button variant="outline" onClick={handleViewLiveLogs}>
                    <Terminal className="mr-2 h-4 w-4" />
                    View Live Logs
                  </Button>
                )}

                {subtask.status === 'COMPLETED' && (
                  <>
                    {subtask.pr_url && (
                      <Button variant="outline" asChild>
                        <a
                          href={subtask.pr_url}
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          <ExternalLink className="mr-2 h-4 w-4" />
                          View PR #{subtask.pr_number}
                        </a>
                      </Button>
                    )}
                    <Button onClick={onMarkMerged} disabled={isMarkingMerged}>
                      {isMarkingMerged ? (
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <CheckCheck className="mr-2 h-4 w-4" />
                      )}
                      Mark Merged
                    </Button>
                  </>
                )}

                {subtask.status === 'MERGED' && subtask.pr_url && (
                  <Button variant="outline" asChild>
                    <a
                      href={subtask.pr_url}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      <ExternalLink className="mr-2 h-4 w-4" />
                      View PR #{subtask.pr_number}
                    </a>
                  </Button>
                )}

                {isFailure && (
                  <Button
                    variant="outline"
                    onClick={onRetry}
                    disabled={isRetrying}
                  >
                    {isRetrying ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <RotateCcw className="mr-2 h-4 w-4" />
                    )}
                    Retry
                  </Button>
                )}
              </div>

              {/* Branch */}
              {subtask.branch_name && (
                <>
                  <Separator />
                  <div>
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                      <GitBranch className="h-4 w-4" />
                      Branch
                    </div>
                    <p className="mt-1 font-mono text-sm">{subtask.branch_name}</p>
                  </div>
                </>
              )}

              {/* Spec */}
              {subtask.spec && (
                <>
                  <Separator />
                  <Collapsible open={specOpen} onOpenChange={setSpecOpen}>
                    <CollapsibleTrigger className="flex w-full items-center justify-between text-left">
                      <div className="flex items-center gap-2 text-sm font-medium">
                        <FileText className="h-4 w-4 text-muted-foreground" />
                        Specification
                      </div>
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <div className="mt-3 rounded-lg border bg-muted/30 p-4 prose prose-sm prose-invert max-w-none">
                        <ReactMarkdown>{subtask.spec}</ReactMarkdown>
                      </div>
                    </CollapsibleContent>
                  </Collapsible>
                </>
              )}

              {/* Implementation Plan */}
              {subtask.implementation_plan && (
                <>
                  <Separator />
                  <Collapsible open={planOpen} onOpenChange={setPlanOpen}>
                    <CollapsibleTrigger className="flex w-full items-center justify-between text-left">
                      <div className="flex items-center gap-2 text-sm font-medium">
                        <ListTree className="h-4 w-4 text-muted-foreground" />
                        Implementation Plan
                      </div>
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <div className="mt-3 rounded-lg border bg-muted/30 p-4 prose prose-sm prose-invert max-w-none">
                        <ReactMarkdown>{subtask.implementation_plan}</ReactMarkdown>
                      </div>
                    </CollapsibleContent>
                  </Collapsible>
                </>
              )}

              {/* Agent Runs */}
              <Separator />
              <Collapsible open={runsOpen} onOpenChange={setRunsOpen}>
                <CollapsibleTrigger className="flex w-full items-center justify-between text-left">
                  <div className="flex items-center gap-2 text-sm font-medium">
                    <History className="h-4 w-4 text-muted-foreground" />
                    Agent Runs
                    {runs && runs.length > 0 && (
                      <Badge variant="secondary" className="text-xs">
                        {runs.length}
                      </Badge>
                    )}
                  </div>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <div className="mt-3">
                    <AgentRunList
                      runs={runs}
                      isLoading={runsLoading}
                      onViewLogs={handleViewLogs}
                    />
                  </div>
                </CollapsibleContent>
              </Collapsible>

              {/* Stats */}
              <Separator />
              <div className="flex items-center gap-6 text-sm text-muted-foreground">
                <div>
                  <span className="font-medium">Retries:</span> {subtask.retry_count}
                </div>
                <div>
                  <span className="font-medium">Tokens:</span>{' '}
                  {subtask.token_usage.toLocaleString()}
                </div>
              </div>
            </div>
          </ScrollArea>
        </SheetContent>
      </Sheet>

      <LogViewer
        open={logViewerOpen}
        onOpenChange={setLogViewerOpen}
        title={`Logs - Attempt ${runs?.find((r) => r.id === selectedRunId)?.attempt_number ?? ''}`}
        content={logsData?.content}
        isLoading={logsLoading}
      />
    </>
  )
}
