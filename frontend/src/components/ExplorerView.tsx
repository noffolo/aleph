import React from 'react';
import { Search, Table as TableIcon, Map as MapIcon, Clock, Share2 as GraphIcon } from 'lucide-react';
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
  data: any;
  onRowClick: (row: any) => void;
  isLoading: boolean;
}

export const ExplorerView: React.FC<ExplorerViewProps> = ({
  availableObjects, selectedObject, setSelectedObject,
  searchQuery, setSearchQuery, activeView, setActiveView,
  data, onRowClick, isLoading
}) => {
  return (
    <div className="max-w-6xl mx-auto space-y-6">
      <div className="flex items-center space-x-2 overflow-x-auto pb-2 no-scrollbar">
        {availableObjects.map(obj => (
          <button 
            key={obj} 
            onClick={() => setSelectedObject(obj)}
            className={`px-4 py-2 rounded-full text-xs font-bold transition-all whitespace-nowrap ${selectedObject === obj ? 'bg-blue-600 text-white shadow-lg' : 'bg-white border text-gray-500 hover:border-blue-300'}`}
          >
            {obj}
          </button>
        ))}
      </div>

      <div className="flex flex-col md:flex-row items-center justify-between mb-4 gap-4">
        <div className="relative flex-1 w-full">
          <Search className="absolute left-4 top-4 text-gray-400" size={20} />
          <input 
            type="text" 
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={`Cerca in ${selectedObject || '...'}...`} 
            className="w-full pl-12 pr-4 py-4 bg-white border border-gray-200 rounded-2xl focus:outline-none focus:ring-4 focus:ring-blue-500/10 transition-all text-lg shadow-sm"
          />
        </div>
        <div className="flex bg-white p-1 rounded-2xl border border-gray-100 shadow-sm">
           <button onClick={() => setActiveView('table')} className={`p-3 rounded-xl ${activeView === 'table' ? 'bg-blue-600 text-white shadow-lg' : 'text-gray-400'}`} title="Tabella"><TableIcon size={20} /></button>
           <button onClick={() => setActiveView('map')} className={`p-3 rounded-xl ${activeView === 'map' ? 'bg-blue-600 text-white shadow-lg' : 'text-gray-400'}`} title="Mappa"><MapIcon size={20} /></button>
           <button onClick={() => setActiveView('timeline')} className={`p-3 rounded-xl ${activeView === 'timeline' ? 'bg-blue-600 text-white shadow-lg' : 'text-gray-400'}`} title="Timeline"><Clock size={20} /></button>
           <button onClick={() => setActiveView('graph')} className={`p-3 rounded-xl ${activeView === 'graph' ? 'bg-blue-600 text-white shadow-lg' : 'text-gray-400'}`} title="Grafo Relazionale"><GraphIcon size={20} /></button>
        </div>
      </div>

      <div className="min-h-[500px]">
        {isLoading ? (
          <SkeletonLoader />
        ) : (
          <>
            {activeView === 'table' && <AlephTable columns={data?.columns || []} rows={data?.rows || []} onRowClick={onRowClick} />}
            {activeView === 'map' && <AlephMap rows={data?.rows || []} onRowClick={onRowClick} />}
            {activeView === 'timeline' && <AlephTimeline rows={data?.rows || []} onRowClick={onRowClick} />}
            {activeView === 'graph' && <AlephGraph rows={data?.rows || []} onRowClick={onRowClick} />}
          </>
        )}
      </div>
      
      {data?.sql && (
        <div className="mt-12 p-6 bg-gray-900 rounded-3xl overflow-hidden shadow-2xl border border-gray-800">
           <div className="text-gray-400 text-[10px] font-mono mb-4 uppercase tracking-widest flex justify-between items-center">
             <span>DuckDB Engine • No-ETL Query</span>
             <div className="flex space-x-1">
                <div className="w-2 h-2 rounded-full bg-red-500"></div>
                <div className="w-2 h-2 rounded-full bg-yellow-500"></div>
                <div className="w-2 h-2 rounded-full bg-green-500"></div>
             </div>
           </div>
           <code className="text-blue-300 font-mono text-xs break-all leading-relaxed">{data.sql}</code>
        </div>
      )}
    </div>
  );
};
