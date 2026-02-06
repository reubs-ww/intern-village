import { useDroppable } from '@dnd-kit/core'
import {
  SortableContext,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import type { Subtask } from '@/types/api'
import type { ReactNode } from 'react'

interface ColumnProps {
  id: string
  title: string
  count: number
  subtasks: Subtask[]
  children: ReactNode
}

export function Column({ id, title, count, subtasks, children }: ColumnProps) {
  const { setNodeRef, isOver } = useDroppable({ id })

  return (
    <div className="flex h-full w-72 flex-shrink-0 flex-col rounded-lg bg-muted/30">
      <div className="flex items-center justify-between border-b px-3 py-2">
        <h3 className="font-medium text-sm">{title}</h3>
        <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
          {count}
        </span>
      </div>

      <ScrollArea className="flex-1">
        <div
          ref={setNodeRef}
          className={cn(
            'min-h-[200px] p-2 transition-colors',
            isOver && 'bg-accent/30'
          )}
        >
          <SortableContext
            items={subtasks.map((s) => s.id)}
            strategy={verticalListSortingStrategy}
          >
            <div className="space-y-2">{children}</div>
          </SortableContext>

          {count === 0 && (
            <div className="flex h-24 items-center justify-center text-sm text-muted-foreground">
              No subtasks
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  )
}
