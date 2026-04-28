import type { StateCreator } from 'zustand'
import type { ChatMessage, PendingConfirmation } from './types'

export interface CopilotSlice {
  chat: ChatMessage[]
  setChat: (c: ChatMessage[]) => void
  addChatMessage: (msg: ChatMessage) => void
  clearChat: () => void
  input: string
  setInput: (i: string) => void
  isStreaming: boolean
  setIsStreaming: (s: boolean) => void
  streamAbortController: AbortController | null
  setStreamAbortController: (c: AbortController | null) => void
  cancelStream: () => void
  pendingConfirmation: PendingConfirmation | null
  setPendingConfirmation: (c: PendingConfirmation | null) => void
  selectedAgent: string
  setSelectedAgent: (a: string) => void
  splitView: boolean
  setSplitView: (s: boolean) => void
  bookmarkedIds: Set<number>
  toggleBookmark: (idx: number) => void
  chatSearchQuery: string
  setChatSearchQuery: (q: string) => void
  onlyBookmarks: boolean
  setOnlyBookmarks: (b: boolean) => void
  resetCopilot: () => void
}

export const createCopilotSlice: StateCreator<CopilotSlice> = (set) => ({
  chat: [],
  setChat: (c) => set({ chat: c }),
  addChatMessage: (msg) =>
    set((state) => ({ chat: [...state.chat, msg] })),
  clearChat: () => set({ chat: [] }),
  input: '',
  setInput: (i) => set({ input: i }),
  isStreaming: false,
  setIsStreaming: (s) => set({ isStreaming: s }),
  streamAbortController: null,
  setStreamAbortController: (c) => set({ streamAbortController: c }),
  cancelStream: () =>
    set((state) => {
      state.streamAbortController?.abort()
      return { isStreaming: false, streamAbortController: null }
    }),
  pendingConfirmation: null,
  setPendingConfirmation: (c) => set({ pendingConfirmation: c }),
  selectedAgent: '',
  setSelectedAgent: (a) => set({ selectedAgent: a }),
  splitView: false,
  setSplitView: (s) => set({ splitView: s }),
  bookmarkedIds: new Set(),
  toggleBookmark: (idx) =>
    set((state) => {
      const next = new Set(state.bookmarkedIds)
      if (next.has(idx)) next.delete(idx)
      else next.add(idx)
      return { bookmarkedIds: next }
    }),
  chatSearchQuery: '',
  setChatSearchQuery: (q) => set({ chatSearchQuery: q }),
  onlyBookmarks: false,
  setOnlyBookmarks: (b) => set({ onlyBookmarks: b }),
  resetCopilot: () => set({
    chat: [],
    input: '',
    isStreaming: false,
    streamAbortController: null,
    pendingConfirmation: null,
    selectedAgent: '',
    splitView: false,
    bookmarkedIds: new Set(),
    chatSearchQuery: '',
    onlyBookmarks: false,
  }),
})
