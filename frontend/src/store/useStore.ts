import { create } from 'zustand'
import type { AuthSlice } from './authSlice'
import { createAuthSlice } from './authSlice'
import type { NavigationSlice } from './navigationSlice'
import { createNavigationSlice } from './navigationSlice'
import type { CopilotSlice } from './copilotSlice'
import { createCopilotSlice } from './copilotSlice'
import type { WorkspaceSlice } from './workspaceSlice'
import { createWorkspaceSlice } from './workspaceSlice'
import type { HealthSlice } from './healthSlice'
import { createHealthSlice } from './healthSlice'
import type { UISlice } from './uiSlice'
import { createUISlice } from './uiSlice'

export interface InlineContent {
  type: 'explore' | 'agent' | 'ontology' | 'data' | 'health' | 'skill' | 'tool' | 'component' | 'settings' | 'library' | 'predict' | null
  title: string
  data?: unknown
}

export interface SlideOverContent {
  type:
    | 'explore'
    | 'ontology'
    | 'data'
    | 'health'
    | 'skill'
    | 'tool'
    | 'sandbox'
    | 'agent'
    | 'datasource'
    | 'component'
    | 'settings'
    | 'library'
    | 'predict'
    | 'asset'
    | 'detail'
    | 'agent-form'
    | 'skill-form'
    | 'tool-form'
    | 'datasource-form'
    | 'component-form'
    | 'component-detail'
    | 'tool-intelligence'
    | 'scenario-comparison'
  title: string
  data?: unknown
}

export type AppState = AuthSlice & NavigationSlice & CopilotSlice & WorkspaceSlice & HealthSlice & UISlice

export const useStore = create<AppState>()((...a) => ({
  ...createAuthSlice(...a),
  ...createNavigationSlice(...a),
  ...createCopilotSlice(...a),
  ...createWorkspaceSlice(...a),
  ...createHealthSlice(...a),
  ...createUISlice(...a),

  setProjectContext: (projectID, apiKey) => {
    const set = a[0]
    const state = a[1]()
    state.resetAuth()
    state.resetCopilot()
    state.resetWorkspace()
    state.resetHealth()
    state.resetUI()
    state.resetNavigation()
    set({ projectID, apiKey })
  },
}))

if (typeof window !== 'undefined') {
  (window as Window & { __ALEPH_STORE__?: typeof useStore }).__ALEPH_STORE__ = useStore
}
