import { StateCreator } from 'zustand'
import { Asset } from './types'

export interface UISlice {
  showOnboarding: boolean
  setShowOnboarding: (s: boolean) => void
  showWizard: boolean
  setShowWizard: (s: boolean) => void
  isExplorerLoading: boolean
  setIsExplorerLoading: (s: boolean) => void
  selectedAssetContent: string | null
  setSelectedAssetContent: (c: string | null) => void
  selectedAssetId: string | null
  setSelectedAssetId: (id: string | null) => void
  globalSearchResults: Record<string, unknown> | null
  setGlobalSearchResults: (r: Record<string, unknown> | null) => void
  assets: Asset[]
  setAssets: (a: Asset[]) => void
  resetUI: () => void
}

export const createUISlice: StateCreator<UISlice> = (set) => ({
  showOnboarding: true,
  setShowOnboarding: (s) => set({ showOnboarding: s }),
  showWizard: false,
  setShowWizard: (s) => set({ showWizard: s }),
  isExplorerLoading: false,
  setIsExplorerLoading: (s) => set({ isExplorerLoading: s }),
  selectedAssetContent: null,
  setSelectedAssetContent: (c) => set({ selectedAssetContent: c }),
  selectedAssetId: null,
  setSelectedAssetId: (id) => set({ selectedAssetId: id }),
  globalSearchResults: null,
  setGlobalSearchResults: (r) => set({ globalSearchResults: r }),
  assets: [],
  setAssets: (a) => set({ assets: a }),
  resetUI: () => set({
    assets: [],
    globalSearchResults: null,
  }),
})
