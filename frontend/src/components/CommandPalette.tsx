import React, { useState, useEffect, useRef } from 'react';
import { Command, Database, Zap, ArrowRight } from 'lucide-react';

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
  availableObjects: string[];
  projects: any[];
  onSelectProject: (id: string) => void;
  onSelectObject: (name: string) => void;
}

interface PaletteItem {
  type: 'object' | 'project';
  key: string;
  label: string;
  id: string;
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

  if (!isOpen) return null;

  const filteredObjects = availableObjects.filter(o => o.toLowerCase().includes(search.toLowerCase()));
  const filteredProjects = projects.filter(p => p.name.toLowerCase().includes(search.toLowerCase()));

  const items: PaletteItem[] = [
    ...filteredObjects.map(o => ({ type: 'object' as const, key: `obj-${o}`, label: o, id: o })),
    ...filteredProjects.map(p => ({ type: 'project' as const, key: `proj-${p.id}`, label: p.name, id: p.id })),
  ];

  const executeSelected = () => {
    if (selectedIndex < 0 || selectedIndex >= items.length) return;
    const item = items[selectedIndex];
    if (item.type === 'object') {
      onSelectObject(item.id);
    } else {
      onSelectProject(item.id);
    }
    onClose();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') { onClose(); return; }
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIndex(i => Math.min(i + 1, items.length - 1)); return; }
    if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIndex(i => Math.max(i - 1, 0)); return; }
    if (e.key === 'Enter') { e.preventDefault(); executeSelected(); return; }
  };

  useEffect(() => {
    if (listRef.current && selectedIndex >= 0) {
      const el = listRef.current.querySelector(`[data-idx="${selectedIndex}"]`) as HTMLElement;
      el?.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  return (
     <div className="fixed inset-0 bg-black/60 backdrop-blur-md z-[200] flex items-start justify-center pt-[15vh] p-4 animate-in fade-in duration-200" onClick={onClose} onKeyDown={handleKeyDown}>
      <div 
        className="bg-surface w-full max-w-2xl rounded-lg   overflow-hidden border border-border animate-in zoom-in-95 duration-200"
        onClick={e => e.stopPropagation()}
      >
        <div className="p-6 border-b flex items-center space-x-4 bg-surface-alt/50">
           <Command size={24} className="text-primary" />
           <input 
              autoFocus
              value={search}
              onChange={e => { setSearch(e.target.value); setSelectedIndex(0); }}
              placeholder="Cerca entità, progetti o comandi (CMD+K)..."
              className="flex-1 bg-transparent border-none outline-none text-xl font-medium text-text placeholder:text-textDim"
           />
           <div className="px-2 py-1 bg-surface rounded-lg border border-border text-[10px] font-bold text-textMuted">ESC</div>
        </div>

        <div className="max-h-[60vh] overflow-auto p-4 custom-scrollbar" ref={listRef}>
           {search && filteredObjects.length > 0 && (
             <div className="mb-6">
                <div className="px-4 mb-2 text-[10px] font-bold text-textMuted uppercase tracking-widest">Entità Ontologiche</div>
                <div className="space-y-1">
                   {filteredObjects.map((o, idx) => (
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
                   ))}
                </div>
             </div>
           )}

           {search && filteredProjects.length > 0 && (
             <div className="mb-6">
                 <div className="px-4 mb-2 text-[10px] font-bold text-textMuted uppercase tracking-widest">Spazi di lavoro</div>
                <div className="space-y-1">
                   {filteredProjects.map((p, pIdx) => {
                     const idx = filteredObjects.length + pIdx;
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
                <p className="text-textMuted font-bold text-sm">Digita per navigare istantaneamente in Aleph</p>
             </div>
           )}
        </div>
      </div>
    </div>
  );
};
