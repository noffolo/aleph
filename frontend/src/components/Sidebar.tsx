import React from 'react'
import { LayoutGrid, Binary, Activity, Bot, Eye, Book, Zap, Database, Users, Cpu, Wrench, Package, Sliders, Settings } from 'lucide-react'
import { useStore } from '../store/useStore'
import type { SlideOverContent } from '../store/useStore'

interface SidebarProps {
  projectID: string
  onShowOnboarding: () => void
}

const ID_TO_INLINE_TYPE: Record<string, SlideOverContent['type']> = {
  'Explorer': 'explore',
  'Data Health': 'health',
  'Oracle': 'predict',
  'Library': 'library',
  'Ontologies': 'ontology',
  'Data Sources': 'data',
  'Agents': 'agent',
  'Skills': 'skill',
  'Tools': 'tool',
  'Components': 'component',
  'Settings': 'settings',
}

const PANEL_TITLES: Record<string, string> = {
  Explorer: 'Explorer',
  'Data Health': 'Data Health',
  Oracle: 'Oracle',
  Library: 'Library',
  Ontologies: 'Ontologies',
  'Data Sources': 'Data Sources',
  Agents: 'Agents',
  Skills: 'Skills',
  Tools: 'Tools',
  Components: 'Components',
  Settings: 'Settings',
}

const sidebarItems = [
  { id: 'Explorer', icon: LayoutGrid, command: '/explore' },
  { id: 'Data Health', icon: Activity, command: '/health' },
  { id: 'Copilot', icon: Bot, command: '' },
  { id: 'Oracle', icon: Eye, command: '/predict' },
  { id: 'Library', icon: Book, command: '/library' },
  { id: 'Ontologies', icon: Zap, command: '/ontology' },
  { id: 'Data Sources', icon: Database, command: '/data' },
  { id: 'Agents', icon: Users, command: '/agent' },
  { id: 'Skills', icon: Cpu, command: '/skills' },
  { id: 'Tools', icon: Wrench, command: '/tools' },
  { id: 'Components', icon: Package, command: '/components' },
  { id: 'Settings', icon: Sliders, command: '/settings' },
]

const DIVIDER_AFTER = new Set([1, 4, 7])

export const Sidebar: React.FC<SidebarProps> = React.memo(({ projectID, onShowOnboarding }) => {
  const inlineContent = useStore(s => s.inlineContent)
  const slideOverContent = useStore(s => s.slideOverContent)
  const currentView = useStore(s => s.currentView)
  const inlineType = inlineContent?.type
  const slideOverType = slideOverContent?.type
  const activeType = slideOverType || inlineType

  const isActive = (id: string) => {
    if (id === 'Copilot') return !activeType && currentView === 'copilot'
    return ID_TO_INLINE_TYPE[id] === activeType
  }

  const handleClick = (item: { id: string; command: string }) => {
    if (!projectID) {
      onShowOnboarding()
      return
    }
    if (item.id === 'Copilot') {
      useStore.getState().setCurrentView('copilot')
      useStore.getState().setShowInlinePanel(false)
      useStore.getState().setSlideOverContent(null)
      return
    }
    
    const type = ID_TO_INLINE_TYPE[item.id]
    if (type) {
      useStore.getState().setCurrentView('copilot')
      useStore.getState().setShowInlinePanel(false)
      useStore.getState().setInlineContent(null)
      useStore.getState().setSlideOverContent({ type, title: PANEL_TITLES[item.id] || item.id })
    }
  }

  return (
    <div className="w-12 h-full flex flex-col items-center py-3 border-r border-border bg-surface shrink-0">
      <div className="mb-4 flex items-center justify-center w-8 h-8 rounded bg-primary/10 text-primary">
        <Binary size={18} />
      </div>

       <nav aria-label="Main navigation" className="flex-1 flex flex-col items-center gap-0.5 overflow-y-auto no-scrollbar">
        {sidebarItems.map((item, i) => {
          const Icon = item.icon
          const active = isActive(item.id)
          return (
            <React.Fragment key={item.id}>
              {DIVIDER_AFTER.has(i) && <div className="w-6 h-px bg-border my-2" />}
                 <button
                   onClick={() => handleClick(item)}
                   title={item.id}
                   aria-label={item.id}
                   data-testid={`sidebar-${item.id.toLowerCase().replace(/\s+/g, '-')}`}
                   aria-current={active ? 'page' : undefined}
                  className={`relative w-9 h-9 flex items-center justify-center rounded transition-colors focus:ring-2 focus:ring-primary ${
                    active
                      ? 'bg-primary/10 text-primary border-l-2 border-primary rounded-none pl-[calc(0.75rem-2px)]'
                      : 'text-textMuted hover:text-text hover:bg-surface-alt'
                  }`}
                >
                  <Icon size={18} />
                  <span className="sr-only">{item.id}</span>
                  <span className="absolute bottom-0.5 right-0.5 w-1.5 h-1.5 rounded-full bg-success" style={{ boxShadow: '0 0 4px rgba(34,197,94,0.5)' }} />
                </button>
            </React.Fragment>
          )
        })}
      </nav>

      <button
        onClick={onShowOnboarding}
        title={projectID || 'Select Project'}
        aria-label={projectID || 'Select Project'}
        className="w-9 h-9 flex items-center justify-center rounded text-textMuted hover:text-primary hover:bg-primary/10 transition-colors mt-2 focus:ring-2 focus:ring-primary"
      >
        <Settings size={16} />
        <span className="sr-only">{projectID || 'Select Project'}</span>
      </button>
    </div>
  )
});
