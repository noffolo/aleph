import { useEffect } from 'react'
import { useQueryState } from 'nuqs'
import { useStore } from '../store/useStore'
import { EXPLORE_VIEWS, VIEW_LABELS } from '../store/sceneMapping'
import type { SlideOverContent } from '../store/useStore'

export function ExploreScene() {
  const [view] = useQueryState('view')
  const activeView = view && EXPLORE_VIEWS.includes(view) ? view : 'explore'

  useEffect(() => {
    useStore.getState().setSlideOverContent({
      type: activeView as SlideOverContent['type'],
      title: VIEW_LABELS[activeView] ?? activeView,
    })
  }, [activeView])

  return null
}
