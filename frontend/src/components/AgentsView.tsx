import React from 'react';
import { Bot, Plus, Trash2, Pencil, Globe, Server } from 'lucide-react';
import { useStore } from '../store/useStore';

interface Agent {
  id: string;
  name: string;
  model: string;
  systemPrompt: string;
  provider?: string;
  apiKey?: string;
  baseUrl?: string;
  skillIds?: string[];
}

interface AgentsViewProps {
  agents: Agent[];
  onCreateAgent: (name: string, model: string, systemPrompt: string, provider: string, apiKey: string, baseUrl: string) => void;
  onDeleteAgent: (id: string) => void;
  onUpdateAgent: (agent: Agent) => void;
  ollamaHealthy?: boolean;
  ollamaModels?: string[];
  inline?: boolean;
}

export const AgentsView: React.FC<AgentsViewProps> = ({ agents, onCreateAgent, onDeleteAgent, onUpdateAgent, ollamaHealthy = false, ollamaModels = [], inline = false }) => {
  const openCreate = () => {
    useStore.getState().setSlideOverContent({ type: 'agent-form', title: 'Nuovo Agente', data: undefined });
  };

  const openEdit = (a: Agent) => {
    useStore.getState().setSlideOverContent({ type: 'agent-form', title: 'Modifica Agente', data: a as any });
  };

  return (
    <div className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-8'}>
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Gestore Agenti</h2>
          <p className="text-textMuted text-sm mt-1">Configura agenti AI con qualsiasi provider — locale o cloud.</p>
        </div>
         <button onClick={openCreate} className="flex items-center space-x-2 bg-primary text-background px-6 py-3 rounded-lg font-bold hover:bg-primary/90 transition-all shadow-lg ">
           <Plus size={20} />
           <span>Nuovo Agente</span>
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {agents.map(a => (
          <div key={a.id} className="bg-surface p-6 rounded-lg border border-border shadow-sm hover:shadow-lg transition-all group relative">
             <div className="absolute top-6 right-6 flex items-center space-x-1">
               <button onClick={(e) => { e.stopPropagation(); openEdit(a); }} className="p-2 text-textDim hover:text-primary hover:bg-primary/10 rounded-xl transition-all opacity-0 group-hover:opacity-100"><Pencil size={18} /></button>
               <button onClick={(e) => { e.stopPropagation(); if (confirm('Sei sicuro di voler eliminare questo agente?')) onDeleteAgent(a.id); }} className="p-2 text-textDim hover:text-danger hover:bg-danger/10 rounded-xl transition-all opacity-0 group-hover:opacity-100"><Trash2 size={18} /></button>
             </div>
             <div className="w-12 h-12 bg-primary/10 rounded-lg flex items-center justify-center text-primary mb-4 group-hover:bg-primary group-hover:text-white transition-colors"><Bot size={24} /></div>
              <h3 className="text-xl font-bold mb-1">{a.name}</h3>
              <div className="flex items-center flex-wrap gap-2 mb-4">
                <div className="inline-block px-2 py-1 bg-surface-alt rounded-md text-[10px] font-mono font-bold text-textMuted">{a.model}</div>
                {a.provider && <div className="inline-block px-2 py-1 bg-primary/10 rounded-md text-[10px] font-mono font-bold text-primary uppercase">{a.provider}</div>}
                {a.baseUrl && <div className="inline-block px-2 py-1 bg-primary/10 rounded-md text-[10px] font-mono font-bold text-primary truncate max-w-[160px]">{a.baseUrl}</div>}
              </div>
             <p className="text-sm text-textMuted line-clamp-4 mb-6 leading-relaxed">{a.systemPrompt || "Nessun prompt di sistema configurato."}</p>
              <div className="flex items-center space-x-2 border-t pt-4 border-border">
                 <div className={`h-2 w-2 rounded-full ${ollamaHealthy ? 'bg-success' : 'bg-textDim'}`}></div>
                 <span className="text-[10px] font-bold text-textMuted uppercase tracking-widest">{ollamaHealthy ? 'Servizio Attivo' : 'Offline'}</span>
              </div>
          </div>
        ))}
        {agents.length === 0 && (
          <div className="col-span-full py-20 bg-surface-alt border-2 border-dashed border-border rounded-lg text-center">
             <Bot size={48} className="mx-auto text-textDim mb-4" />
              <p className="text-textMuted font-medium font-mono uppercase text-xs tracking-widest">Nessun agente configurato</p>
          </div>
        )}
      </div>
    </div>
  );
};