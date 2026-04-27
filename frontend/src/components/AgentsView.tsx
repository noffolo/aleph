import React, { useCallback } from 'react';
import { Bot, Plus, Trash2, Pencil, Globe, Server } from 'lucide-react';
import { useStore } from '../store/useStore';
import { useCursorPagination } from '../hooks/useCursorPagination';
import { agentClient } from '../api/factory';
import { ListAgentsRequest } from '../api/proto/aleph/v1/query_pb';
import { useQueryState } from 'nuqs';
import { t } from '../i18n';

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

export const AgentsView: React.FC<AgentsViewProps> = React.memo(({ agents: initialAgents, onCreateAgent, onDeleteAgent, onUpdateAgent, ollamaHealthy = false, ollamaModels = [], inline = false }) => {
  const setAgents = useStore(state => state.setAgents);
  const projectId = useStore(state => state.selectedObject);
  const [searchQuery, setSearchQuery] = useQueryState('q', { defaultValue: '' });

  const { items: agents, hasMore, loadMore, loading } = useCursorPagination({
    clientMethod: agentClient.listAgents,
    requestBuilder: useCallback((cursor: string) => new ListAgentsRequest({ projectId, after: cursor, limit: 25 }), [projectId]),
    responseExtractor: (res) => ({ items: res.agents, nextCursor: res.nextCursor }),
    storeSetter: setAgents,
    initialItems: initialAgents,
  });

  const filteredAgents = agents.filter(a => 
    !searchQuery || a.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const openCreate = useCallback(() => {
    useStore.getState().setSlideOverContent({ type: 'agent-form', title: t('agents.create'), data: undefined });
  }, []);

  const openEdit = useCallback((a: Agent) => {
    useStore.getState().setSlideOverContent({ type: 'agent-form', title: t('agents.edit'), data: a as any });
  }, []);

  return (
    <div 
      className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-8'} 
      role="region" 
      aria-label="Agents"
    >
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">{t('agents.title')}</h2>
          <p className="text-textMuted text-sm mt-1">{t('agents.subtitle')}</p>
        </div>
          <button 
            onClick={openCreate} 
            className="flex items-center space-x-2 bg-primary text-background px-6 py-3 rounded-lg font-bold hover:bg-primary/90 transition-all shadow-lg "
            aria-label="Create new agent"
          >
          <Plus size={20} />
          <span>{t('agents.create')}</span>
        </button>
       </div>

       <input
         type="text"
         value={searchQuery}
         onChange={e => setSearchQuery(e.target.value)}
         placeholder={t('agents.search')}
         className="w-full max-w-md px-4 py-2 bg-surface-alt border border-border rounded-lg text-sm font-mono text-textPrimary placeholder-textDim focus:outline-none focus:border-primary/50"
       />

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
         {filteredAgents.map(a => (
           <div key={a.id} className="bg-surface p-6 rounded-lg border border-border shadow-sm hover:shadow-lg transition-all group relative">

              <div className="absolute top-6 right-6 flex items-center space-x-1">
                 <button 
                   onClick={(e) => { e.stopPropagation(); openEdit(a); }} 
                   className="p-2 text-textDim hover:text-primary hover:bg-primary/10 rounded-xl transition-all opacity-0 group-hover:opacity-100"
                   aria-label="Edit agent"
                 ><Pencil size={18} /></button>
                 <button 
                   onClick={(e) => { e.stopPropagation(); if (confirm('Sei sicuro di voler eliminare questo agente?')) onDeleteAgent(a.id); }} 
                   className="p-2 text-textDim hover:text-danger hover:bg-danger/10 rounded-xl transition-all opacity-0 group-hover:opacity-100"
                   aria-label="Delete agent"
                 ><Trash2 size={18} /></button>
              </div>
              <div className="w-12 h-12 bg-primary/10 rounded-lg flex items-center justify-center text-primary mb-4 group-hover:bg-primary group-hover:text-white transition-colors"><Bot size={24} /></div>
               <h3 className="text-xl font-bold mb-1">{a.name}</h3>
               <div className="flex items-center flex-wrap gap-2 mb-4">
                 <div className="inline-block px-2 py-1 bg-surface-alt rounded-md text-[10px] font-mono font-bold text-textMuted">{a.model}</div>
                 {a.provider && <div className="inline-block px-2 py-1 bg-primary/10 rounded-md text-[10px] font-mono font-bold text-primary uppercase">{a.provider}</div>}
                 {a.baseUrl && <div className="inline-block px-2 py-1 bg-primary/10 rounded-md text-[10px] font-mono font-bold text-primary truncate max-w-[160px]">{a.baseUrl}</div>}
               </div>
              <p className="text-sm text-textMuted line-clamp-4 mb-6 leading-relaxed">{a.systemPrompt || t('agents.noSystemPrompt')}</p>
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

      {hasMore && (
        <div className="flex justify-center">
          <button 
            onClick={loadMore} 
            disabled={loading}
            className="rounded-lg border border-border px-4 py-2 text-sm text-textMuted hover:text-textPrimary hover:border-textMuted transition-colors disabled:opacity-50"
          >
            {loading ? 'Caricamento...' : 'Carica Altri'}
          </button>
        </div>
      )}
    </div>
  );
});