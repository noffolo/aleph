import { useEffect } from 'react'
import { useQueryState } from 'nuqs'
import { useStore } from '../store/useStore'
import { SYSTEM_VIEWS, VIEW_LABELS } from '../store/sceneMapping'
import { SkeletonLoader } from '../components/SkeletonLoader'
import type { SlideOverContent } from '../store/useStore'

export function SystemScene() {
  const [view] = useQueryState('view')
  const activeView = view && SYSTEM_VIEWS.includes(view) ? view : 'health'

  useEffect(() => {
    const existing = useStore.getState().slideOverContent
    if (existing && SYSTEM_VIEWS.includes(existing.type) && existing.type !== 'health') return
    useStore.getState().setSlideOverContent({
      type: activeView as SlideOverContent['type'],
      title: VIEW_LABELS[activeView] ?? activeView,
    })
  }, [activeView])

  return (
    <div className="flex flex-col items-center justify-center h-full gap-4">
      <div className="flex items-center justify-center">
        <SkeletonLoader rows={12} cols={1} />
      </div>
    </div>
  )
}
