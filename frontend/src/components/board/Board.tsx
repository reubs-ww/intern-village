import { useState, useMemo } from 'react'
import {
  DndContext,
  PointerSensor,
  useSensor,
  useSensors,
  closestCenter,
} from '@dnd-kit/core'
import type { DragEndEvent } from '@dnd-kit/core'
import { arrayMove } from '@dnd-kit/sortable'
import { Column } from './Column'
import { SubtaskCard } from './SubtaskCard'
import { TaskCard } from '@/components/tasks/TaskCard'
import { SUBTASK_COLUMNS, COLUMN_STATUS_MAP } from '@/lib/constants'
import type { Subtask, Task, AgentType } from '@/types/api'
import type { ActiveRun } from '@/api/events'

interface BoardProps {
  tasks: Task[]
  subtasks: Subtask[]
  onStart: (subtaskId: string) => void
  onMarkMerged: (subtaskId: string) => void
  onRetry: (subtaskId: string) => void
  onPositionUpdate: (subtaskId: string, newPosition: number) => void
  onSubtaskClick: (subtask: Subtask) => void
  startingIds?: Set<string>
  mergingIds?: Set<string>
  retryingIds?: Set<string>
  activeRuns?: ActiveRun[]
  onViewLogs?: (runId: string, title: string, agentType: AgentType) => void
  onRetryPlanning?: (taskId: string) => void
  retryingPlanningIds?: Set<string>
  onDeleteTask?: (taskId: string) => void
  onViewTaskDetails?: (task: Task) => void
}

