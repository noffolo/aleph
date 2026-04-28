import { useEffect } from 'react'
import { useQueryState } from 'nuqs'
import { useStore } from '../store/useStore'
import type { SlideOverContent } from '../store/useStore'

export function NavigationStateSync() {
  const [view, setView] = useQueryState('view', {
    defaultValue: 'copilot',
  })
  const [tab, setTab] = useQueryState('tab', {
    defaultValue: 'table',
  })
  const [slide, setSlide] = useQueryState('slide', {
    defaultValue: null,
    parse: (v: string | null) => v || null,
    serialize: (v: string | null) => v || '',
  })

  useEffect(() => {
    useStore.getState().setCurrentView(view as 'copilot' | 'inline')
    useStore.getState().setActiveView(tab)
    if (slide) {
      useStore.getState().setSlideOverContent({
        type: slide as SlideOverContent['type'],
        title: '',
      })
    } else {
      useStore.getState().setSlideOverContent(null)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    const unsubscribe = useStore.subscribe((state) => {
      if (state.currentView !== view) {
        setView(state.currentView)
      }
      if (state.activeView !== tab) {
        setTab(state.activeView)
      }
      const slideType = state.slideOverContent?.type ?? null
      if (slideType !== slide) {
        setSlide(slideType)
      }
    })

    return () => unsubscribe()
  }, [view, tab, slide, setView, setTab, setSlide])

  return null
}
