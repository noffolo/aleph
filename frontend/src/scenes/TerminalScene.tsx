import { lazy, Suspense } from 'react'
import { useQueryState } from 'nuqs'
import { TerminalView } from '../components/terminal/TerminalView'

const DashboardView = lazy(() => import('../components/DashboardView').then(m => ({ default: m.DashboardView })))

export function TerminalScene() {
  const [view] = useQueryState('view')

  if (view === 'dashboard') {
    return (
      <Suspense fallback={null}>
        <div className="absolute inset-0 z-10 bg-background animate-fade-in">
          <DashboardView />
        </div>
      </Suspense>
    )
  }

  return <TerminalView />
}
