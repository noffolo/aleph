import { StateCreator } from 'zustand'
import * as Y from 'yjs'
import {
  Agent,
  ChatMessage,
  IngestionTask,
  Prediction,
  QueryData,
  Row,
  SandboxResult,
  Skill,
  Tool,
} from './types'

export const ydoc = new (Y as any).Doc()
export const yMap = ydoc.getMap('state')

export interface WorkspaceSlice {
  sandboxResult: SandboxResult | null
  setSandboxResult: (r: SandboxResult | null) => void
  sandboxInput: string
  setSandboxInput: (s: string) => void
  searchQuery: string
  setSearchQuery: (query: string) => void
  selectedObject: string
  setSelectedObject: (obj: string) => void
  predictions: Prediction[]
  setPredictions: (preds: Prediction[]) => void
  data: QueryData | null
  setData: (d: QueryData | null) => void
  selectedRow: Row | null
  setSelectedRow: (r: Row | null) => void
  agents: Agent[]
  setAgents: (a: Agent[]) => void
  ingestionTasks: IngestionTask[]
  setIngestionTasks: (t: IngestionTask[]) => void
  ontologyRaw: string
  setOntologyRaw: (o: string) => void
  availableObjects: string[]
  setAvailableObjects: (o: string[]) => void
  taskLogs: string
  setTaskLogs: (l: string) => void
  skills: Skill[]
  setSkills: (s: Skill[]) => void
  tools: Tool[]
  setTools: (t: Tool[]) => void
  resetWorkspace: () => void
}

export const createWorkspaceSlice: StateCreator<WorkspaceSlice> = (set) => {
  const SYNCED_KEYS = new Set(['searchQuery', 'selectedObject'])
  let skipYMapSet = false

  if (yMap.has('activeTab')) {
    ydoc.transact(() => {
      yMap.delete('activeTab')
    })
  }

  yMap.observe((event: any) => {
    if (skipYMapSet) return
    const newState: Partial<WorkspaceSlice> = {}
    event.keysChanged.forEach((key: string) => {
      if (SYNCED_KEYS.has(key)) {
        ;(newState as Record<string, unknown>)[key] = yMap.get(key)
      }
    })
    if (Object.keys(newState).length > 0) {
      set((state) => ({ ...state, ...newState }))
    }
  })

  return {
    sandboxResult: null,
    setSandboxResult: (r) => set({ sandboxResult: r }),
    sandboxInput: '{}',
    setSandboxInput: (s) => set({ sandboxInput: s }),
    searchQuery: (yMap.get('searchQuery') as string) || '',
    setSearchQuery: (query) => {
      ydoc.transact(() => {
        yMap.set('searchQuery', query)
        skipYMapSet = true
        set({ searchQuery: query })
        queueMicrotask(() => { skipYMapSet = false })
      })
    },
    selectedObject: (yMap.get('selectedObject') as string) || '',
    setSelectedObject: (obj) => {
      ydoc.transact(() => {
        yMap.set('selectedObject', obj)
        skipYMapSet = true
        set({ selectedObject: obj })
        queueMicrotask(() => { skipYMapSet = false })
      })
    },
    predictions: [],
    setPredictions: (preds) => set({ predictions: preds }),
    data: null,
    setData: (d) => set({ data: d }),
    selectedRow: null,
    setSelectedRow: (r) => set({ selectedRow: r }),
    agents: [],
    setAgents: (a) => set({ agents: a }),
    ingestionTasks: [],
    setIngestionTasks: (t) => set({ ingestionTasks: t }),
    ontologyRaw: '',
    setOntologyRaw: (o) => set({ ontologyRaw: o }),
    availableObjects: [],
    setAvailableObjects: (o) => set({ availableObjects: o }),
    taskLogs: '',
    setTaskLogs: (l) => set({ taskLogs: l }),
    skills: [],
    setSkills: (s) => set({ skills: s }),
    tools: [],
    setTools: (t) => set({ tools: t }),
    resetWorkspace: () => set({
      sandboxResult: null,
      sandboxInput: '{}',
      predictions: [],
      data: null,
      selectedRow: null,
      agents: [],
      ingestionTasks: [],
      ontologyRaw: '',
      availableObjects: [],
      taskLogs: '',
      skills: [],
      tools: [],
    }),
  }
}
