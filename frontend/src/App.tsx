import { useEffect, Suspense, useState, lazy } from 'react'
import { Sidebar } from './components/Sidebar'
const WorkspaceOnboarding = lazy(() => import('./components/WorkspaceOnboarding').then(module => ({ default: module.WorkspaceOnboarding })))
import { CopilotView } from './components/CopilotView'
import { CommandPalette } from './components/CommandPalette'
const SetupWizard = lazy(() => import('./components/SetupWizard').then(module => ({ default: module.SetupWizard })))
import { AlephErrorBoundary } from './components/AlephErrorBoundary'
import { StatusBar } from './components/terminal'
import { InlineRenderer } from './components/terminal/InlineRenderer'
import { SlideOverPanel } from './components/terminal/SlideOverPanel'
import { TerminalEffects } from './components/terminal/TerminalEffects'
import { ToastBar } from './components/ToastBar'
import { useStore } from './store/useStore'
import { useAppActions } from './hooks/useAppActions'
import { setApiKey, getStoredApiKey } from './api/client'

const SlideOverContent = lazy(() => import('./components/terminal/SlideOverContent').then(module => ({ default: module.SlideOverContent })))

import { projectClient, queryClient, authClient } from './api/factory'

function App() {
  const store = useStore()
  const actions = useAppActions()
  const { handleError, loadProjectData, onSend, onConfirmAction } = actions

  useEffect(() => {
    projectClient.listProjects({}).then((res: any) => store.setProjects(res.projects)).catch((e) => handleError(e, 'listProjects'))
  }, [])

  useEffect(() => { loadProjectData() }, [loadProjectData])

  useEffect(() => {
    if (!store.projectID || !store.selectedObject) return
    store.setIsExplorerLoading(true)
    queryClient.executeQuery({ projectId: store.projectID, objectType: store.selectedObject, limit: 100 }).then((res: any) => {
      store.setData(res)
    }).catch((e) => {
      store.setData(null)
      handleError(e, 'executeQuery')
    }).finally(() => store.setIsExplorerLoading(false))

    queryClient.getDataStats({ projectId: store.projectID, objectType: store.selectedObject }).then((res: any) => {
      store.setDataHealthStats(res.stats || [])
    }).catch((e) => {
      store.setDataHealthStats([])
      handleError(e, 'getDataStats')
    })
  }, [store.projectID, store.selectedObject])

  useEffect(() => {
    if (!store.projectID || !store.selectedAgent) return
    queryClient.getChatHistory({ projectId: store.projectID, agentId: store.selectedAgent }).then((res: any) => {
      if (res.messages?.length > 0) {
        store.setChat(res.messages.map((m: any) => ({
          role: m.role,
          content: m.content,
          toolCall: m.toolCall || '',
          requiresConfirmation: false,
          createdAt: m.createdAt || 0,
        })))
      }
    }).catch((e) => handleError(e, 'getChatHistory'))
  }, [store.projectID, store.selectedAgent])

  useEffect(() => {
    if (!store.searchQuery || !store.projectID) {
      store.setGlobalSearchResults(null)
      return
    }
    const timer = setTimeout(() => {
      const objMatch = store.availableObjects.find(o => o.toLowerCase().includes(store.searchQuery.toLowerCase()))
      if (objMatch) {
        queryClient.globalQuery({ projectId: store.projectID, objectType: objMatch, limit: 20 }).then((res: any) => {
          store.setGlobalSearchResults(res)
        }).catch(() => {
          store.setGlobalSearchResults(null)
        })
      } else {
        store.setGlobalSearchResults(null)
      }
    }, 400)
    return () => clearTimeout(timer)
  }, [store.searchQuery, store.projectID, store.availableObjects])

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

  if (store.showWizard) return (
    <AlephErrorBoundary>
      <Suspense fallback={<div className="flex items-center justify-center h-screen text-textDim text-xs font-mono">Caricamento Setup Wizard...</div>}>
        <SetupWizard
          onCreateProject={async (n: string) => { const r = await projectClient.createProject({ id: n.toLowerCase(), name: n }); return r.project?.id ?? n.toLowerCase() }}
          onCreateApiKey={async (pid: string, l: string) => { const r = await authClient.createApiKey({ projectId: pid, label: l }); return r.key?.key ?? '' }}
          onComplete={(pid, key) => { setApiKey(key); store.setProjectContext(pid, key); store.setShowWizard(false); store.setShowOnboarding(false) }}
        />
      </Suspense>
    </AlephErrorBoundary>
  )

  if (store.showOnboarding) return (
    <AlephErrorBoundary>
      <Suspense fallback={<div className="flex items-center justify-center h-screen text-textDim text-xs font-mono">Caricamento Onboarding...</div>}>
        <WorkspaceOnboarding
          projects={store.projects}
          onSelectProject={(id, key) => { setApiKey(key); store.setProjectContext(id, key); store.setShowOnboarding(false) }}
          onDeleteProject={(id, key) => { setApiKey(key); projectClient.deleteProject({ id: id }).then(() => { projectClient.listProjects({}).then((res: any) => store.setProjects(res.projects)) }).catch((e) => handleError(e, 'deleteProject')) }}
          onCreateProject={() => store.setShowWizard(true)}
        />
      </Suspense>
    </AlephErrorBoundary>
  )

  return (
    <div className="flex h-screen bg-background text-text font-mono overflow-hidden">
      <AlephErrorBoundary>
        <a href="#main-content" className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-[100] focus:px-4 focus:py-2 focus:bg-primary focus:text-background focus:rounded-lg">Skip to main content</a>
        <CommandPalette
          isOpen={store.isCommandPaletteOpen}
          onClose={() => store.setIsCommandPaletteOpen(false)}
          availableObjects={store.availableObjects}
          projects={store.projects}
          onSelectProject={(id) => {
            const p = store.projects.find((x: any) => x.id === id)
            if (p) {
              store.setProjectContext(p.id, getStoredApiKey() || '')
              store.setShowOnboarding(false)
            } else {
              store.setShowOnboarding(true)
            }
          }}
          onSelectObject={(obj) => {
            store.setSelectedObject(obj)
            store.setInlineContent({ type: 'explore', title: obj || 'Explore' })
            store.setCurrentView('inline')
            store.setShowInlinePanel(true)
          }}
        />
        <Sidebar projectID={store.projectID} onShowOnboarding={() => store.setShowOnboarding(true)} />

        <div className="flex-1 flex flex-col overflow-hidden relative">
          {store.lastError && (
            <div className="mx-4 mt-4 px-4 py-2 bg-danger/10 border border-danger/30 text-danger rounded text-sm font-mono flex items-center justify-between">
              <span>{store.lastError}</span>
              <button onClick={() => store.setLastError(null)} className="text-danger/60 hover:text-danger ml-4">✕</button>
            </div>
          )}

           <main id="main-content" className="flex-1 overflow-hidden p-4 relative">
              <TerminalEffects />
              <CopilotView
                agents={store.agents}
                selectedAgent={store.selectedAgent}
                setSelectedAgent={store.setSelectedAgent}
                chat={store.chat}
                input={store.input}
                setInput={store.setInput}
                onSend={onSend}
                isStreaming={store.isStreaming}
                onCancelStream={() => store.cancelStream()}
                onConfirmAction={onConfirmAction}
                onClearChat={() => store.clearChat()}
              />
           </main>


          <AlephErrorBoundary>
            <InlineRenderer />
          </AlephErrorBoundary>

           {store.slideOverContent && (
             <AlephErrorBoundary>
               <SlideOverPanel
                 isOpen={true}
                 onClose={() => store.setSlideOverContent(null)}
                 title={store.slideOverContent.title}
               >
                 <Suspense fallback={<div className="p-4 text-textDim text-xs font-mono">Loading...</div>}>
                   <SlideOverContent />
                 </Suspense>
               </SlideOverPanel>
             </AlephErrorBoundary>
           )}
            <StatusBar projectID={store.projectID} ollamaHealthy={store.ollamaHealthy} nlpHealthy={store.nlpHealthy} />
            <ToastBar />
          </div>


      </AlephErrorBoundary>
    </div>
  )
}

export default App