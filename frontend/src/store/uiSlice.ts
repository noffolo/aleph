import type { StateCreator } from 'zustand'
import type { Asset } from './types'

export interface ToastMessage {
  id: string
  message: string
  type: 'error' | 'info' | 'success'
  context?: string
  retry?: () => void
}

export interface UISlice {
  showOnboarding: boolean
  setShowOnboarding: (s: boolean) => void
  showWizard: boolean
  setShowWizard: (s: boolean) => void
  selectedAssetContent: string | null
  setSelectedAssetContent: (c: string | null) => void
  selectedAssetId: string | null
  setSelectedAssetId: (id: string | null) => void
  assets: Asset[]
  setAssets: (a: Asset[]) => void
  enableScanline: boolean
  setEnableScanline: (v: boolean) => void
  enableGlow: boolean
  setEnableGlow: (v: boolean) => void
  enableFlicker: boolean
  setEnableFlicker: (v: boolean) => void
  toastMessages: ToastMessage[]
  addToast: (t: Omit<ToastMessage, 'id'>) => void
  removeToast: (id: string) => void
  inputMode: boolean
  setInputMode: (v: boolean) => void
  /** Progressive disclosure: expanded section keys */
  expandedSections: Record<string, boolean>
  toggleSection: (key: string) => void
  collapseAll: () => void
  expandAll: () => void
  resetUI: () => void
}

export type ExpandedSections = Record<string, boolean>

let _toastCounter = 0

export const createUISlice: StateCreator<UISlice> = (set) => ({
  showOnboarding: true,
  setShowOnboarding: (s) => set({ showOnboarding: s }),
  showWizard: false,
  setShowWizard: (s) => set({ showWizard: s }),
  selectedAssetContent: null,
  setSelectedAssetContent: (c) => set({ selectedAssetContent: c }),
  selectedAssetId: null,
  setSelectedAssetId: (id) => set({ selectedAssetId: id }),
  assets: [],
  setAssets: (a) => set({ assets: a }),
  enableScanline: true,
  setEnableScanline: (v: boolean) => set({ enableScanline: v }),
  enableGlow: false,
  setEnableGlow: (v: boolean) => set({ enableGlow: v }),
  enableFlicker: false,
  setEnableFlicker: (v: boolean) => set({ enableFlicker: v }),
  toastMessages: [],
  addToast: (t) =>
    set((state) => ({
      toastMessages: [...state.toastMessages, { ...t, id: `toast-${++_toastCounter}` }],
    })),
  removeToast: (id) =>
    set((state) => ({
      toastMessages: state.toastMessages.filter((m) => m.id !== id),
    })),
  inputMode: false,
  setInputMode: (v) => set({ inputMode: v }),
  expandedSections: {},
  toggleSection: (key) =>
    set((state) => ({
      expandedSections: {
        ...state.expandedSections,
        [key]: !state.expandedSections[key],
      },
    })),
  collapseAll: () => set({ expandedSections: {} }),
  expandAll: () => set({ expandedSections: {} }),
  resetUI: () => set({
    assets: [],
    toastMessages: [],
    inputMode: false,
    enableScanline: true,
    enableGlow: false,
    enableFlicker: false,
  }),
})
