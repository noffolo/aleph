import { useEffect, Suspense, lazy } from 'react'
import { Sidebar } from './components/Sidebar'
import { SkeletonLoader, SkeletonList } from './components/SkeletonLoader'
const WorkspaceOnboarding = lazy(() => import('./components/WorkspaceOnboarding').then(module => ({ default: module.WorkspaceOnboarding })))
import { CommandPalette } from './components/CommandPalette'
import { NavigationStateSync } from './hooks/NavigationStateSync'
const SetupWizard = lazy(() => import('./components/SetupWizard').then(module => ({ default: module.SetupWizard })))
import { AlephErrorBoundary } from './components/AlephErrorBoundary'
import { StatusBar } from './components/terminal'
import { SlideOverPanel } from './components/terminal/SlideOverPanel'
import { ToastContainer } from './components/Toast'
import { t } from './i18n'
import { useStore } from './store/useStore'
import { useAppActions } from './hooks/useAppActions'
import { createSession } from './api/client'

const SceneSelector = lazy(() => import('./scenes/SceneSelector').then(module => ({ default: module.SceneSelector })))
const SlideOverContent = lazy(() => import('./components/terminal/SlideOverContent').then(module => ({ default: module.SlideOverContent })))

import { projectClient, authClient, queryClient } from './api/factory'

declare global {
  interface Window {
    __ALEPH_STORE__: typeof useStore
  }
}

