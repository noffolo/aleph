import type { StateCreator } from 'zustand'
import type {
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
  return {
    sandboxResult: null,
    setSandboxResult: (r) => set({ sandboxResult: r }),
    sandboxInput: '{}',
    setSandboxInput: (s) => set({ sandboxInput: s }),
    searchQuery: '',
    setSearchQuery: (query) => set({ searchQuery: query }),
    selectedObject: '',
    setSelectedObject: (obj) => set({ selectedObject: obj }),
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
