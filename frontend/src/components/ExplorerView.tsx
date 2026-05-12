import React from 'react';
import { Search, Table as TableIcon, Map as MapIcon, Clock, Share2 as GraphIcon } from 'lucide-react';
import { t } from '../i18n';
import { AlephTable } from '../lib/AlephTable';
import { AlephMap } from '../lib/AlephMap';
import { AlephTimeline } from '../lib/AlephTimeline';
import { AlephGraph } from '../lib/AlephGraph';
import { SkeletonLoader } from './SkeletonLoader';

interface ExplorerViewProps {
  availableObjects: string[];
  selectedObject: string;
  setSelectedObject: (obj: string) => void;
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  activeView: string;
  setActiveView: (view: string) => void;
  data: Record<string, unknown> | null;
  onRowClick: (row: Record<string, unknown>) => void;
  isLoading: boolean;
  inline?: boolean;
}

type QueryData = {
  columns?: string[];
  rows?: Record<string, unknown>[];
  sql?: string;
}

export const ExplorerView: React.FC<ExplorerViewProps> = React.memo(({
  availableObjects, selectedObject, setSelectedObject,
  searchQuery, setSearchQuery, activeView, setActiveView,
  data, onRowClick, isLoading
}) => {
  const queryData = data as QueryData | null
  // Cast rows for lib components which expect their own Row type
  const rows = (queryData?.rows ?? []) as any[]

  return (
    <div className="max-w-6xl mx-auto space-y-4">
      <div className="flex items-center space-x-1 overflow-x-auto pb-2 no-scrollbar">
        {availableObjects.map(obj => (
          <button 
            key={obj} 
            onClick={() => setSelectedObject(obj)}
            className={`px-3 py-1.5 text-xs font-mono font-bold transition-colors whitespace-nowrap border ${selectedObject === obj ? 'bg-primary/10 text-primary border-primary/30' : 'bg-surface text-textMuted border-border hover:text-text hover:border-textDim'}`}
          >
            {obj}
          </button>
        ))}
      </div>

      <div className="flex flex-col md:flex-row items-center justify-between mb-4 gap-4">
        <div className="relative flex-1 w-full">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-textDim" size={16} />
          <input 
            type="text" 
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={`Cerca in ${selectedObject || '...'}...`} 
            className="w-full pl-10 pr-4 py-2.5 bg-surface border border-border font-mono text-sm text-text placeholder:text-textDim focus:outline-none focus:border-primary/50 transition-colors"
          />
        </div>
         <div className="flex bg-surface p-0.5 border border-border">
            <button onClick={() => setActiveView('table')} className={`px-2.5 py-1.5 ${activeView === 'table' ? 'bg-primary/10 text-primary' : 'text-textMuted hover:text-text'}`} title={t('explorer.view.table')} aria-label={t('explorer.view.table')}><TableIcon size={16} /></button>
             <button onClick={() => setActiveView('map')} className={`px-2.5 py-1.5 ${activeView === 'map' ? 'bg-primary/10 text-primary' : 'text-textMuted hover:text-text'}`} title={t('explorer.view.map')} aria-label={t('explorer.view.map')}><MapIcon size={16} /></button>
             <button onClick={() => setActiveView('timeline')} className={`px-2.5 py-1.5 ${activeView === 'timeline' ? 'bg-primary/10 text-primary' : 'text-textMuted hover:text-text'}`} title={t('explorer.view.timeline')} aria-label={t('explorer.view.timeline')}><Clock size={16} /></button>
             <button onClick={() => setActiveView('graph')} className={`px-2.5 py-1.5 ${activeView === 'graph' ? 'bg-primary/10 text-primary' : 'text-textMuted hover:text-text'}`} title={t('explorer.view.graph')} aria-label={t('explorer.view.graph')}><GraphIcon size={16} /></button>
         </div>
      </div>

      <div className="min-h-[500px]">
        {isLoading ? (
          <SkeletonLoader />
        ) : (
          <>
            {activeView === 'table' && <AlephTable columns={queryData?.columns || []} rows={rows} onRowClick={onRowClick as any} />}
            {activeView === 'map' && <AlephMap rows={rows} onRowClick={onRowClick as any} />}
            {activeView === 'timeline' && <AlephTimeline rows={rows} onRowClick={onRowClick as any} />}
            {activeView === 'graph' && <AlephGraph rows={rows} onRowClick={onRowClick as any} />}
          </>
        )}
      </div>
      
      {queryData?.sql && (
        <div className="mt-6 p-4 bg-surface overflow-hidden border border-border">
           <div className="text-textDim text-[10px] font-mono mb-3 uppercase tracking-widest flex justify-between items-center">
             <span>DuckDB Engine • No-ETL Query</span>
             <div className="flex space-x-1">
                <div className="w-1.5 h-1.5 rounded-full bg-red-500"></div>
                <div className="w-1.5 h-1.5 rounded-full bg-yellow-500"></div>
                <div className="w-1.5 h-1.5 rounded-full bg-green-500"></div>
             </div>
           </div>
           <code className="text-primary font-mono text-xs break-all leading-relaxed">{queryData.sql}</code>
        </div>
      )}
    </div>
  );
});
