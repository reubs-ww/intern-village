import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Folder } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Layout } from '@/components/layout/Layout'
import { ProjectCard } from '@/components/projects/ProjectCard'
import { AddProjectDialog } from '@/components/projects/AddProjectDialog'
import { DeleteProjectDialog } from '@/components/projects/DeleteProjectDialog'
import { useAuth } from '@/hooks/useAuth'
import { useProjects, useCreateProject, useDeleteProject } from '@/hooks/useProjects'
import type { Project, CreateProjectResponse } from '@/types/api'

export function ProjectsPage() {
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const { data: projects, isLoading } = useProjects()
  const createProject = useCreateProject()
  const deleteProject = useDeleteProject()

  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [projectToDelete, setProjectToDelete] = useState<Project | null>(null)

  const handleAddProject = async (repoUrl: string): Promise<CreateProjectResponse> => {
    const result = await createProject.mutateAsync(repoUrl)
    if (result.was_forked) {
      toast.success('Repository forked and added successfully')
    } else {
      toast.success('Project added successfully')
    }
    return result
  }

  const handleDeleteProject = async () => {
    if (!projectToDelete) return
    await deleteProject.mutateAsync(projectToDelete.id)
    toast.success('Project deleted')
    setProjectToDelete(null)
  }

  const handleProjectClick = (project: Project) => {
    navigate(`/projects/${project.id}`)
  }

  const openDeleteDialog = (project: Project) => {
    setProjectToDelete(project)
    setDeleteDialogOpen(true)
  }

  return (
    <Layout
      user={user}
      projects={projects}
      projectsLoading={isLoading}
      onLogout={logout}
      onAddProject={() => setAddDialogOpen(true)}
    >
      <div className="p-6">
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-2xl font-bold">Projects</h1>
          <Button onClick={() => setAddDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add Project
          </Button>
        </div>

        {isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Skeleton className="h-24 rounded-xl" />
            <Skeleton className="h-24 rounded-xl" />
            <Skeleton className="h-24 rounded-xl" />
          </div>
        ) : projects?.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <Folder className="mb-4 h-16 w-16 text-muted-foreground/50" />
            <h2 className="mb-2 text-xl font-semibold">No projects yet</h2>
            <p className="mb-6 text-muted-foreground">
              Add a GitHub repository to get started with AI-powered task management.
            </p>
            <Button onClick={() => setAddDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Add your first project
            </Button>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {projects?.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onClick={() => handleProjectClick(project)}
                onDelete={() => openDeleteDialog(project)}
              />
            ))}
          </div>
        )}
      </div>

      <AddProjectDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        onSubmit={handleAddProject}
      />

      <DeleteProjectDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        project={projectToDelete}
        onConfirm={handleDeleteProject}
        isLoading={deleteProject.isPending}
      />
    </Layout>
  )
}
