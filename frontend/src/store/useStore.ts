import { create } from 'zustand'
import * as Y from 'yjs'
import { WebrtcProvider } from 'y-webrtc'

interface AppState {
  activeTab: string
  setActiveTab: (tab: string) => void
  searchQuery: string
  setSearchQuery: (query: string) => void
  selectedObject: string
  setSelectedObject: (obj: string) => void
  globalFilter: string
  setGlobalFilter: (filter: string) => void
  projectID: string
  apiKey: string
  setProjectContext: (projectID: string, apiKey: string) => void
  predictions: any[]
  setPredictions: (preds: any[]) => void
  timeOffset: number 
  setTimeOffset: (offset: number) => void
}

const ydoc = new Y.Doc()
let provider: any | null = null
const yMap = ydoc.getMap('state')

const simpleHash = (str: string) => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash;
  }
  return Math.abs(hash).toString(16);
};

export const useStore = create<AppState>((set) => {
  yMap.observe((event: any) => {
    const newState: Partial<AppState> = {}
    event.keysChanged.forEach((key: string) => {
      (newState as any)[key] = yMap.get(key)
    })
    set((state) => ({ ...state, ...newState }))
  })

  return {
    activeTab: (yMap.get('activeTab') as string) || 'Explorer',
    setActiveTab: (tab) => {
      yMap.set('activeTab', tab)
      set({ activeTab: tab })
    },
    searchQuery: (yMap.get('searchQuery') as string) || '',
    setSearchQuery: (query) => {
      yMap.set('searchQuery', query)
      set({ searchQuery: query })
    },
    selectedObject: (yMap.get('selectedObject') as string) || '',
    setSelectedObject: (obj) => {
      yMap.set('selectedObject', obj)
      set({ selectedObject: obj })
    },
    globalFilter: (yMap.get('globalFilter') as string) || '',
    setGlobalFilter: (filter) => {
      yMap.set('globalFilter', filter)
      set({ globalFilter: filter })
    },
    projectID: '',
    apiKey: '',
    setProjectContext: (projectID, apiKey) => {
      if (provider) provider.destroy()
      const roomName = `aleph-nexus-${projectID}-${simpleHash(apiKey)}`
      provider = new WebrtcProvider(roomName, ydoc)
      set({ projectID, apiKey })
    },
    predictions: [],
    setPredictions: (preds) => set({ predictions: preds }),
    timeOffset: 0,
    setTimeOffset: (offset) => set({ timeOffset: offset })
  }
})
