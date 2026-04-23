import { create } from 'zustand'
import { WebrtcProvider } from 'y-webrtc'
import { AuthSlice, createAuthSlice } from './authSlice'
import { NavigationSlice, createNavigationSlice } from './navigationSlice'
import { CopilotSlice, createCopilotSlice } from './copilotSlice'
import { WorkspaceSlice, createWorkspaceSlice, ydoc } from './workspaceSlice'
import { HealthSlice, createHealthSlice } from './healthSlice'
import { UISlice, createUISlice } from './uiSlice'

export interface InlineContent {
  type: 'explore' | 'agent' | 'ontology' | 'data' | 'health' | 'skill' | 'tool' | 'component' | 'settings' | 'library' | 'predict' | null
  title: string
  data?: unknown
}

export interface SlideOverContent {
  type:
    | 'skill'
    | 'tool'
    | 'sandbox'
    | 'agent'
    | 'datasource'
    | 'component'
    | 'asset'
    | 'detail'
    | 'agent-form'
    | 'skill-form'
    | 'tool-form'
    | 'datasource-form'
    | 'component-form'
    | 'component-detail'
  title: string
  data?: unknown
}

export type AppState = AuthSlice & NavigationSlice & CopilotSlice & WorkspaceSlice & HealthSlice & UISlice

let provider: { destroy(): void } | null = null

export const useStore = create<AppState>()((...a) => ({
  ...createAuthSlice(...a),
  ...createNavigationSlice(...a),
  ...createCopilotSlice(...a),
  ...createWorkspaceSlice(...a),
  ...createHealthSlice(...a),
  ...createUISlice(...a),

  setProjectContext: (projectID, apiKey) => {
    const set = a[0]
    if (provider) provider.destroy()
    const roomName = `aleph-nexus-${projectID}`
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const { WebrtcProvider } = require('y-webrtc')
    provider = new WebrtcProvider(roomName, ydoc)

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