import React, { useState, useEffect, useRef } from 'react';
import { t } from '../i18n';
import { Command, ArrowRight, Navigation, Zap, Settings, Database } from 'lucide-react';
import { SLASH_COMMANDS, executeCommand, getTabCompletion, type SlashCommand } from './terminal/slashCommands';

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
  availableObjects: string[];
  projects: any[];
  onSelectProject: (id: string) => void;
  onSelectObject: (name: string) => void;
}

interface PaletteItem {
  type: 'command' | 'object' | 'project';
  key: string;
  label: string;
  description?: string;
  id: string;
  section: 'navigate' | 'actions' | 'system';
}

const NAVIGATE_COMMAND_NAMES = new Set([
  '/explore', '/agent', '/ontology', '/data', '/predict',
  '/library', '/health', '/skills', '/tools', '/components', '/settings',
])

const ACTIONS_COMMAND_NAMES = new Set([
  '/tool install', '/tool list', '/tool health', '/tool health-all', '/tool diagnose',
])

const SYSTEM_COMMAND_NAMES = new Set([
  '/help', '/model', '/clear',
])

function categorizeCommand(c: SlashCommand): PaletteItem['section'] {
  if (NAVIGATE_COMMAND_NAMES.has(c.name)) return 'navigate'
  if (ACTIONS_COMMAND_NAMES.has(c.name)) return 'actions'
  if (SYSTEM_COMMAND_NAMES.has(c.name)) return 'system'
  return 'navigate'
}

function renderCommandItem(
  c: SlashCommand,
  idx: number,
  selectedIndex: number,
  onExecute: () => void,
) {
  return (
    <button
      key={c.name}
      data-idx={idx}
      onClick={onExecute}
      className={`w-full flex items-center justify-between p-4 rounded-lg transition-colors group ${selectedIndex === idx ? 'bg-primary/10' : 'hover:bg-primary/10'}`}
    >
      <div className={`flex items-center space-x-3 font-bold ${selectedIndex === idx ? 'text-primary' : 'text-textMuted group-hover:text-primary'}`}>
        <Navigation size={18} />
        <span className="flex items-center space-x-2">
          <span>{c.name}</span>
          <span className="text-[10px] font-normal text-textDim opacity-50 group-hover:opacity-100 transition-opacity">{c.description}</span>
        </span>
      </div>
      <ArrowRight size={16} className={selectedIndex === idx ? 'text-primary/50' : 'text-textDim group-hover:text-primary/50'} />
    </button>
  )
}