function App() {
  const projects = useStore(s => s.projects)
  const projectID = useStore(s => s.projectID)
  const selectedObject = useStore(s => s.selectedObject)
  const selectedAgent = useStore(s => s.selectedAgent)
  const showWizard = useStore(s => s.showWizard)
  const showOnboarding = useStore(s => s.showOnboarding)
  const isCommandPaletteOpen = useStore(s => s.isCommandPaletteOpen)
  const availableObjects = useStore(s => s.availableObjects)
  const lastError = useStore(s => s.lastError)
  const slideOverContent = useStore(s => s.slideOverContent)
  const currentScene = useStore(s => s.currentScene)
  const ollamaHealthy = useStore(s => s.ollamaHealthy)
  const nlpHealthy = useStore(s => s.nlpHealthy)
  const actions = useAppActions()
  const { handleError, loadProjectData, onSend, onConfirmAction } = actions

  useEffect(() => {
    const controller = new AbortController()
    projectClient.listProjects({ signal: controller.signal }).then((res: { projects: any[] }) => useStore.getState().setProjects(res.projects)).catch((e) => {
      if (e.name === 'AbortError') return
      handleError(e, 'listProjects')
    })
    return () => controller.abort()
  }, [])

  useEffect(() => { loadProjectData() }, [loadProjectData])

  useEffect(() => {
    if (!projectID || !selectedObject) return
    const controller = new AbortController()
    useStore.getState().setIsExplorerLoading(true)
    const opts = { signal: controller.signal }
    Promise.all([
      (async () => {
        const res = await queryClient.executeQuery({ ...opts, projectId: projectID, objectType: selectedObject })
        useStore.getState().setData(res)
      })(),
      (async () => {
        const res = await queryClient.getDataStats({ ...opts, projectId: projectID, objectType: selectedObject })
        useStore.getState().setDataHealthStats(res.stats || [])
      })(),
    ]).catch((e) => {
      if (e.name === 'AbortError') return
      useStore.getState().setData(null)
      useStore.getState().setDataHealthStats([])
      handleError(e, 'loadData')
    }).finally(() => useStore.getState().setIsExplorerLoading(false))
    return () => controller.abort()
  }, [projectID, selectedObject])

  useEffect(() => {
    if (!projectID || !selectedAgent) return
    const controller = new AbortController()
    queryClient.getChatHistory({ projectId: projectID, agentId: selectedAgent }, { signal: controller.signal }).then((res: { messages?: any[] }) => {
      const messages = res.messages
      if (messages && messages.length > 0) {
        useStore.getState().setMessages(messages.map((m: { role: string; content: string; toolCall?: string; createdAt?: number }) => ({
          role: m.role as "user" | "assistant" | "system",
          content: m.content,
          toolCall: m.toolCall || '',
          requiresConfirmation: false,
          createdAt: m.createdAt || 0,
        })))
      }
    }).catch((e) => {
      if (e.name === 'AbortError') return
      handleError(e, 'getChatHistory')
    })
    return () => controller.abort()
  }, [projectID, selectedAgent])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        useStore.getState().setIsCommandPaletteOpen(!useStore.getState().isCommandPaletteOpen)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  if (showWizard) return (
    <AlephErrorBoundary>
      <Suspense fallback={<div className="flex items-center justify-center h-screen"><SkeletonLoader rows={10} cols={1} /></div>}>
                <SetupWizard
                  onLogin={async (key: string) => { await createSession(key) }}
                  onCreateProject={async (n: string) => { const r = await projectClient.createProject({ id: n.toLowerCase(), name: n }); return r.project?.id ?? n.toLowerCase() }}
                  onCreateApiKey={async (pid: string, l: string) => { const r = await authClient.createApiKey({ projectId: pid, label: l }); return r.key?.key ?? '' }}
                  onComplete={async (pid: string, key: string) => { await createSession(key); useStore.getState().setProjectContext(pid); useStore.getState().setShowWizard(false); useStore.getState().setShowOnboarding(false) }}
                />
      </Suspense>
    </AlephErrorBoundary>
  )

  if (showOnboarding) return (
    <AlephErrorBoundary>
      <Suspense fallback={<div className="flex items-center justify-center h-screen"><SkeletonList itemCount={8} /></div>}>
            <WorkspaceOnboarding
              projects={projects}
              onSelectProject={async (id: string, key: string) => { await createSession(key); useStore.getState().setProjectContext(id); useStore.getState().setShowOnboarding(false) }}
              onDeleteProject={async (id: string, key: string) => { await createSession(key); projectClient.deleteProject({ id }).then(() => projectClient.listProjects({}).then((res: { projects: any[] }) => useStore.getState().setProjects(res.projects))).catch((e) => handleError(e, 'deleteProject')) }}
              onCreateProject={() => useStore.getState().setShowWizard(true)}
            />
      </Suspense>
    </AlephErrorBoundary>
  )

  return (
    <div className="flex h-screen bg-background text-text font-mono overflow-hidden">
      <AlephErrorBoundary>
        <NavigationStateSync />
        <a href="#main-content" className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-[100] focus:px-4 focus:py-2 focus:bg-primary focus:text-background focus:rounded-lg">Skip to main content</a>
         <CommandPalette
           isOpen={isCommandPaletteOpen}
           onClose={() => useStore.getState().setIsCommandPaletteOpen(false)}
           availableObjects={availableObjects}
           projects={projects}
              onSelectProject={(id: string) => {
                const p = projects.find((x: any) => x.id === id)
                if (p) {
                  useStore.getState().setProjectContext(p.id)
                  useStore.getState().setShowOnboarding(false)
                } else {
                  useStore.getState().setShowOnboarding(true)
                }
              }}
           onSelectObject={(name: string) => {
             useStore.getState().setSelectedObject(name)
             useStore.getState().setIsCommandPaletteOpen(false)
           }}
         />
         <Sidebar projectID={projectID} onShowOnboarding={() => useStore.getState().setShowOnboarding(true)} />

        <div className="flex-1 flex flex-col overflow-hidden relative">
           {lastError && (
             <div className="mx-4 mt-4 px-4 py-2 bg-danger/10 border border-danger/30 text-danger rounded text-sm font-mono flex items-center justify-between">
               <span>{lastError}</span>
               <button onClick={() => useStore.getState().setLastError(null)} className="text-danger/60 hover:text-danger ml-4 focus:ring-2 focus:ring-primary rounded" aria-label="Dismiss error">✕</button>
             </div>
           )}

          <main id="main-content" className="flex-1 overflow-hidden relative">
            <div className="h-full w-full relative">
              <Suspense fallback={<div className="absolute inset-0 z-10 bg-background animate-fade-in"><SkeletonLoader rows={15} cols={1} /></div>}>
                <SceneSelector />
              </Suspense>
            </div>
          </main>

           {currentScene !== 'terminal' && slideOverContent && (
             <AlephErrorBoundary>
               <SlideOverPanel
                 isOpen={true}
                 onClose={() => useStore.getState().setSlideOverContent(null)}
                 title={slideOverContent.title}
               >
                 <Suspense fallback={<SkeletonLoader rows={8} cols={1} />}>
                   <SlideOverContent />
                 </Suspense>
               </SlideOverPanel>
             </AlephErrorBoundary>
           )}

           <StatusBar projectID={projectID} ollamaHealthy={ollamaHealthy} nlpHealthy={nlpHealthy} />
           <ToastContainer />
        </div>
      </AlephErrorBoundary>
    </div>
  )
}

export default App
