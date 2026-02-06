import { useEffect, useRef, useState, useCallback, useMemo } from 'react'
import {
  Copy,
  Check,
  X,
  Loader2,
  Terminal,
  ArrowDownToLine,
  Pause,
  AlertCircle,
  Search,
  Pencil,
  Play,
  MessageSquare,
  CheckCircle2,
  Sparkles,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { useLiveLog } from '@/hooks/useLiveLog'
import type { AgentType } from '@/types/api'
import type { LogLine } from '@/api/events'
import { cn } from '@/lib/utils'

/**
 * Agent activity status inferred from log content
 */
interface AgentStatus {
  label: string
  icon: React.ReactNode
  colorClass: string
}

/**
 * Infer agent activity status from a log line
 */
function inferStatusFromLog(content: string): AgentStatus | null {
  const lowerContent = content.toLowerCase()

  // Reading/Exploring files
  if (
    content.includes('🔍') ||
    content.includes('📖') ||
    content.includes('📂') ||
    lowerContent.includes('reading') ||
    lowerContent.includes('exploring') ||
    lowerContent.includes('searching') ||
    lowerContent.includes('looking at') ||
    lowerContent.includes('examining')
  ) {
    return {
      label: 'Reading files...',
      icon: <Search className="h-3.5 w-3.5" />,
      colorClass: 'text-blue-500 bg-blue-500/10 border-blue-500/20',
    }
  }

  // Writing/Editing code
  if (
    content.includes('✏️') ||
    content.includes('📝') ||
    lowerContent.includes('writing') ||
    lowerContent.includes('editing') ||
    lowerContent.includes('creating') ||
    lowerContent.includes('updating') ||
    lowerContent.includes('modifying')
  ) {
    return {
      label: 'Writing code...',
      icon: <Pencil className="h-3.5 w-3.5" />,
      colorClass: 'text-yellow-500 bg-yellow-500/10 border-yellow-500/20',
    }
  }

  // Running commands
  if (
    content.includes('🛠️') ||
    content.includes('⚙️') ||
    content.includes('🔧') ||
    lowerContent.includes('running') ||
    lowerContent.includes('executing') ||
    lowerContent.includes('building') ||
    lowerContent.includes('testing')
  ) {
    return {
      label: 'Running command...',
      icon: <Play className="h-3.5 w-3.5" />,
      colorClass: 'text-orange-500 bg-orange-500/10 border-orange-500/20',
    }
  }

  // Thinking/Planning
  if (
    content.includes('💬') ||
    content.includes('🤔') ||
    content.includes('💭') ||
    lowerContent.includes("i'll") ||
    lowerContent.includes('let me') ||
    lowerContent.includes('i will') ||
    lowerContent.includes('planning') ||
    lowerContent.includes('analyzing')
  ) {
    return {
      label: 'Thinking...',
      icon: <MessageSquare className="h-3.5 w-3.5" />,
      colorClass: 'text-purple-500 bg-purple-500/10 border-purple-500/20',
    }
  }

  // Completed/Done
  if (
    content.includes('✅') ||
    content.includes('✓') ||
    lowerContent.includes('complete') ||
    lowerContent.includes('finished') ||
    lowerContent.includes('done') ||
    lowerContent.includes('successfully')
  ) {
    return {
      label: 'Completed',
      icon: <CheckCircle2 className="h-3.5 w-3.5" />,
      colorClass: 'text-green-500 bg-green-500/10 border-green-500/20',
    }
  }

  return null
}

/**
 * Get the current agent status from the most recent log lines
 */
function getCurrentAgentStatus(lines: LogLine[]): AgentStatus | null {
  // Check from most recent to oldest
  for (let i = lines.length - 1; i >= Math.max(0, lines.length - 5); i--) {
    const status = inferStatusFromLog(lines[i].content)
    if (status) return status
  }
  return null
}

interface LiveLogPanelProps {
  runId: string | null
  title: string
  agentType: AgentType
  onClose: () => void
  open: boolean
}

/**
 * LiveLogPanel displays real-time streaming logs from an agent run.
 * Uses a Sheet (side panel) that slides in from the right.
 */
export function LiveLogPanel({
  runId,
  title,
  agentType,
  onClose,
  open,
}: LiveLogPanelProps) {
  const { lines, isStreaming, error } = useLiveLog(open ? runId : null)
  const [copied, setCopied] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const scrollRef = useRef<HTMLDivElement>(null)
  const lastLineCountRef = useRef(0)

  // Handle copy to clipboard
  const handleCopy = useCallback(async () => {
    if (lines.length === 0) return
    try {
      const content = lines.map((line) => line.content).join('\n')
      await navigator.clipboard.writeText(content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Clipboard API not available
    }
  }, [lines])

  // Auto-scroll to bottom when new lines arrive
  useEffect(() => {
    if (!autoScroll || !scrollRef.current) return
    if (lines.length > lastLineCountRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
    lastLineCountRef.current = lines.length
  }, [lines.length, autoScroll])

  // Reset auto-scroll when panel opens
  useEffect(() => {
    if (open) {
      setAutoScroll(true)
      lastLineCountRef.current = 0
    }
  }, [open])

  // Handle user scroll - disable auto-scroll if user scrolls up
  const handleScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    const target = e.target as HTMLDivElement
    const isAtBottom =
      Math.abs(target.scrollHeight - target.scrollTop - target.clientHeight) < 10
    // Only disable auto-scroll if user scrolls up, re-enable if at bottom
    if (!isAtBottom && autoScroll) {
      setAutoScroll(false)
    } else if (isAtBottom && !autoScroll) {
      setAutoScroll(true)
    }
  }, [autoScroll])

  // Parse timestamp from log line and highlight it
  const renderLogLine = useCallback((line: LogLine) => {
    const content = line.content
    // Match timestamp pattern [HH:MM:SS] at the start
    const timestampMatch = content.match(/^\[(\d{2}:\d{2}:\d{2})\]/)

    if (timestampMatch) {
      const timestamp = timestampMatch[0]
      const rest = content.slice(timestamp.length)
      return (
        <>
          <span className="text-blue-500 dark:text-blue-400">{timestamp}</span>
          <span className="text-foreground">{rest}</span>
        </>
      )
    }

    return <span className="text-foreground">{content || ' '}</span>
  }, [])

  // Get current agent activity status from logs
  const agentStatus = useMemo(() => getCurrentAgentStatus(lines), [lines])

  // Compute connection status display
  const getConnectionStatusDisplay = () => {
    if (error) {
      return (
        <Badge variant="destructive" className="gap-1">
          <AlertCircle className="h-3 w-3" />
          Error
        </Badge>
      )
    }
    if (isStreaming) {
      return (
        <Badge variant="secondary" className="gap-1 text-green-600 bg-green-500/10 border-green-500/20">
          <span className="relative flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
          </span>
          Live
        </Badge>
      )
    }
    return (
      <Badge variant="success" className="gap-1">
        <Check className="h-3 w-3" />
        Complete
      </Badge>
    )
  }

  return (
    <Sheet open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <SheetContent className="w-full sm:max-w-2xl flex flex-col p-0 gap-0">
        {/* Header */}
        <SheetHeader className="flex flex-row items-center justify-between px-6 py-4 border-b">
          <div className="flex items-center gap-3">
            <Terminal className="h-5 w-5 text-muted-foreground" />
            <div>
              <SheetTitle className="text-left">{title}</SheetTitle>
              <SheetDescription className="sr-only">
                Real-time agent logs for {agentType.toLowerCase()} run
              </SheetDescription>
              <div className="flex items-center gap-2 mt-1">
                <Badge
                  variant={agentType === 'PLANNER' ? 'secondary' : 'default'}
                  className="text-xs"
                >
                  {agentType}
                </Badge>
                {getConnectionStatusDisplay()}
              </div>
            </div>
          </div>
        </SheetHeader>

        {/* Agent activity status bar */}
        {isStreaming && agentStatus && (
          <div className={cn(
            "flex items-center gap-2 px-6 py-2.5 border-b",
            agentStatus.colorClass
          )}>
            {agentStatus.icon}
            <span className="text-sm font-medium">{agentStatus.label}</span>
            <Sparkles className="h-3 w-3 ml-auto opacity-60 animate-pulse" />
          </div>
        )}

        {/* Error message */}
        {error && (
          <div className="px-6 py-3 bg-destructive/10 border-b border-destructive/20">
            <p className="text-sm text-destructive flex items-center gap-2">
              <AlertCircle className="h-4 w-4" />
              Connection error: {error}
            </p>
          </div>
        )}

        {/* Log content area */}
        <div className="flex-1 min-h-0 bg-muted/50">
          {lines.length === 0 && !isStreaming && !error ? (
            <div className="flex h-full items-center justify-center">
              <div className="text-center text-muted-foreground">
                <X className="mx-auto mb-2 h-8 w-8 opacity-50" />
                <p>No logs available</p>
              </div>
            </div>
          ) : lines.length === 0 && isStreaming ? (
            <div className="flex h-full items-center justify-center">
              <div className="text-center text-muted-foreground">
                <Loader2 className="mx-auto mb-2 h-8 w-8 animate-spin opacity-50" />
                <p>Waiting for logs...</p>
              </div>
            </div>
          ) : (
            <div
              ref={scrollRef}
              onScroll={handleScroll}
              className="h-full overflow-auto"
            >
              <div className="p-4">
                <pre className="font-mono text-sm leading-relaxed">
                  {lines.map((line) => (
                    <div key={line.lineNumber} className="flex">
                      <span className="mr-4 w-10 select-none text-right text-muted-foreground/50">
                        {line.lineNumber}
                      </span>
                      <span className="flex-1 whitespace-pre-wrap break-all">
                        {renderLogLine(line)}
                      </span>
                    </div>
                  ))}
                </pre>
              </div>
            </div>
          )}
        </div>

        {/* Footer with actions */}
        <div className="flex items-center justify-between px-6 py-3 border-t bg-background">
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setAutoScroll(!autoScroll)}
              className={cn(
                autoScroll && 'bg-accent'
              )}
            >
              {autoScroll ? (
                <>
                  <ArrowDownToLine className="mr-2 h-4 w-4" />
                  Auto-scroll On
                </>
              ) : (
                <>
                  <Pause className="mr-2 h-4 w-4" />
                  Auto-scroll Off
                </>
              )}
            </Button>
            <span className="text-xs text-muted-foreground">
              {lines.length} {lines.length === 1 ? 'line' : 'lines'}
            </span>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={handleCopy}
            disabled={lines.length === 0}
          >
            {copied ? (
              <>
                <Check className="mr-2 h-4 w-4" />
                Copied
              </>
            ) : (
              <>
                <Copy className="mr-2 h-4 w-4" />
                Copy All
              </>
            )}
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  )
}
