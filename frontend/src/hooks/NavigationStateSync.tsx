import { useEffect } from 'react'
import { useQueryState } from 'nuqs'
import { useStore } from '../store/useStore'
import type { SlideOverContent } from '../store/useStore'
import { VIEW_TO_SCENE } from '../store/sceneMapping'

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
  const [scene, setScene] = useQueryState('scene', {
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
    if (scene) {
      useStore.getState().setCurrentScene(scene)
    } else if (view) {
      const inferred = VIEW_TO_SCENE[view] ?? null
      if (inferred) {
        useStore.getState().setCurrentScene(inferred)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    const unsubscribe = useStore.subscribe((state) => {
      if (state.currentScene !== scene) {
        setScene(state.currentScene)
      }
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
  }, [view, tab, slide, scene, setView, setTab, setSlide, setScene])

  return null
}
