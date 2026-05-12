import { useEffect } from 'react'
import { useQueryState } from 'nuqs'
import { useStore } from '../store/useStore'
import { AGENT_VIEWS, VIEW_LABELS } from '../store/sceneMapping'
import type { SlideOverContent } from '../store/useStore'

export function AgentsScene() {
  const [view] = useQueryState('view')
  const activeView = view && AGENT_VIEWS.includes(view) ? view : 'agent'

  useEffect(() => {
    const existing = useStore.getState().slideOverContent
    if (existing && AGENT_VIEWS.includes(existing.type) && existing.type !== 'agent') return
    useStore.getState().setSlideOverContent({
      type: activeView as SlideOverContent['type'],
      title: VIEW_LABELS[activeView] ?? activeView,
    })
  }, [activeView])

  return null
}
