import { lazy, Suspense } from 'react'
import { useStore } from '../store/useStore'
import { TerminalView } from '../components/terminal/TerminalView'
import { SkeletonLoader } from '../components/SkeletonLoader'

const loadingFallback = <div className="flex items-center justify-center h-full"><SkeletonLoader rows={12} cols={1} /></div>

const DashboardView = lazy(() => import('../components/DashboardView').then(m => ({ default: m.DashboardView })))
const ExploreScene = lazy(() => import('./ExploreScene').then(m => ({ default: m.ExploreScene })))
const AgentsScene = lazy(() => import('./AgentsScene').then(m => ({ default: m.AgentsScene })))
const SystemScene = lazy(() => import('./SystemScene').then(m => ({ default: m.SystemScene })))

export function SceneSelector() {
  const scene = useStore(s => s.currentScene)

  switch (scene) {
    case 'terminal':
      return (
        <Suspense fallback={loadingFallback}>
          <TerminalSceneInner />
        </Suspense>
      )
    case 'explore':
      return (
        <Suspense fallback={loadingFallback}>
          <ExploreScene />
        </Suspense>
      )
    case 'agents':
      return (
        <Suspense fallback={loadingFallback}>
          <AgentsScene />
        </Suspense>
      )
    case 'system':
      return (
        <Suspense fallback={loadingFallback}>
          <SystemScene />
        </Suspense>
      )
    default:
      return <TerminalView />
  }
}

function TerminalSceneInner() {
  const slideOverContent = useStore(s => s.slideOverContent)
  const isDashboard = slideOverContent?.type === 'dashboard'
  if (isDashboard) {
    return (
      <div className="absolute inset-0 z-10 bg-background animate-fade-in">
        <Suspense fallback={loadingFallback}>
          <DashboardView />
        </Suspense>
      </div>
    )
  }
  return <TerminalView />
}
