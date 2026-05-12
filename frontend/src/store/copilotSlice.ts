import type { StateCreator } from 'zustand'
import type { ChatMessage } from './types'

export interface ToolCall {
  name: string
  args: string
}

export interface CopilotSlice {
  messages: ChatMessage[]
  setMessages: (m: ChatMessage[]) => void
  addChatMessage: (msg: ChatMessage) => void
  clearMessages: () => void
  isStreaming: boolean
  setIsStreaming: (s: boolean) => void
  streamingMessage: string
  setStreamingMessage: (s: string) => void
  streamingToolCalls: ToolCall[]
  setStreamingToolCalls: (c: ToolCall[]) => void
  selectedAgent: string
  setSelectedAgent: (a: string) => void
  resetCopilot: () => void
}

export const createCopilotSlice: StateCreator<CopilotSlice> = (set) => ({
  messages: [],
  setMessages: (m) => set({ messages: m }),
  addChatMessage: (msg) =>
    set((state) => ({ messages: [...state.messages, msg] })),
  clearMessages: () => set({ messages: [] }),
  isStreaming: false,
  setIsStreaming: (s) => set({ isStreaming: s }),
  streamingMessage: '',
  setStreamingMessage: (s) => set({ streamingMessage: s }),
  streamingToolCalls: [],
  setStreamingToolCalls: (c) => set({ streamingToolCalls: c }),
  selectedAgent: '',
  setSelectedAgent: (a) => set({ selectedAgent: a }),
  resetCopilot: () => set({
    messages: [],
    isStreaming: false,
    streamingMessage: '',
    streamingToolCalls: [],
    selectedAgent: '',
  }),
})
