import React, { useCallback } from 'react';
import { Terminal, Plus, Code, Trash2, Play, Activity, X } from 'lucide-react';
import { useStore } from '../store/useStore';
import { useCursorPagination } from '../hooks/useCursorPagination';
import { toolClient } from '../api/factory';
import { ListToolsRequest } from '../api/proto/aleph/v1/query_pb';
import { useQueryState } from 'nuqs';
import { t } from '../i18n';
import { SkeletonLoader } from './SkeletonLoader';
import { InlineError } from './ui/InlineError';
import { ToolResultDisplay } from './ToolResultDisplay';
import { ToolHealthIndicator } from './ToolHealthIndicator';
import { ToolConfigPanel } from './ToolConfigPanel';
import { GlassPanel } from './ui/GlassPanel';

interface Tool {
  id: string;
  name: string;
  description: string;
  code: string;
  healthStatus?: 'healthy' | 'warning' | 'error' | 'unknown';
  lastCheckedAt?: string;
}

export interface ToolsViewProps {
  tools: Tool[];
  onCreateTool: (name: string, description: string, code: string) => void;
  onEditTool: (tool: Tool) => void;
  onDeleteTool: (id: string) => void;
  onExecuteTool: (id: string) => void;
  inline?: boolean;
  isLoading?: boolean;
  error?: string | null;
}

