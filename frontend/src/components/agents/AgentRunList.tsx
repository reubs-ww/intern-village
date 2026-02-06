import { Skeleton } from '@/components/ui/skeleton'
import { AgentRunItem } from './AgentRunItem'
import type { AgentRun } from '@/types/api'

interface AgentRunListProps {
  runs?: AgentRun[]
  isLoading?: boolean
  onViewLogs: (runId: string) => void
}

export function AgentRunList({ runs, isLoading, onViewLogs }: AgentRunListProps) {
  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-16 w-full rounded-lg" />
        <Skeleton className="h-16 w-full rounded-lg" />
      </div>
    )
  }

  if (!runs || runs.length === 0) {
    return (
      <div className="py-6 text-center text-sm text-muted-foreground">
        No agent runs yet
      </div>
    )
  }

  // Sort by attempt number descending (most recent first)
  const sortedRuns = [...runs].sort((a, b) => b.attempt_number - a.attempt_number)

  return (
    <div className="space-y-2">
      {sortedRuns.map((run) => (
        <AgentRunItem
          key={run.id}
          run={run}
          onViewLogs={() => onViewLogs(run.id)}
        />
      ))}
    </div>
  )
}
