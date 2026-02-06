import { useState, useEffect, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { Header } from './Header'
import { Sidebar } from './Sidebar'
import type { User, Project } from '@/types/api'

const SIDEBAR_COLLAPSED_KEY = 'sidebar-collapsed'

interface LayoutProps {
  children: ReactNode
  user?: User | null
  projects?: Project[]
  projectsLoading?: boolean
  activeProjectId?: string
  projectName?: string
  showSidebar?: boolean
  onLogout?: () => void
  onAddProject?: () => void
}

export function Layout({
  children,
  user,
  projects,
  projectsLoading,
  activeProjectId,
  projectName,
  showSidebar = true,
  onLogout,
  onAddProject,
}: LayoutProps) {
  const navigate = useNavigate()
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    const stored = localStorage.getItem(SIDEBAR_COLLAPSED_KEY)
    return stored === 'true'
  })

  useEffect(() => {
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(sidebarCollapsed))
  }, [sidebarCollapsed])

  const handleProjectClick = (project: Project) => {
    navigate(`/projects/${project.id}`)
  }

  const handleToggleCollapse = () => {
    setSidebarCollapsed((prev) => !prev)
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      <Header user={user} projectName={projectName} onLogout={onLogout} />
      <div className="flex flex-1 overflow-hidden">
        {showSidebar && (
          <Sidebar
            projects={projects}
            isLoading={projectsLoading}
            activeProjectId={activeProjectId}
            collapsed={sidebarCollapsed}
            onAddProject={onAddProject}
            onProjectClick={handleProjectClick}
            onToggleCollapse={handleToggleCollapse}
          />
        )}
        <main className="flex-1 overflow-auto">{children}</main>
      </div>
    </div>
  )
}