export const CommandPalette: React.FC<CommandPaletteProps> = ({
  isOpen, onClose, availableObjects, projects, onSelectProject, onSelectObject
}) => {
  const [search, setSearch] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setSearch('');
    setSelectedIndex(-1);
  }, [isOpen]);

  const allCommands = SLASH_COMMANDS;

  const filteredCommands = allCommands.filter(c =>
    c.name.toLowerCase().includes(search.toLowerCase()) ||
    c.description.toLowerCase().includes(search.toLowerCase())
  );

  const filteredObjects = availableObjects.filter(o => o.toLowerCase().includes(search.toLowerCase()));
  const filteredProjects = projects.filter(p => p.name.toLowerCase().includes(search.toLowerCase()));

  const sections: { section: PaletteItem['section']; label: string; cmds: SlashCommand[] }[] = [
    {
      section: 'navigate' as const,
      label: t('commandPalette.section.navigate'),
      cmds: filteredCommands.filter(c => categorizeCommand(c) === 'navigate'),
    },
    {
      section: 'actions' as const,
      label: t('commandPalette.section.actions'),
      cmds: filteredCommands.filter(c => categorizeCommand(c) === 'actions'),
    },
    {
      section: 'system' as const,
      label: t('commandPalette.section.system'),
      cmds: filteredCommands.filter(c => categorizeCommand(c) === 'system'),
    },
  ].filter(s => s.cmds.length > 0);

  const totalCommandCount = sections.reduce((sum, s) => sum + s.cmds.length, 0);
  const objectBaseIdx = totalCommandCount;
  const projectBaseIdx = totalCommandCount + filteredObjects.length;

  const sectionIcons: Record<PaletteItem['section'], React.ReactNode> = {
    navigate: <Navigation size={16} className="text-primary" />,
    actions: <Zap size={16} className="text-warning" />,
    system: <Settings size={16} className="text-textMuted" />,
  }

  const executeSelected = () => {
    if (selectedIndex < 0) return;
    if (selectedIndex < totalCommandCount) {
      let cursor = 0;
      for (const s of sections) {
        if (selectedIndex < cursor + s.cmds.length) {
          const cmd = s.cmds[selectedIndex - cursor];
          executeCommand(cmd.name);
          onClose();
          return;
        }
        cursor += s.cmds.length;
      }
    }
    const objIdx = selectedIndex - objectBaseIdx;
    if (objIdx >= 0 && objIdx < filteredObjects.length) {
      onSelectObject(filteredObjects[objIdx]);
      onClose();
      return;
    }
    const projIdx = selectedIndex - projectBaseIdx;
    if (projIdx >= 0 && projIdx < filteredProjects.length) {
      onSelectProject(filteredProjects[projIdx].id);
      onClose();
    }
  };

  const totalItems = totalCommandCount + filteredObjects.length + filteredProjects.length;

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') { onClose(); return; }
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIndex(i => Math.min(i + 1, totalItems - 1)); return; }
    if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIndex(i => Math.max(i - 1, 0)); return; }
    if (e.key === 'Enter') { e.preventDefault(); executeSelected(); return; }
    if (e.key === 'Tab') {
      e.preventDefault();
      if (search.startsWith('/')) {
        const completions = getTabCompletion(search);
        if (completions.length > 0) {
          const currentC = completions[0];
          const nextC = completions[1] || completions[0];
          if (search === currentC) {
            setSearch(nextC);
          } else {
            setSearch(currentC);
          }
          setSelectedIndex(0);
        }
      }
      return;
    }
  };

  useEffect(() => {
    if (listRef.current && selectedIndex >= 0) {
      const el = listRef.current.querySelector(`[data-idx="${selectedIndex}"]`) as HTMLElement;
      el?.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  if (!isOpen) return null;

  return (
     <div role="dialog" aria-modal="true" aria-label="Command palette" className="fixed inset-0 bg-black/60 backdrop-blur-md z-[200] flex items-start justify-center pt-[15vh] p-4 animate-in fade-in duration-200" onClick={onClose} onKeyDown={handleKeyDown}>
      <div
        className="bg-surface w-full max-w-2xl rounded-lg overflow-hidden border border-border animate-in zoom-in-95 duration-200"
        onClick={e => e.stopPropagation()}
      >
        <div className="p-6 border-b flex items-center space-x-4 bg-surface-alt/50">
           <Command size={24} className="text-primary" />
            <input
               autoFocus
               aria-label="Search commands"
               value={search}
              onChange={e => { setSearch(e.target.value); setSelectedIndex(0); }}
              placeholder={t('commandPalette.search')}
              className="flex-1 bg-transparent border-none outline-none text-xl font-medium text-text placeholder:text-textDim"
           />
           <div className="px-2 py-1 bg-surface rounded-lg border border-border text-[10px] font-bold text-textMuted">ESC</div>
        </div>

        <div className="max-h-[60vh] overflow-auto p-4 custom-scrollbar" ref={listRef}>
            {search && sections.map((section) => (
              <div key={section.section} className="mb-6">
                 <div className="px-4 mb-2 text-[10px] font-bold text-textMuted uppercase tracking-widest flex items-center gap-1">
                   {sectionIcons[section.section]}
                   {section.label}
                 </div>
                 <div className="space-y-1">
                    {section.cmds.map((c, cIdx) => {
                      let baseIdx = 0;
                      for (const s of sections) {
                        if (s.section === section.section) break;
                        baseIdx += s.cmds.length;
                      }
                      const idx = baseIdx + cIdx;
                      return renderCommandItem(c, idx, selectedIndex, () => { executeCommand(c.name); onClose(); });
                    })}
                 </div>
              </div>
            ))}

            {search && filteredObjects.length > 0 && (
              <div className="mb-6">
                 <div className="px-4 mb-2 text-[10px] font-bold text-textMuted uppercase tracking-widest flex items-center gap-1">
                   <Navigation size={16} className="text-primary" />
                   {t('commandPalette.section.navigate')}
                 </div>
                 <div className="space-y-1">
                    {filteredObjects.map((o, oIdx) => {
                      const idx = objectBaseIdx + oIdx;
                      return (
                        <button
                          key={o}
                          data-idx={idx}
                          onClick={() => { onSelectObject(o); onClose(); }}
                          className={`w-full flex items-center justify-between p-4 rounded-lg transition-colors group ${selectedIndex === idx ? 'bg-primary/10' : 'hover:bg-primary/10'}`}
                        >
                         <div className={`flex items-center space-x-3 font-bold ${selectedIndex === idx ? 'text-primary' : 'text-textMuted group-hover:text-primary'}`}>
                            <Database size={18} />
                            <span>{o}</span>
                         </div>
                         <ArrowRight size={16} className={selectedIndex === idx ? 'text-primary/50' : 'text-textDim group-hover:text-primary/50'} />
                        </button>
                      );
                    })}
                 </div>
              </div>
            )}

            {search && filteredProjects.length > 0 && (
              <div className="mb-6">
                  <div className="px-4 mb-2 text-[10px] font-bold text-textMuted uppercase tracking-widest flex items-center gap-1">
                    <Zap size={16} className="text-warning" />
                    {t('commandPalette.section.system')}
                  </div>
                 <div className="space-y-1">
                    {filteredProjects.map((p, pIdx) => {
                      const idx = projectBaseIdx + pIdx;
                      return (
                        <button
                          key={p.id}
                          data-idx={idx}
                          onClick={() => { onSelectProject(p.id); onClose(); }}
                          className={`w-full flex items-center justify-between p-4 rounded-lg transition-colors group ${selectedIndex === idx ? 'bg-warning/10' : 'hover:bg-warning/10'}`}
                        >
                           <div className={`flex items-center space-x-3 font-bold ${selectedIndex === idx ? 'text-warning' : 'text-textMuted group-hover:text-warning'}`}>
                              <Zap size={18} />
                              <span>{p.name}</span>
                           </div>
                           <ArrowRight size={16} className={selectedIndex === idx ? 'text-warning/50' : 'text-textDim group-hover:text-warning/50'} />
                        </button>
                      );
                    })}
                 </div>
              </div>
            )}

           {!search && (
             <div className="text-center py-20">
                <Command size={48} className="mx-auto text-textDim mb-4" />
                <p className="text-textMuted font-bold text-sm">{t('commandPalette.prompt')}</p>
             </div>
           )}
        </div>
      </div>
    </div>
  );
};
