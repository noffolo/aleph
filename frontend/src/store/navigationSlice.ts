import type { StateCreator } from 'zustand'
import type { InlineContent, SlideOverContent } from './useStore'

export interface NavigationSlice {
  currentView: 'copilot' | 'inline'
  setCurrentView: (v: 'copilot' | 'inline') => void
  currentScene: string | null
  setCurrentScene: (s: string | null) => void
  inlineContent: InlineContent | null
  setInlineContent: (c: InlineContent | null) => void
  showInlinePanel: boolean
  setShowInlinePanel: (s: boolean) => void
  slideOverContent: SlideOverContent | null
  setSlideOverContent: (c: SlideOverContent | null) => void
  isCommandPaletteOpen: boolean
  setIsCommandPaletteOpen: (s: boolean) => void
  resetNavigation: () => void
}

export const createNavigationSlice: StateCreator<NavigationSlice> = (set) => ({
  currentView: 'copilot',
  setCurrentView: (v) => set({ currentView: v }),
  currentScene: null,
  setCurrentScene: (s) => set({ currentScene: s }),
  inlineContent: null,
  setInlineContent: (c) => set({ inlineContent: c }),
  showInlinePanel: false,
  setShowInlinePanel: (s) => set({ showInlinePanel: s }),
  slideOverContent: null,
  setSlideOverContent: (c) => set({ slideOverContent: c }),
  isCommandPaletteOpen: false,
  setIsCommandPaletteOpen: (s) => set({ isCommandPaletteOpen: s }),
  resetNavigation: () => {
    set({
      currentView: 'copilot',
      currentScene: null,
      inlineContent: null,
      showInlinePanel: false,
      slideOverContent: null,
      isCommandPaletteOpen: false,
    })
  },
})
