import type { StateCreator } from 'zustand'
import type { ApiKey, NotificationChannel, Project, RegistryComponent } from './types'

export interface AuthSlice {
  projectID: string
  setProjectContext: (projectID: string) => void
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
  setProjectContext: (projectID) => {
    set({ projectID })
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
