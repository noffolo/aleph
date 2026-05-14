import type { StateCreator } from 'zustand'
import type { ColumnStats } from './types'

export interface HealthSlice {
  ollamaHealthy: boolean
  setOllamaHealthy: (h: boolean) => void
  nlpHealthy: boolean
  setNlpHealthy: (h: boolean) => void
  dataHealthStats: ColumnStats[]
  setDataHealthStats: (s: ColumnStats[]) => void
  ollamaModels: string[]
  setOllamaModels: (m: string[]) => void
  resetHealth: () => void
}

export const createHealthSlice: StateCreator<HealthSlice> = (set) => ({
  ollamaHealthy: false,
  setOllamaHealthy: (h) => set({ ollamaHealthy: h }),
  nlpHealthy: false,
  setNlpHealthy: (h) => set({ nlpHealthy: h }),
  dataHealthStats: [],
  setDataHealthStats: (s) => set({ dataHealthStats: s }),
  ollamaModels: [],
  setOllamaModels: (m) => set({ ollamaModels: m }),
  resetHealth: () => set({
    ollamaHealthy: false,
    nlpHealthy: false,
    dataHealthStats: [],
    ollamaModels: [],
  }),
})
