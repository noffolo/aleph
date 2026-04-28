import type { StateCreator } from 'zustand'
import type { Asset } from './types'

export interface ToastMessage {
  id: string
  message: string
  type: 'error' | 'info' | 'success'
  context?: string
  retry?: () => void
}

interface ConfirmDialog {
  isOpen: boolean
  message: string
  confirmLabel?: string
  onConfirm?: () => void
}

export interface UISlice {
  showOnboarding: boolean
  setShowOnboarding: (s: boolean) => void
  showWizard: boolean
  setShowWizard: (s: boolean) => void
  showGuide: boolean
  setShowGuide: (s: boolean) => void
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
  confirmDialog: ConfirmDialog
  showConfirmDialog: (message: string, confirmLabel?: string, onConfirm?: () => void) => void
  hideConfirmDialog: () => void
  enableScanline: boolean
  setEnableScanline: (v: boolean) => void
  enableGlow: boolean
  setEnableGlow: (v: boolean) => void
  enableFlicker: boolean
  setEnableFlicker: (v: boolean) => void
  toastMessages: ToastMessage[]
  addToast: (t: Omit<ToastMessage, 'id'>) => void
  removeToast: (id: string) => void
  errorToast: { message: string, type: 'error' | 'success' | 'info', id: string } | null
  setErrorToast: (message: string, type: 'error' | 'success' | 'info') => void
  clearErrorToast: () => void
  inputMode: boolean
  setInputMode: (v: boolean) => void
  resetUI: () => void
}

let _toastCounter = 0

export const createUISlice: StateCreator<UISlice> = (set) => ({
  showOnboarding: true,
  setShowOnboarding: (s) => set({ showOnboarding: s }),
  showWizard: false,
  setShowWizard: (s) => set({ showWizard: s }),
  showGuide: false,
  setShowGuide: (s) => set({ showGuide: s }),
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
  confirmDialog: { isOpen: false, message: '' },
  showConfirmDialog: (message, confirmLabel, onConfirm) =>
    set({ confirmDialog: { isOpen: true, message, confirmLabel, onConfirm } }),
  hideConfirmDialog: () =>
    set({ confirmDialog: { isOpen: false, message: '' } }),
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
  errorToast: null,
  setErrorToast: (message, type) =>
    set({ errorToast: { message, type, id: `error-${Date.now()}` } }),
  clearErrorToast: () => set({ errorToast: null }),
  inputMode: false,
  setInputMode: (v) => set({ inputMode: v }),
  resetUI: () => set({
    assets: [],
    globalSearchResults: null,
    showGuide: false,
    confirmDialog: { isOpen: false, message: '' },
    toastMessages: [],
    inputMode: false,
    enableScanline: true,
    enableGlow: false,
    enableFlicker: false,
  }),
})
