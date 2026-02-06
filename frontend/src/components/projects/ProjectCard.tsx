import { GitFork, Folder, MoreVertical, Trash2 } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { Project } from '@/types/api'

interface ProjectCardProps {
  project: Project
  onClick?: () => void
  onDelete?: () => void
}

export function ProjectCard({ project, onClick, onDelete }: ProjectCardProps) {
  return (
    <Card
      className="cursor-pointer transition-colors hover:bg-accent/50"
      onClick={onClick}
    >
      <CardHeader className="flex flex-row items-start justify-between space-y-0">
        <div className="flex items-start gap-3">
          {project.is_fork ? (
            <GitFork className="mt-1 h-5 w-5 text-muted-foreground" />
          ) : (
            <Folder className="mt-1 h-5 w-5 text-muted-foreground" />
          )}
          <div>
            <CardTitle className="text-base">
              {project.github_owner}/{project.github_repo}
            </CardTitle>
            <CardDescription className="mt-1 flex items-center gap-2">
              <span>Branch: {project.default_branch}</span>
              {project.is_fork && (
                <Badge variant="secondary" className="text-xs">
                  Fork
                </Badge>
              )}
            </CardDescription>
          </div>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
            <Button variant="ghost" size="icon" className="h-8 w-8">
              <MoreVertical className="h-4 w-4" />
              <span className="sr-only">Project options</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onClick={(e) => {
                e.stopPropagation()
                onDelete?.()
              }}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete Project
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </CardHeader>
    </Card>
  )
}
