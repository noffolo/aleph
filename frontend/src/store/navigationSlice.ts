import { StateCreator } from 'zustand'
import { InlineContent, SlideOverContent } from './useStore'

export interface NavigationSlice {
  currentView: 'copilot' | 'inline'
  setCurrentView: (v: 'copilot' | 'inline') => void
  inlineContent: InlineContent | null
  setInlineContent: (c: InlineContent | null) => void
  showInlinePanel: boolean
  setShowInlinePanel: (s: boolean) => void
  commandHistory: string[]
  addToHistory: (cmd: string) => void
  slideOverContent: SlideOverContent | null
  setSlideOverContent: (c: SlideOverContent | null) => void
  isCommandPaletteOpen: boolean
  setIsCommandPaletteOpen: (s: boolean) => void
  activeView: string
  setActiveView: (v: string) => void
  resetNavigation: () => void
}

export const createNavigationSlice: StateCreator<NavigationSlice> = (set) => ({
  currentView: 'copilot',
  setCurrentView: (v) => set({ currentView: v }),
  inlineContent: null,
  setInlineContent: (c) => set({ inlineContent: c }),
  showInlinePanel: false,
  setShowInlinePanel: (s) => set({ showInlinePanel: s }),
  commandHistory: typeof window !== 'undefined' 
    ? (() => {
        try {
          const stored = sessionStorage.getItem('aleph:commandHistory')
          return stored ? JSON.parse(stored) : []
        } catch {
          return []
        }
      })()
    : [],
  addToHistory: (cmd) =>
    set((state) => {
      const newHistory = [cmd, ...state.commandHistory].slice(0, 50)
      
      if (typeof window !== 'undefined') {
        try {
          sessionStorage.setItem('aleph:commandHistory', JSON.stringify(newHistory))
        } catch {}
      }
      
      return { commandHistory: newHistory }
    }),
  slideOverContent: null,
  setSlideOverContent: (c) => set({ slideOverContent: c }),
  isCommandPaletteOpen: false,
  setIsCommandPaletteOpen: (s) => set({ isCommandPaletteOpen: s }),
  activeView: 'table',
  setActiveView: (v) => set({ activeView: v }),
  resetNavigation: () => {
    if (typeof window !== 'undefined') {
      try {
        sessionStorage.removeItem('aleph:commandHistory')
      } catch {}
    }
    set({
      currentView: 'copilot',
      inlineContent: null,
      showInlinePanel: false,
      commandHistory: [],
      slideOverContent: null,
      isCommandPaletteOpen: false,
      activeView: 'table',
    })
  },
})