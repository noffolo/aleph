import React from 'react'
import { LayoutGrid, Binary, Activity, Bot, Eye, Book, Compass, Cpu, Database, Gauge, Package, Sliders, Monitor, Settings as SettingsIcon, Terminal, Wrench, Zap, Users } from 'lucide-react'
import { useStore } from '../store/useStore'
import type { SlideOverContent } from '../store/useStore'
import { isEnabled, FEATURE_COMPACT_SIDEBAR } from '../config/features'
import type { LucideIcon } from 'lucide-react'

interface SidebarProps {
  projectID: string
  onShowOnboarding: () => void
}

interface SidebarItem {
  id: string
  icon: LucideIcon
  scene: string | null
}

const SIDEBAR_ITEMS_ALL: SidebarItem[] = [
  { id: 'Dashboard', icon: Gauge, scene: 'terminal' },
  { id: 'Explorer', icon: LayoutGrid, scene: 'explore' },
  { id: 'Data Health', icon: Activity, scene: 'system' },
  { id: 'Copilot', icon: Bot, scene: null },
  { id: 'Oracle', icon: Eye, scene: 'system' },
  { id: 'Library', icon: Book, scene: 'explore' },
  { id: 'Ontologies', icon: Zap, scene: 'explore' },
  { id: 'Data Sources', icon: Database, scene: 'explore' },
  { id: 'Agents', icon: Users, scene: 'agents' },
  { id: 'Skills', icon: Cpu, scene: 'agents' },
  { id: 'Tools', icon: Wrench, scene: 'agents' },
  { id: 'Components', icon: Package, scene: 'agents' },
  { id: 'Settings', icon: Sliders, scene: 'system' },
]

const SIDEBAR_ITEMS_CORE: SidebarItem[] = [
  { id: 'Terminal', icon: Terminal, scene: 'terminal' },
  { id: 'Explore', icon: Compass, scene: 'explore' },
  { id: 'Agents', icon: Users, scene: 'agents' },
  { id: 'System', icon: Monitor, scene: 'system' },
  { id: 'Copilot', icon: Bot, scene: null },
]

const DIVIDER_SET_ALL = new Set([1, 4, 7])
const DIVIDER_SET_CORE = new Set<number>([])

export const Sidebar: React.FC<SidebarProps> = React.memo(({ projectID, onShowOnboarding }) => {
  const slideOverContent = useStore(s => s.slideOverContent)
  const currentScene = useStore(s => s.currentScene)
  const currentView = useStore(s => s.currentView)

  const isUxRedesign = isEnabled(FEATURE_COMPACT_SIDEBAR)
  const sidebarItems = isUxRedesign ? SIDEBAR_ITEMS_CORE : SIDEBAR_ITEMS_ALL
  const dividerSet = isUxRedesign ? DIVIDER_SET_CORE : DIVIDER_SET_ALL

  const isActive = (item: SidebarItem) => {
    if (item.id === 'Copilot') return currentScene === null && currentView === 'copilot'
    return currentScene === item.scene
  }

  const SIDEBAR_TO_SLIDEOVER: Record<string, string> = {
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
    'Dashboard': 'dashboard',
  }

  const handleClick = (item: SidebarItem) => {
    if (!projectID) {
      onShowOnboarding()
      return
    }
    if (item.scene === null) {
      useStore.getState().setCurrentScene(null)
      useStore.getState().setCurrentView('copilot')
      useStore.getState().setSlideOverContent(null)
      return
    }
    useStore.getState().setCurrentScene(item.scene)
    const slideType = SIDEBAR_TO_SLIDEOVER[item.id]
    if (slideType) {
      useStore.getState().setSlideOverContent({ type: slideType as SlideOverContent['type'], title: item.id })
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
          const active = isActive(item)
          return (
            <React.Fragment key={item.id}>
              {dividerSet.has(i) && <div className="w-6 h-px bg-border my-2" />}
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
        <SettingsIcon size={16} />
        <span className="sr-only">{projectID || 'Select Project'}</span>
      </button>
    </div>
  )
});
