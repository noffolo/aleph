import type { StateCreator } from 'zustand'

export interface ExplorerSlice {
  searchQuery: string
  setSearchQuery: (query: string) => void
  selectedObject: string
  setSelectedObject: (obj: string) => void
  activeView: string
  setActiveView: (v: string) => void
  isExplorerLoading: boolean
  setIsExplorerLoading: (s: boolean) => void
  globalSearchResults: Record<string, unknown> | null
  setGlobalSearchResults: (r: Record<string, unknown> | null) => void
  resetExplorer: () => void
}

export const createExplorerSlice: StateCreator<ExplorerSlice> = (set) => ({
  searchQuery: '',
  setSearchQuery: (query) => set({ searchQuery: query }),
  selectedObject: '',
  setSelectedObject: (obj) => set({ selectedObject: obj }),
  activeView: 'table',
  setActiveView: (v) => set({ activeView: v }),
  isExplorerLoading: false,
  setIsExplorerLoading: (s) => set({ isExplorerLoading: s }),
  globalSearchResults: null,
  setGlobalSearchResults: (r) => set({ globalSearchResults: r }),
  resetExplorer: () => set({
    searchQuery: '',
    selectedObject: '',
    activeView: 'table',
    isExplorerLoading: false,
    globalSearchResults: null,
  }),
})
