import { FileText } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import type { Task, TaskStatus } from '@/types/api'

interface TaskDetailProps {
  task: Task | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

const STATUS_CONFIG: Record<TaskStatus, {
  label: string
  variant: 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning' | 'error'
}> = {
  PLANNING: {
    label: 'Planning...',
    variant: 'secondary',
  },
  PLANNING_FAILED: {
    label: 'Planning Failed',
    variant: 'error',
  },
  ACTIVE: {
    label: 'Active',
    variant: 'outline',
  },
  DONE: {
    label: 'Complete',
    variant: 'success',
  },
}

export function TaskDetail({
  task,
  open,
  onOpenChange,
}: TaskDetailProps) {
  if (!task) return null

  const statusConfig = STATUS_CONFIG[task.status]

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-lg overflow-hidden flex flex-col">
        <SheetHeader>
          <SheetTitle className="pr-8">Task Details</SheetTitle>
          <SheetDescription className="flex items-center gap-2">
            <Badge variant={statusConfig.variant}>{statusConfig.label}</Badge>
          </SheetDescription>
        </SheetHeader>

        <ScrollArea className="flex-1 -mx-6 px-6">
          <div className="space-y-6 pb-6">
            {/* Title */}
            <div>
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <FileText className="h-4 w-4" />
                Title
              </div>
              <p className="mt-1 text-sm">{task.title}</p>
            </div>

            <Separator />

            {/* Description */}
            <div>
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <FileText className="h-4 w-4" />
                Description
              </div>
              <div className="mt-2 rounded-lg border bg-muted/30 p-4">
                <p className="text-sm whitespace-pre-wrap">{task.description || 'No description provided.'}</p>
              </div>
            </div>

            <Separator />

            {/* Created At */}
            <div className="text-sm text-muted-foreground">
              <span className="font-medium">Created:</span>{' '}
              {new Date(task.created_at).toLocaleString()}
            </div>
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
