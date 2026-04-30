import React from 'react';
import { useQueryState } from 'nuqs';
import { useStore } from '../store/useStore';
import { Terminal, Plus, Trash2, Activity, AlertCircle, CheckCircle2, Search, BarChart3 } from 'lucide-react';
import { t } from '../i18n';
import type { Tool } from '../store/types';

interface ToolManagementViewProps {
  inline?: boolean;
}

export const ToolManagementView: React.FC<ToolManagementViewProps> = ({ inline }) => {
  const store = useStore();
  const [searchQuery, setSearchQuery] = useQueryState('q', { defaultValue: '' });
  
  const tools = store.tools.filter(t => 
    t.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
    t.description?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const getStatusIcon = (status: 'healthy' | 'warning' | 'error' | 'unknown') => {
    const s = String(status);
    switch (s) {
      case 'healthy': return <CheckCircle2 size={14} className="text-success" />;
      case 'warning': return <AlertCircle size={14} className="text-warning" />;
      case 'error': return <AlertCircle size={14} className="text-danger" />;
      default: return <Activity size={14} className="text-textMuted" />;
    }
  };

  return (
    <div className={(inline ? '' : 'max-w-6xl mx-auto ') + 'p-6 space-y-6 h-full overflow-auto'}>
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Gestione Strumenti</h2>
          <p className="text-textMuted text-xs mt-1">Monitoraggio e manutenzione dei tool operativi.</p>
        </div>
        <div className="flex gap-2">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-textMuted" size={14} />
            <input 
              type="text" 
              placeholder={t('tools.search')} 
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9 pr-3 py-1.5 bg-background border border-border rounded-lg text-xs outline-none focus:border-primary/50 focus:ring-2 focus:ring-primary"
            />
          </div>
          <button 
            onClick={() => useStore.getState().setSlideOverContent({ type: 'tool-intelligence', title: 'Tool Intelligence' })}
            className="flex items-center gap-2 px-3 py-1.5 bg-primary/10 text-primary border border-primary/20 rounded-lg text-xs font-bold hover:bg-primary/20 transition-colors focus:ring-2 focus:ring-primary"
          >
            <BarChart3 size={14} />
            Intelligence
          </button>
        </div>

      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {tools.length > 0 ? tools.map(tool => (
          <div key={tool.id} className="bg-surface p-4 rounded-lg border border-border hover:border-primary/30 transition-all group">
            <div className="flex justify-between items-start mb-3">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-surface-alt rounded-lg flex items-center justify-center text-textMuted">
                  <Terminal size={16} />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-text">{tool.name}</h3>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="text-[10px] text-textDim font-mono">{tool.id}</span>
                    <div className="flex items-center gap-1 text-textDim">
                      {getStatusIcon((tool.healthStatus as 'healthy' | 'warning' | 'error' | 'unknown') || 'unknown')}
                      <span className="text-[10px] uppercase tracking-tighter">{String(tool.healthStatus || 'Sconosciuto')}</span>
                    </div>
                  </div>
                </div>
              </div>
              <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button className="p-1.5 hover:bg-border rounded text-textMuted hover:text-primary transition-colors focus:ring-2 focus:ring-primary" aria-label="Check tool health">
                  <Activity size={14} />
                </button>
                <button className="p-1.5 hover:bg-border rounded text-textMuted hover:text-danger transition-colors focus:ring-2 focus:ring-primary" aria-label="Delete tool">
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
            
            <p className="text-xs text-textMuted mb-4 line-clamp-2">{tool.description || 'Nessuna descrizione fornita.'}</p>
            
            <div className="grid grid-cols-2 gap-2 mb-4">
              <div className="bg-background p-2 rounded border border-border">
                <div className="text-[9px] font-bold text-textDim uppercase mb-1">Versione</div>
                <div className="text-[11px] font-mono">{String(tool.version || '1.0.0')}</div>
              </div>
              <div className="bg-background p-2 rounded border border-border">
                <div className="text-[9px] font-bold text-textDim uppercase mb-1">Categoria</div>
                <div className="text-[11px] font-mono">{String(tool.category || 'generale')}</div>
              </div>
            </div>
            
            <div className="flex justify-between items-center">
               <div className="text-[10px] text-textDim">
                 Ultimo check: {tool.lastCheckedAt ? new Date(tool.lastCheckedAt as string | number | Date).toLocaleString() : 'Mai'}
               </div>
               <button 
                onClick={() => useStore.getState().setSlideOverContent({ type: 'tool', title: 'Dettagli Tool', data: tool })}
                className="text-xs font-bold text-primary hover:underline"
               >
                 Dettagli →
               </button>
            </div>
          </div>
        )) : (
          <div className="col-span-full py-12 text-center text-textDim text-sm">
            Nessun strumento trovato.
          </div>
        )}
      </div>
    </div>
  );
};
