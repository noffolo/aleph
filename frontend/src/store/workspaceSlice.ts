import type { StateCreator } from 'zustand'
import type {
  Agent,
  IngestionTask,
  Prediction,
  QueryData,
  Row,
  Scenario,
  Skill,
  Tool,
} from './types'

export interface WorkspaceSlice {
  sandboxInput: string
  setSandboxInput: (s: string) => void
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
  scenarios: Scenario[]
  setScenarios: (s: Scenario[]) => void
  selectedScenarioIds: string[]
  setSelectedScenarioIds: (ids: string[]) => void
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
    sandboxInput: '{}',
    setSandboxInput: (s) => set({ sandboxInput: s }),
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
    scenarios: [],
    setScenarios: (s) => set({ scenarios: s }),
    selectedScenarioIds: [],
    setSelectedScenarioIds: (ids) => set({ selectedScenarioIds: ids }),
    taskLogs: '',
    setTaskLogs: (l) => set({ taskLogs: l }),
    skills: [],
    setSkills: (s) => set({ skills: s }),
    tools: [],
    setTools: (t) => set({ tools: t }),
    resetWorkspace: () => set({
      sandboxInput: '{}',
      predictions: [],
      data: null,
      selectedRow: null,
      agents: [],
      ingestionTasks: [],
      ontologyRaw: '',
      availableObjects: [],
      scenarios: [],
      selectedScenarioIds: [],
      taskLogs: '',
      skills: [],
      tools: [],
    }),
  }
}
