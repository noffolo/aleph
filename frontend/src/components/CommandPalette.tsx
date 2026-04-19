import React, { useState } from 'react';
import { Search, Command, FileText, Bot, Database, Zap, ArrowRight } from 'lucide-react';

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
  availableObjects: string[];
  projects: any[];
  onSelectProject: (id: string) => void;
  onSelectObject: (name: string) => void;
  setActiveTab: (tab: string) => void;
}

export const CommandPalette: React.FC<CommandPaletteProps> = ({
  isOpen, onClose, availableObjects, projects, onSelectProject, onSelectObject, setActiveTab
}) => {
  const [search, setSearch] = useState('');

  if (!isOpen) return null;

  const filteredObjects = availableObjects.filter(o => o.toLowerCase().includes(search.toLowerCase()));
  const filteredProjects = projects.filter(p => p.name.toLowerCase().includes(search.toLowerCase()));

  return (
    <div className="fixed inset-0 bg-gray-900/40 backdrop-blur-md z-[200] flex items-start justify-center pt-[15vh] p-4 animate-in fade-in duration-200" onClick={onClose}>
      <div 
        className="bg-white w-full max-w-2xl rounded-[32px] shadow-2xl overflow-hidden border border-gray-100 animate-in zoom-in-95 duration-200"
        onClick={e => e.stopPropagation()}
      >
        <div className="p-6 border-b flex items-center space-x-4 bg-gray-50/50">
           <Command size={24} className="text-blue-600" />
           <input 
              autoFocus
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Cerca entità, progetti o comandi (CMD+K)..."
              className="flex-1 bg-transparent border-none outline-none text-xl font-medium text-gray-900 placeholder:text-gray-300"
           />
           <div className="px-2 py-1 bg-white rounded-lg border border-gray-200 text-[10px] font-bold text-gray-400">ESC</div>
        </div>

        <div className="max-h-[60vh] overflow-auto p-4 custom-scrollbar">
           {search && filteredObjects.length > 0 && (
             <div className="mb-6">
                <div className="px-4 mb-2 text-[10px] font-bold text-gray-400 uppercase tracking-widest">Entità Ontologiche</div>
                <div className="space-y-1">
                   {filteredObjects.map(o => (
                     <button key={o} onClick={() => { onSelectObject(o); setActiveTab('Explorer'); onClose(); }} className="w-full flex items-center justify-between p-4 hover:bg-blue-50 rounded-2xl transition-colors group">
                        <div className="flex items-center space-x-3 text-gray-700 group-hover:text-blue-700 font-bold">
                           <Database size={18} />
                           <span>{o}</span>
                        </div>
                        <ArrowRight size={16} className="text-gray-200 group-hover:text-blue-300" />
                     </button>
                   ))}
                </div>
             </div>
           )}

           {search && filteredProjects.length > 0 && (
             <div className="mb-6">
                <div className="px-4 mb-2 text-[10px] font-bold text-gray-400 uppercase tracking-widest">Workspaces</div>
                <div className="space-y-1">
                   {filteredProjects.map(p => (
                     <button key={p.id} onClick={() => { onSelectProject(p.id); onClose(); }} className="w-full flex items-center justify-between p-4 hover:bg-amber-50 rounded-2xl transition-colors group">
                        <div className="flex items-center space-x-3 text-gray-700 group-hover:text-amber-700 font-bold">
                           <Zap size={18} />
                           <span>{p.name}</span>
                        </div>
                        <ArrowRight size={16} className="text-gray-200 group-hover:text-amber-300" />
                     </button>
                   ))}
                </div>
             </div>
           )}

           {!search && (
             <div className="text-center py-20">
                <Command size={48} className="mx-auto text-gray-100 mb-4" />
                <p className="text-gray-400 font-bold text-sm">Digita per navigare istantaneamente in Aleph</p>
             </div>
           )}
        </div>
      </div>
    </div>
  );
};