export function Board({
  tasks,
  subtasks,
  onStart,
  onMarkMerged,
  onRetry,
  onPositionUpdate,
  onSubtaskClick,
  startingIds = new Set(),
  mergingIds = new Set(),
  retryingIds = new Set(),
  activeRuns = [],
  onViewLogs,
  onRetryPlanning,
  retryingPlanningIds = new Set(),
  onDeleteTask,
  onViewTaskDetails,
}: BoardProps) {
  const [localSubtasks, setLocalSubtasks] = useState<Subtask[]>([])

  // Use local state for optimistic reordering, or fall back to props
  const displaySubtasks = localSubtasks.length > 0 ? localSubtasks : subtasks

  // Reset local state when props change
  useMemo(() => {
    setLocalSubtasks([])
  }, [subtasks])

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    })
  )

  const taskMap = useMemo(() => {
    const map = new Map<string, Task>()
    tasks.forEach((t) => map.set(t.id, t))
    return map
  }, [tasks])

  // Tasks that are still being planned (show TaskCards for these)
  const planningTasks = useMemo(() => {
    return tasks.filter((t) => t.status === 'PLANNING' || t.status === 'PLANNING_FAILED')
  }, [tasks])

  // Tasks that are in progress (ACTIVE status)
  const activeTasks = useMemo(() => {
    return tasks.filter((t) => t.status === 'ACTIVE')
  }, [tasks])

  // Completed tasks (DONE status)
  const completedTasks = useMemo(() => {
    return tasks.filter((t) => t.status === 'DONE')
  }, [tasks])

  const columnSubtasks = useMemo(() => {
    const result: Record<string, Subtask[]> = {}

    SUBTASK_COLUMNS.forEach((column) => {
      const statuses = COLUMN_STATUS_MAP[column]
      result[column] = displaySubtasks
        .filter((s) => statuses.includes(s.status))
        .sort((a, b) => a.position - b.position)
    })

    return result
  }, [displaySubtasks])

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event

    if (!over || active.id === over.id) return

    const activeSubtask = displaySubtasks.find((s) => s.id === active.id)
    const overSubtask = displaySubtasks.find((s) => s.id === over.id)

    if (!activeSubtask || !overSubtask) return

    // Only allow reordering within the same column/status
    if (activeSubtask.status !== overSubtask.status) return

    const column = SUBTASK_COLUMNS.find((c) =>
      COLUMN_STATUS_MAP[c].includes(activeSubtask.status)
    )
    if (!column) return

    const columnItems = columnSubtasks[column]
    const oldIndex = columnItems.findIndex((s) => s.id === active.id)
    const newIndex = columnItems.findIndex((s) => s.id === over.id)

    if (oldIndex === -1 || newIndex === -1) return

    // Optimistically update local state
    const reordered = arrayMove(columnItems, oldIndex, newIndex)

    // Calculate new position
    let newPosition: number
    if (newIndex === 0) {
      newPosition = reordered[1]?.position ? reordered[1].position - 1 : 0
    } else if (newIndex === reordered.length - 1) {
      newPosition = reordered[newIndex - 1].position + 1
    } else {
      const before = reordered[newIndex - 1].position
      const after = reordered[newIndex + 1].position
      newPosition = (before + after) / 2
    }

    // Update local state for optimistic UI
    const updated = displaySubtasks.map((s) =>
      s.id === active.id ? { ...s, position: newPosition } : s
    )
    setLocalSubtasks(updated)

    // Persist to server
    onPositionUpdate(active.id as string, newPosition)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Planning Tasks Section */}
      {planningTasks.length > 0 && (
        <div className="border-b px-4 py-3">
          <h3 className="text-sm font-medium text-muted-foreground mb-3">Planning</h3>
          <div className="flex gap-3 overflow-x-auto pb-1">
            {planningTasks.map((task) => (
              <div key={task.id} className="w-80 shrink-0">
                <TaskCard
                  task={task}
                  activeRuns={activeRuns}
                  onViewLogs={onViewLogs}
                  onRetryPlanning={onRetryPlanning ? () => onRetryPlanning(task.id) : undefined}
                  isRetrying={retryingPlanningIds.has(task.id)}
                  onDelete={onDeleteTask ? () => onDeleteTask(task.id) : undefined}
                />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Active Tasks Section */}
      {activeTasks.length > 0 && (
        <div className="border-b px-4 py-3">
          <h3 className="text-sm font-medium text-muted-foreground mb-3">
            Active Tasks ({activeTasks.length})
          </h3>
          <div className="flex gap-3 overflow-x-auto pb-1">
            {activeTasks.map((task) => (
              <div key={task.id} className="w-80 shrink-0">
                <TaskCard
                  task={task}
                  activeRuns={activeRuns}
                  onViewLogs={onViewLogs}
                  onDelete={onDeleteTask ? () => onDeleteTask(task.id) : undefined}
                  onViewDetails={onViewTaskDetails ? () => onViewTaskDetails(task) : undefined}
                />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Completed Tasks Section */}
      {completedTasks.length > 0 && (
        <div className="border-b px-4 py-3">
          <h3 className="text-sm font-medium text-green-600 mb-3">
            Completed Tasks ({completedTasks.length})
          </h3>
          <div className="flex gap-3 overflow-x-auto pb-1">
            {completedTasks.map((task) => (
              <div key={task.id} className="w-80 shrink-0">
                <TaskCard
                  task={task}
                  activeRuns={activeRuns}
                  onViewLogs={onViewLogs}
                  onDelete={onDeleteTask ? () => onDeleteTask(task.id) : undefined}
                  onViewDetails={onViewTaskDetails ? () => onViewTaskDetails(task) : undefined}
                />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Subtask Columns */}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <div className="flex gap-4 overflow-x-auto p-4 flex-1">
        {SUBTASK_COLUMNS.map((column) => {
          const items = columnSubtasks[column]
          return (
            <Column
              key={column}
              id={column}
              title={column}
              count={items.length}
              subtasks={items}
            >
              {items.map((subtask) => (
                <SubtaskCard
                  key={subtask.id}
                  subtask={subtask}
                  taskTitle={taskMap.get(subtask.task_id)?.title}
                  onStart={() => onStart(subtask.id)}
                  onMarkMerged={() => onMarkMerged(subtask.id)}
                  onRetry={() => onRetry(subtask.id)}
                  onClick={() => onSubtaskClick(subtask)}
                  isStarting={startingIds.has(subtask.id)}
                  isMarkingMerged={mergingIds.has(subtask.id)}
                  isRetrying={retryingIds.has(subtask.id)}
                  activeRuns={activeRuns}
                  onViewLogs={onViewLogs}
                />
              ))}
            </Column>
          )
        })}
        </div>
      </DndContext>
    </div>
  )
}
