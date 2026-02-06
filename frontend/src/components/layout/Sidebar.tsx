import { GitFork, Plus, Folder, PanelLeftClose, PanelLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import type { Project } from '@/types/api'

interface SidebarProps {
  projects?: Project[]
  isLoading?: boolean
  activeProjectId?: string
  collapsed?: boolean
  onAddProject?: () => void
  onProjectClick?: (project: Project) => void
  onToggleCollapse?: () => void
}

export function Sidebar({
  projects,
  isLoading,
  activeProjectId,
  collapsed = false,
  onAddProject,
  onProjectClick,
  onToggleCollapse,
}: SidebarProps) {
  return (
    <aside
      className={cn(
        'flex h-full flex-col border-r bg-background transition-all duration-200',
        collapsed ? 'w-14' : 'w-64'
      )}
    >
      <div className="flex h-14 items-center justify-between border-b px-2">
        {!collapsed && (
          <span className="pl-2 text-sm font-medium text-muted-foreground">
            Projects
          </span>
        )}
        <div className={cn('flex items-center gap-1', collapsed && 'w-full justify-center')}>
          {!collapsed && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={onAddProject}
            >
              <Plus className="h-4 w-4" />
              <span className="sr-only">Add project</span>
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onToggleCollapse}
          >
            {collapsed ? (
              <PanelLeft className="h-4 w-4" />
            ) : (
              <PanelLeftClose className="h-4 w-4" />
            )}
            <span className="sr-only">
              {collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            </span>
          </Button>
        </div>
      </div>

      <ScrollArea className="flex-1">
        <div className={cn('space-y-1', collapsed ? 'p-1' : 'p-2')}>
          {isLoading ? (
            <>
              <Skeleton className={cn('h-9', collapsed ? 'w-10' : 'w-full')} />
              <Skeleton className={cn('h-9', collapsed ? 'w-10' : 'w-full')} />
              <Skeleton className={cn('h-9', collapsed ? 'w-10' : 'w-full')} />
            </>
          ) : projects?.length === 0 ? (
            collapsed ? (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-10 w-10"
                      onClick={onAddProject}
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="right">Add project</TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ) : (
              <div className="px-3 py-8 text-center text-sm text-muted-foreground">
                <Folder className="mx-auto mb-2 h-8 w-8 opacity-50" />
                <p>No projects yet</p>
                <Button
                  variant="link"
                  size="sm"
                  className="mt-2"
                  onClick={onAddProject}
                >
                  Add your first project
                </Button>
              </div>
            )
          ) : collapsed ? (
            <TooltipProvider>
              {projects?.map((project) => (
                <Tooltip key={project.id}>
                  <TooltipTrigger asChild>
                    <button
                      onClick={() => onProjectClick?.(project)}
                      className={cn(
                        'flex h-10 w-10 items-center justify-center rounded-md transition-colors hover:bg-accent',
                        activeProjectId === project.id && 'bg-accent'
                      )}
                    >
                      {project.is_fork ? (
                        <GitFork className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <Folder className="h-4 w-4 text-muted-foreground" />
                      )}
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="right">
                    {project.github_owner}/{project.github_repo}
                  </TooltipContent>
                </Tooltip>
              ))}
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    onClick={onAddProject}
                    className="flex h-10 w-10 items-center justify-center rounded-md transition-colors hover:bg-accent"
                  >
                    <Plus className="h-4 w-4 text-muted-foreground" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="right">Add project</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ) : (
            projects?.map((project) => (
              <button
                key={project.id}
                onClick={() => onProjectClick?.(project)}
                className={cn(
                  'flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-accent',
                  activeProjectId === project.id && 'bg-accent'
                )}
              >
                {project.is_fork ? (
                  <GitFork className="h-4 w-4 shrink-0 text-muted-foreground" />
                ) : (
                  <Folder className="h-4 w-4 shrink-0 text-muted-foreground" />
                )}
                <span className="truncate">
                  {project.github_owner}/{project.github_repo}
                </span>
              </button>
            ))
          )}
        </div>
      </ScrollArea>
    </aside>
  )
}
