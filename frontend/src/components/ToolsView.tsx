import React from 'react';
import { Terminal, Plus, Code, Trash2, Play } from 'lucide-react';
import { useStore } from '../store/useStore';
import { useCursorPagination } from '../hooks/useCursorPagination';
import { toolClient } from '../api/factory';
import { ListToolsRequest } from '../api/proto/aleph/v1/query_pb';

interface Tool {
  id: string;
  name: string;
  description: string;
  code: string;
}

interface ToolsViewProps {
  tools: Tool[];
  onCreateTool: (name: string, description: string, code: string) => void;
  onEditTool: (tool: Tool) => void;
  onDeleteTool: (id: string) => void;
  onExecuteTool: (id: string) => void;
  inline?: boolean;
}

export const ToolsView: React.FC<ToolsViewProps> = ({ tools: initialTools, onCreateTool, onEditTool, onDeleteTool, onExecuteTool, inline = false }) => {
  const setTools = useStore(state => state.setTools);
  const projectId = useStore(state => state.selectedObject);

  const { items: tools, hasMore, loadMore, loading } = useCursorPagination({
    clientMethod: toolClient.listTools,
    requestBuilder: (cursor) => new ListToolsRequest({ projectId, after: cursor, limit: 25 }),
    responseExtractor: (res) => ({ items: res.tools, nextCursor: res.nextCursor }),
    storeSetter: setTools,
    initialItems: initialTools,
  });

  const openCreate = () => {
    useStore.getState().setSlideOverContent({ type: 'tool-form', title: 'Nuovo Strumento', data: undefined });
  };

  return (
    <div className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-8'}>
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Strumenti Operativi</h2>
          <p className="text-textMuted text-sm mt-1">Definisci funzioni eseguibili dagli agenti (SQL, Python, API).</p>
        </div>
        <button onClick={openCreate} className="flex items-center space-x-2 bg-primary text-background px-6 py-3 rounded-2xl font-bold hover:bg-primary/90 transition-all shadow-lg">
           <Plus size={20} />
           <span>Crea Strumento</span>
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {tools.map(t => (
          <div key={t.id} className="bg-surface p-8 rounded-lg border border-border shadow-sm hover:shadow-lg shadow-primary/5 transition-all flex flex-col h-full group relative">
              <button 
                 onClick={(e) => { e.stopPropagation(); if (confirm('Eliminare questo strumento?')) onDeleteTool(t.id); }}
                 className="absolute top-8 right-8 p-2 text-textDim hover:text-danger hover:bg-danger/10 rounded-lg transition-all opacity-0 group-hover:opacity-100"
              >
                 <Trash2 size={16} />
              </button>
              <div className="flex items-center space-x-3 mb-4">
                 <div className="w-10 h-10 bg-surface-alt rounded-xl flex items-center justify-center text-textMuted"><Terminal size={20} /></div>
                 <h3 className="text-xl font-bold">{t.name}</h3>
              </div>
              <p className="text-sm text-textMuted mb-6 flex-1">{t.description}</p>
              <div className="bg-surface-alt p-4 rounded-lg border border-border mb-6">
                 <div className="flex items-center space-x-2 mb-2"><Code size={12} className="text-textMuted" /><span className="text-[10px] font-bold text-textMuted uppercase tracking-widest">Anteprima Codice</span></div>
                 <pre className="text-[10px] font-mono text-textMuted line-clamp-3 overflow-hidden">{t.code}</pre>
              </div>
               <div className="flex space-x-2">
                 <button onClick={() => onExecuteTool(t.id)} className="flex-1 py-3 bg-primary text-background rounded-lg text-xs font-bold hover:bg-primary/90 transition-colors uppercase tracking-widest flex items-center justify-center space-x-1">
                   <Play size={12} />
                   <span>Esegui</span>
                 </button>
                 <button onClick={() => onEditTool(t)} className="flex-1 py-3 bg-surface-alt text-textMuted rounded-lg text-xs font-bold hover:bg-border transition-colors uppercase tracking-widest">Dettagli</button>
               </div>
           </div>
        ))}
        {tools.length === 0 && (
          <div className="col-span-full py-20 bg-surface-alt border-2 border-dashed border-border rounded-lg text-center">
            <Terminal size={48} className="mx-auto text-textDim mb-4" />
            <p className="text-textMuted font-medium font-mono uppercase text-xs tracking-widest">Nessun strumento configurato</p>
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
};