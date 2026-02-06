import { useState, useMemo, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Plus, ArrowLeft, Wifi, WifiOff } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Layout } from '@/components/layout/Layout'
import { Board } from '@/components/board/Board'
import { CreateTaskDialog } from '@/components/tasks/CreateTaskDialog'
import { DeleteTaskDialog } from '@/components/tasks/DeleteTaskDialog'
import { TaskDetail } from '@/components/tasks/TaskDetail'
import { SubtaskDetail } from '@/components/subtasks/SubtaskDetail'
import { LiveLogPanel } from '@/components/agents/LiveLogPanel'
import { ProjectEventsProvider, useProjectEvents } from '@/contexts/ProjectEventsContext'
import { useAuth } from '@/hooks/useAuth'
import { useProjects, useProject } from '@/hooks/useProjects'
import { useTasks, useCreateTask, useDeleteTask, useRetryPlanning } from '@/hooks/useTasks'
import {
  useSubtasks,
  useStartSubtask,
  useMarkMerged,
  useRetrySubtask,
  useUpdatePosition,
} from '@/hooks/useSubtasks'
import type { Subtask, Task, AgentType } from '@/types/api'

// Inner component that uses the ProjectEvents context
function BoardPageContent({ projectId }: { projectId: string }) {
  const navigate = useNavigate()
  const { user, logout } = useAuth()

  // Get active runs and connection status from the project events context
  // This must be called before useTasks/useSubtasks so we can pass isConnected to them
  const { activeRuns, isConnected, connectionError } = useProjectEvents()

  const { data: projects, isLoading: projectsLoading } = useProjects()
  const { data: project } = useProject(projectId)
  const { data: tasks = [], isLoading: tasksLoading } = useTasks(projectId, {
    isSSEConnected: isConnected,
  })
  const createTask = useCreateTask(projectId)
  const deleteTask = useDeleteTask(projectId)
  const retryPlanning = useRetryPlanning()
  const startSubtask = useStartSubtask()
  const markMerged = useMarkMerged()
  const retrySubtask = useRetrySubtask()
  const updatePosition = useUpdatePosition()

  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [taskToDelete, setTaskToDelete] = useState<typeof tasks[0] | null>(null)
  const [startingIds, setStartingIds] = useState<Set<string>>(new Set())
  const [mergingIds, setMergingIds] = useState<Set<string>>(new Set())
  const [retryingIds, setRetryingIds] = useState<Set<string>>(new Set())
  const [retryingPlanningIds, setRetryingPlanningIds] = useState<Set<string>>(new Set())
  const [selectedSubtask, setSelectedSubtask] = useState<Subtask | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [selectedTaskForDetails, setSelectedTaskForDetails] = useState<Task | null>(null)
  const [taskDetailOpen, setTaskDetailOpen] = useState(false)

  // Live log panel state
  const [liveLogOpen, setLiveLogOpen] = useState(false)
  const [liveLogRunId, setLiveLogRunId] = useState<string | null>(null)
  const [liveLogTitle, setLiveLogTitle] = useState('')
  const [liveLogAgentType, setLiveLogAgentType] = useState<AgentType>('WORKER')

  // Get task IDs for fetching subtasks
  const taskIds = useMemo(() => {
    return tasks.map((t) => t.id)
  }, [tasks])

  const { data: allSubtasks = [], isLoading: subtasksLoading } = useSubtasks(taskIds, {
    isSSEConnected: isConnected,
  })

  // Filter out PENDING subtasks from display
  const subtasks = useMemo(
    () => allSubtasks.filter((s) => s.status !== 'PENDING'),
    [allSubtasks]
  )

  const projectName = project
    ? `${project.github_owner}/${project.github_repo}`
    : undefined

  const selectedTask = tasks.find((t) => t.id === selectedSubtask?.task_id)

  const handleCreateTask = async (title: string, description: string) => {
    await createTask.mutateAsync({ title, description })
    toast.success('Task created - AI planner is working...')
  }

  const handleStart = async (subtaskId: string) => {
    setStartingIds((prev) => new Set(prev).add(subtaskId))
    try {
      await startSubtask.mutateAsync(subtaskId)
      toast.success('Subtask started')
    } catch {
      toast.error('Failed to start subtask')
    } finally {
      setStartingIds((prev) => {
        const next = new Set(prev)
        next.delete(subtaskId)
        return next
      })
    }
  }

  const handleMarkMerged = async (subtaskId: string) => {
    setMergingIds((prev) => new Set(prev).add(subtaskId))
    try {
      await markMerged.mutateAsync(subtaskId)
      toast.success('Marked as merged')
    } catch {
      toast.error('Failed to mark as merged')
    } finally {
      setMergingIds((prev) => {
        const next = new Set(prev)
        next.delete(subtaskId)
        return next
      })
    }
  }

  const handleRetry = async (subtaskId: string) => {
    setRetryingIds((prev) => new Set(prev).add(subtaskId))
    try {
      await retrySubtask.mutateAsync(subtaskId)
      toast.success('Retrying subtask')
    } catch {
      toast.error('Failed to retry subtask')
    } finally {
      setRetryingIds((prev) => {
        const next = new Set(prev)
        next.delete(subtaskId)
        return next
      })
    }
  }

  const handlePositionUpdate = async (subtaskId: string, position: number) => {
    try {
      await updatePosition.mutateAsync({ id: subtaskId, position })
    } catch {
      toast.error('Failed to update position')
    }
  }

  const handleRetryPlanning = async (taskId: string) => {
    setRetryingPlanningIds((prev) => new Set(prev).add(taskId))
    try {
      await retryPlanning.mutateAsync(taskId)
      toast.success('Retrying planning')
    } catch {
      toast.error('Failed to retry planning')
    } finally {
      setRetryingPlanningIds((prev) => {
        const next = new Set(prev)
        next.delete(taskId)
        return next
      })
    }
  }

  const handleSubtaskClick = (subtask: Subtask) => {
    setSelectedSubtask(subtask)
    setDetailOpen(true)
  }

  const handleViewTaskDetails = (task: Task) => {
    setSelectedTaskForDetails(task)
    setTaskDetailOpen(true)
  }

  // Handler for viewing live logs
  const handleViewLogs = useCallback(
    (runId: string, title: string, agentType: AgentType) => {
      setLiveLogRunId(runId)
      setLiveLogTitle(title)
      setLiveLogAgentType(agentType)
      setLiveLogOpen(true)
    },
    []
  )

  const handleCloseLiveLog = useCallback(() => {
    setLiveLogOpen(false)
    setLiveLogRunId(null)
  }, [])

  const handleDeleteTask = async () => {
    if (!taskToDelete) return
    try {
      await deleteTask.mutateAsync(taskToDelete.id)
      toast.success('Task deleted')
    } catch {
      toast.error('Failed to delete task')
    }
  }

  const openDeleteDialog = (task: typeof tasks[0]) => {
    setTaskToDelete(task)
    setDeleteDialogOpen(true)
  }

  const isLoading = tasksLoading || subtasksLoading

  return (
    <Layout
      user={user}
      projects={projects}
      projectsLoading={projectsLoading}
      activeProjectId={projectId}
      projectName={projectName}
      onLogout={logout}
      onAddProject={() => navigate('/')}
    >
      <div className="flex h-full flex-col">
        <div className="flex h-14 items-center justify-between border-b px-4">
          <div className="flex items-center gap-4">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => navigate('/')}
              className="md:hidden"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex items-center gap-3">
            {/* Connection status indicator */}
            {isConnected ? (
              <Badge variant="outline" className="gap-1 text-green-600 border-green-600/50">
                <Wifi className="h-3 w-3" />
                Live
              </Badge>
            ) : connectionError ? (
              <Badge variant="outline" className="gap-1 text-yellow-600 border-yellow-600/50">
                <WifiOff className="h-3 w-3" />
                Reconnecting
              </Badge>
            ) : null}
            <Button onClick={() => setCreateDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              New Task
            </Button>
          </div>
        </div>

        <div className="flex-1 overflow-auto">
          {isLoading ? (
            <div className="flex gap-4 p-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="w-72 shrink-0">
                  <Skeleton className="mb-2 h-8 w-full rounded-lg" />
                  <Skeleton className="h-24 w-full rounded-lg" />
                  <Skeleton className="mt-2 h-24 w-full rounded-lg" />
                </div>
              ))}
            </div>
          ) : tasks.length === 0 ? (
            <div className="flex h-full flex-col items-center justify-center p-8 text-center">
              <h2 className="mb-2 text-xl font-semibold">No tasks yet</h2>
              <p className="mb-6 text-muted-foreground">
                Create a task to start breaking it down into subtasks.
              </p>
              <Button onClick={() => setCreateDialogOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create your first task
              </Button>
            </div>
          ) : (
            <Board
              tasks={tasks}
              subtasks={subtasks}
              onStart={handleStart}
              onMarkMerged={handleMarkMerged}
              onRetry={handleRetry}
              onPositionUpdate={handlePositionUpdate}
              onSubtaskClick={handleSubtaskClick}
              startingIds={startingIds}
              mergingIds={mergingIds}
              retryingIds={retryingIds}
              activeRuns={activeRuns}
              onViewLogs={handleViewLogs}
              onRetryPlanning={handleRetryPlanning}
              retryingPlanningIds={retryingPlanningIds}
              onDeleteTask={(taskId) => openDeleteDialog(tasks.find(t => t.id === taskId)!)}
              onViewTaskDetails={handleViewTaskDetails}
            />
          )}
        </div>
      </div>

      <CreateTaskDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSubmit={handleCreateTask}
      />

      <SubtaskDetail
        subtask={selectedSubtask}
        task={selectedTask}
        open={detailOpen}
        onOpenChange={setDetailOpen}
        onStart={() => selectedSubtask && handleStart(selectedSubtask.id)}
        onMarkMerged={() => selectedSubtask && handleMarkMerged(selectedSubtask.id)}
        onRetry={() => selectedSubtask && handleRetry(selectedSubtask.id)}
        isStarting={selectedSubtask ? startingIds.has(selectedSubtask.id) : false}
        isMarkingMerged={selectedSubtask ? mergingIds.has(selectedSubtask.id) : false}
        isRetrying={selectedSubtask ? retryingIds.has(selectedSubtask.id) : false}
        onViewLogs={handleViewLogs}
        activeRuns={activeRuns}
        isSSEConnected={isConnected}
      />

      <DeleteTaskDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        task={taskToDelete}
        onConfirm={handleDeleteTask}
        isLoading={deleteTask.isPending}
      />

      <TaskDetail
        task={selectedTaskForDetails}
        open={taskDetailOpen}
        onOpenChange={setTaskDetailOpen}
      />

      <LiveLogPanel
        runId={liveLogRunId}
        title={liveLogTitle}
        agentType={liveLogAgentType}
        onClose={handleCloseLiveLog}
        open={liveLogOpen}
      />
    </Layout>
  )
}

// Wrapper component that provides the ProjectEvents context
export function BoardPage() {
  const { id } = useParams<{ id: string }>()

  if (!id) {
    return null
  }

  return (
    <ProjectEventsProvider projectId={id}>
      <BoardPageContent projectId={id} />
    </ProjectEventsProvider>
  )
}
