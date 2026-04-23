import { StateCreator } from 'zustand'
import { ApiKey, NotificationChannel, Project, RegistryComponent } from './types'

export interface AuthSlice {
  projectID: string
  apiKey: string
  setProjectContext: (projectID: string, apiKey: string) => void
  resetAuth: () => void
  apiKeys: ApiKey[]
  setApiKeys: (k: ApiKey[]) => void
  projects: Project[]
  setProjects: (p: Project[]) => void
  notificationChannels: NotificationChannel[]
  setNotificationChannels: (c: NotificationChannel[]) => void
  registryComponents: RegistryComponent[]
  setRegistryComponents: (c: RegistryComponent[]) => void
}

export const createAuthSlice: StateCreator<AuthSlice> = (set) => ({
  projectID: '',
  apiKey: '',
  setProjectContext: (projectID, apiKey) => {
    set({ projectID, apiKey })
  },
  resetAuth: () => set({
    apiKeys: [],
    notificationChannels: [],
    registryComponents: [],
  }),
  apiKeys: [],
  setApiKeys: (k) => set({ apiKeys: k }),
  projects: [],
  setProjects: (p) => set({ projects: p }),
  notificationChannels: [],
  setNotificationChannels: (c) => set({ notificationChannels: c }),
  registryComponents: [],
  setRegistryComponents: (c) => set({ registryComponents: c }),
})
