import { useEffect, useRef } from 'react'
import { useQueryState } from 'nuqs'
import { useStore } from '../store/useStore'
import { SYSTEM_VIEWS, VIEW_LABELS } from '../store/sceneMapping'
import type { SlideOverContent } from '../store/useStore'

export function SystemScene() {
  const [view] = useQueryState('view')
  const initialView = useRef(view)
  const didSet = useRef(false)
  const activeView = view && SYSTEM_VIEWS.includes(view) ? view : 'health'

  useEffect(() => {
    if (didSet.current && initialView.current === view) return
    didSet.current = true
    const existing = useStore.getState().slideOverContent
    if (existing && activeView !== 'health') return
    useStore.getState().setSlideOverContent({
      type: activeView as SlideOverContent['type'],
      title: VIEW_LABELS[activeView] ?? activeView,
    })
  }, [activeView, view])

  return null
}
