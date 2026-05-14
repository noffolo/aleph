import { useEffect } from 'react'
import { useQueryState } from 'nuqs'
import { useStore } from '../store/useStore'
import { EXPLORE_VIEWS, VIEW_LABELS } from '../store/sceneMapping'
import { SkeletonLoader } from '../components/SkeletonLoader'
import type { SlideOverContent } from '../store/useStore'

export function ExploreScene() {
  const [view] = useQueryState('view')
  const activeView = view && EXPLORE_VIEWS.includes(view) ? view : 'explore'

  useEffect(() => {
    const existing = useStore.getState().slideOverContent
    if (existing && EXPLORE_VIEWS.includes(existing.type) && existing.type !== 'explore') return
    useStore.getState().setSlideOverContent({
      type: activeView as SlideOverContent['type'],
      title: VIEW_LABELS[activeView] ?? activeView,
    })
  }, [activeView])

  return <div className="flex items-center justify-center h-full"><SkeletonLoader rows={12} cols={1} /></div>
}
