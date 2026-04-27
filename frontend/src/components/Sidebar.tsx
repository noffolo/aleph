import React from 'react'
import { LayoutGrid, Binary, Activity, Bot, Eye, Book, Zap, Database, Users, Cpu, Wrench, Package, Sliders, Settings } from 'lucide-react'
import { useStore } from '../store/useStore'

interface SidebarProps {
  projectID: string
  onShowOnboarding: () => void
}

const ID_TO_INLINE_TYPE: Record<string, string> = {
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

export const Sidebar: React.FC<SidebarProps> = ({ projectID, onShowOnboarding }) => {
  const store = useStore()
  const inlineType = store.inlineContent?.type
  const slideOverType = store.slideOverContent?.type
  const activeType = slideOverType || inlineType

  const isActive = (id: string) => {
    if (id === 'Copilot') return !activeType && store.currentView === 'copilot'
    return ID_TO_INLINE_TYPE[id] === activeType
  }

  const handleClick = (item: { id: string; command: string }) => {
    if (!store.projectID) {
      onShowOnboarding()
      return
    }
    if (item.command) {
      store.setInput(item.command)
      store.setCurrentView('copilot')
      store.setShowInlinePanel(false)
      setTimeout(() => {
        const promptEl = document.querySelector('textarea[data-terminal-prompt="true"]') as HTMLTextAreaElement | null
        if (promptEl) {
          const form = promptEl.closest('form')
          form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
        }
      }, 0)
    } else {
      store.setCurrentView('copilot')
      store.setShowInlinePanel(false)
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
                 aria-current={active ? 'page' : undefined}
                 className={`w-9 h-9 flex items-center justify-center rounded transition-colors ${
                   active
                     ? 'bg-primary/10 text-primary'
                     : 'text-textMuted hover:text-text hover:bg-surface-alt'
                 }`}
               >
                <Icon size={18} />
              </button>
            </React.Fragment>
          )
        })}
      </nav>

      <button
        onClick={onShowOnboarding}
        title={projectID || 'Select Project'}
        className="w-9 h-9 flex items-center justify-center rounded text-textMuted hover:text-primary hover:bg-primary/10 transition-colors mt-2"
      >
        <Settings size={16} />
      </button>
    </div>
  )
}