export const ToolsView: React.FC<ToolsViewProps> = React.memo(({ tools: initialTools, onCreateTool, onEditTool, onDeleteTool, onExecuteTool, inline = false, isLoading, error }) => {
  const setTools = useStore(state => state.setTools);
  const projectId = useStore(state => state.selectedObject);
  const expandedSections = useStore(s => s.expandedSections);
  const toggleSection = useStore(s => s.toggleSection);
  const [searchQuery, setSearchQuery] = useQueryState('q', { defaultValue: '' });
  const [debouncedQuery, setDebouncedQuery] = React.useState(searchQuery);

  React.useEffect(() => {
    const handler = setTimeout(() => setDebouncedQuery(searchQuery), 300);
    return () => clearTimeout(handler);
  }, [searchQuery]);

  const { items: tools, hasMore, loadMore, loading } = useCursorPagination({
    clientMethod: toolClient.listTools,
    requestBuilder: useCallback((cursor: string) => new ListToolsRequest({ projectId, after: cursor, limit: 25 }), [projectId]),
    responseExtractor: (res) => ({ items: (res.tools || []) as Tool[], nextCursor: res.nextCursor }),
    storeSetter: setTools as (items: Tool[]) => void,
    initialItems: initialTools as Tool[],
  });

  const filteredTools = tools.filter(t => 
    !debouncedQuery || 
    t.name.toLowerCase().includes(debouncedQuery.toLowerCase()) || 
    t.description.toLowerCase().includes(debouncedQuery.toLowerCase())
  );

  const [selectedToolId, setSelectedToolId] = React.useState<string | null>(null);
  const [executionResult, setExecutionResult] = React.useState<any>(null);

  const selectedTool = tools.find(t => t.id === selectedToolId);


  const openCreate = useCallback(() => {
    useStore.getState().setSlideOverContent({ type: 'tool-form', title: t('tools.create'), data: undefined });
  }, []);

  if (isLoading) return <SkeletonLoader />;
  if (error) return <div className="max-w-6xl mx-auto"><InlineError message={error} /></div>;

  return (
    <div 
      className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-8'} 
      role="region" 
      aria-label="Tools"
    >
      <GlassPanel
        header="Overview"
        icon={<Terminal size={16} />}
        sectionKey="tools.overview"
        expanded={expandedSections['tools.overview']}
        onToggle={() => toggleSection('tools.overview')}
      >
        <h2 className="text-3xl font-bold tracking-tight">{t('tools.title')}</h2>
        <p className="text-textMuted text-sm mt-1">{t('tools.subtitle')}</p>
      </GlassPanel>

      <div className="flex items-center gap-4">
        <input
          type="text"
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          placeholder={t('tools.search')}
          className="flex-1 max-w-md px-4 py-2 bg-surface-alt border border-border rounded-lg text-sm font-mono text-textPrimary placeholder-textDim focus:outline-none focus:border-primary/50 focus:ring-2 focus:ring-primary"
        />
        <button 
          onClick={openCreate} 
          className="flex items-center space-x-2 bg-primary text-background px-6 py-3 rounded-2xl font-bold hover:bg-primary/90 transition-all shadow-lg focus:ring-2 focus:ring-primary shrink-0"
          aria-label="Create new tool"
        >
          <Plus size={20} />
          <span>{t('tools.create')}</span>
        </button>
      </div>

      <GlassPanel
        header="Tools"
        icon={<Code size={16} />}
        sectionKey="tools.list"
        expanded={expandedSections['tools.list']}
        onToggle={() => toggleSection('tools.list')}
      >
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {filteredTools.map(t => (
            <div key={t.id} className="bg-surface p-8 rounded-lg border border-border shadow-sm hover:shadow-lg shadow-primary/5 transition-all flex flex-col h-full group relative">

                <button 
                   onClick={(e) => { e.stopPropagation(); if (confirm('Eliminare questo strumento?')) onDeleteTool(t.id); }}
                   className="absolute top-8 right-8 p-2 text-textDim hover:text-danger hover:bg-danger/10 rounded-lg transition-all opacity-0 group-hover:opacity-100 focus:ring-2 focus:ring-primary"
                   aria-label={`Delete tool ${t.name}`}
                >
                  <Trash2 size={16} />
               </button>
                <div className="flex items-center space-x-3 mb-4">
                   <div className="w-10 h-10 bg-surface-alt rounded-xl flex items-center justify-center text-textMuted"><Terminal size={20} /></div>
                   <div className="flex-1 min-w-0">
                      <h3 className="text-xl font-bold truncate">{t.name}</h3>
                       <div className="flex items-center space-x-2">
                         <ToolHealthIndicator status={t.healthStatus || 'unknown'} lastCheck={t.lastCheckedAt ? new Date(t.lastCheckedAt).toLocaleTimeString() : undefined} /> 
                         <span className="text-[10px] text-textDim font-mono uppercase tracking-tighter">
                           {t.healthStatus === 'healthy' ? 'Healthy' : t.healthStatus === 'warning' ? 'Degraded' : t.healthStatus === 'error' ? 'Down' : 'Unknown'}
                         </span>
                       </div>
                   </div>
                </div>

               <p className="text-sm text-textMuted mb-6 flex-1">{t.description}</p>
               <div className="bg-surface-alt p-4 rounded-lg border border-border mb-6">
                  <div className="flex items-center space-x-2 mb-2"><Code size={12} className="text-textMuted" /><span className="text-[10px] font-bold text-textMuted uppercase tracking-widest">Anteprima Codice</span></div>
                  <pre className="text-[10px] font-mono text-textMuted line-clamp-3 overflow-hidden">{t.code}</pre>
               </div>
                <div className="flex space-x-2">
                  <button 
                        onClick={() => {
                          setSelectedToolId(selectedToolId === t.id ? null : t.id);
                          onExecuteTool(t.id);
                        }} 
                        className="flex-1 py-3 bg-primary text-background rounded-lg text-xs font-bold hover:bg-primary/90 transition-colors uppercase tracking-widest flex items-center justify-center space-x-1 focus:ring-2 focus:ring-primary"
                        aria-label={`Execute tool ${t.name}`}
                      >
                       <Play size={12} />
                       <span>Esegui</span>
                      </button>
                     <button 
                       onClick={() => setSelectedToolId(selectedToolId === t.id ? null : t.id)} 
                       className="flex-1 py-3 bg-surface-alt text-textMuted rounded-lg text-xs font-bold hover:bg-border transition-colors uppercase tracking-widest"
                       aria-label="View tool details"
                     >Dettagli</button>
                </div>

             </div>
          ))}
        </div>
      </GlassPanel>

      {tools.length === 0 && (
       <div className="py-20 bg-surface-alt border-2 border-dashed border-border rounded-lg text-center">
         <Terminal size={48} className="mx-auto text-textDim mb-4" />
         <p className="text-textMuted font-medium font-mono uppercase text-xs tracking-widest">Nessun strumento configurato</p>
       </div>
      )}

      {hasMore && (
        <div className="flex justify-center">
          <button 
            onClick={loadMore} 
            disabled={loading}
            className="rounded-lg border border-border px-4 py-2 text-sm text-textMuted hover:text-textPrimary hover:border-textMuted transition-colors disabled:opacity-50"
          >
            {loading ? t('generic.loadingLower') : t('generic.loadMore')}
          </button>
        </div>
      )}

      {selectedTool && (
        <GlassPanel
          header="Tool Details"
          sectionKey="tools.detail"
          expanded={expandedSections['tools.detail']}
          onToggle={() => toggleSection('tools.detail')}
        >
          <div className="bg-surface-alt border border-border rounded-lg overflow-hidden shadow-lg shadow-primary/5">
            {/* Detail Panel Header */}
            <div className="p-4 border-b border-border flex items-center justify-between bg-surface">
              <div className="flex items-center space-x-3">
                <div className="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center">
                  <Terminal size={16} className="text-primary" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-textPrimary">{selectedTool.name}</h3>
                  <span className="text-[10px] text-textDim font-mono uppercase">{selectedTool.id}</span>
                </div>
              </div>
              <button
                onClick={() => setSelectedToolId(null)}
                className="p-2 text-textDim hover:text-textPrimary hover:bg-border/50 rounded-lg transition-colors"
                aria-label="Close detail panel"
              >
                <X size={16} />
              </button>
            </div>

            {/* Detail Body */}
            <div className="p-4 space-y-4">
              {/* Category Tag */}
              <div className="flex items-center space-x-2">
                <span className="text-[10px] font-bold text-textMuted uppercase tracking-widest">Category</span>
                <span className="px-2 py-0.5 bg-primary/10 text-primary rounded text-[10px] font-mono uppercase tracking-tighter">
                  {'tool'}
                </span>
              </div>

              {/* Health Status */}
              <div className="flex items-center space-x-2">
                <span className="text-[10px] font-bold text-textMuted uppercase tracking-widest">Health</span>
                <ToolHealthIndicator
                  status={selectedTool.healthStatus || 'unknown'}
                  lastCheck={
                    selectedTool.lastCheckedAt
                      ? new Date(selectedTool.lastCheckedAt).toLocaleTimeString()
                      : undefined
                  }
                />
                <span className="text-[10px] text-textDim font-mono uppercase tracking-tighter">
                  {selectedTool.healthStatus === 'healthy'
                    ? 'Healthy'
                    : selectedTool.healthStatus === 'warning'
                    ? 'Degraded'
                    : selectedTool.healthStatus === 'error'
                    ? 'Down'
                    : 'Unknown'}
                </span>
                {selectedTool.lastCheckedAt && (
                  <span className="text-[10px] text-textDim">
                    · {new Date(selectedTool.lastCheckedAt).toLocaleString()}
                  </span>
                )}
              </div>

              {/* Description */}
              <div>
                <span className="text-[10px] font-bold text-textMuted uppercase tracking-widest block mb-1">Description</span>
                <p className="text-sm text-textPrimary">{selectedTool.description}</p>
              </div>

              {/* Code Preview */}
              <div>
                <span className="text-[10px] font-bold text-textMuted uppercase tracking-widest block mb-2">Code</span>
                <pre className="p-3 bg-background border border-border rounded-lg text-[10px] font-mono text-textMuted overflow-auto max-h-32">
                  {selectedTool.code}
                </pre>
              </div>

              {/* Action Buttons */}
              <div className="flex space-x-2">
                <button
                  onClick={() => onExecuteTool(selectedTool.id)}
                  className="flex-1 py-2 bg-primary text-background rounded-lg text-xs font-bold hover:bg-primary/90 transition-colors uppercase tracking-widest flex items-center justify-center space-x-1"
                >
                  <Play size={12} />
                  <span>Execute</span>
                </button>
                <button
                  onClick={() => onEditTool(selectedTool)}
                  className="flex-1 py-2 bg-surface text-textMuted rounded-lg text-xs font-bold hover:bg-border transition-colors uppercase tracking-widest"
                >
                  Edit
                </button>
              </div>

              {/* Execution Result */}
              {executionResult !== null && (
                <div>
                  <span className="text-[10px] font-bold text-textMuted uppercase tracking-widest block mb-2">
                    Execution Result
                  </span>
                  <ToolResultDisplay result={executionResult} />
                </div>
              )}
            </div>
          </div>
        </GlassPanel>
      )}
    </div>
    );
  });