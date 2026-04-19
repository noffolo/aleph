import React from 'react';
import { Plus, Activity, X, Trash2 } from 'lucide-react';

interface IngestionTask {
  id: string;
  name: string;
  sourceType: string;
  status: string;
  progress: number;
}

interface DataSourcesViewProps {
  tasks: IngestionTask[];
  onAddSource: () => void;
  onRunTask: (id: string) => void;
  onViewLogs: (id: string) => void;
  onDeleteTask: (id: string) => void;
  taskLogs: string;
  setTaskLogs: (val: string) => void;
}

export const DataSourcesView: React.FC<DataSourcesViewProps> = ({
  tasks, onAddSource, onRunTask, onViewLogs, onDeleteTask, taskLogs, setTaskLogs
}) => {
  return (
    <div className="max-w-6xl mx-auto space-y-8">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Sorgenti Dati</h2>
          <p className="text-gray-500 text-sm mt-1">Gestisci l'ingestione (No-ETL) da fonti esterne verso il workspace locale.</p>
        </div>
        <button onClick={onAddSource} className="flex items-center space-x-2 bg-blue-600 text-white px-6 py-3 rounded-2xl font-bold hover:bg-blue-700 transition-all shadow-lg shadow-blue-200">
           <Plus size={20} />
           <span>Aggiungi Fonte</span>
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4">
        {tasks.map(t => (
          <div key={t.id} className="bg-white p-6 rounded-3xl border border-gray-100 shadow-sm flex items-center justify-between hover:border-blue-200 transition-colors">
             <div className="flex items-center space-x-6 flex-1">
                <div className={`w-14 h-14 rounded-2xl flex items-center justify-center ${t.status === 'running' ? 'bg-amber-100 text-amber-600' : 'bg-gray-50 text-gray-400'}`}>
                   <Activity size={28} className={t.status === 'running' ? 'animate-pulse' : ''} />
                </div>
                <div className="flex-1">
                   <div className="flex items-center space-x-3 mb-1">
                      <h3 className="font-bold text-xl">{t.name}</h3>
                      <span className="text-[10px] font-mono bg-blue-50 text-blue-600 px-2 py-0.5 rounded uppercase font-bold tracking-widest">{t.sourceType}</span>
                   </div>
                   <div className="flex items-center space-x-4">
                      <div className="flex-1 bg-gray-100 h-2 rounded-full overflow-hidden max-w-md">
                         <div className="bg-blue-600 h-full transition-all duration-700 ease-out" style={{ width: `${t.progress}%` }}></div>
                      </div>
                      <span className="text-xs font-bold text-gray-400">{t.progress}%</span>
                   </div>
                </div>
             </div>
             <div className="flex items-center space-x-3 ml-8 border-l pl-8 border-gray-50">
                <button onClick={() => onViewLogs(t.id)} className="px-5 py-2.5 text-sm font-bold text-gray-500 hover:bg-gray-100 rounded-xl transition-colors">Logs</button>
                <button 
                   onClick={() => onRunTask(t.id)} 
                   disabled={t.status === 'running'} 
                   className={`px-8 py-2.5 rounded-xl text-sm font-bold transition-all ${t.status === 'running' ? 'bg-gray-100 text-gray-400' : 'bg-gray-900 text-white hover:bg-black shadow-lg shadow-gray-200'}`}
                >
                   {t.status === 'running' ? 'In corso...' : 'Esegui'}
                </button>
                <button 
                   onClick={(e) => { e.stopPropagation(); if (confirm('Sei sicuro di voler eliminare questo task?')) onDeleteTask(t.id); }}
                   className="p-2.5 text-gray-300 hover:text-red-500 hover:bg-red-50 rounded-xl transition-all"
                >
                   <Trash2 size={18} />
                </button>
             </div>
          </div>
        ))}
        {tasks.length === 0 && (
           <div className="py-20 bg-white border-2 border-dashed border-gray-100 rounded-3xl text-center">
              <p className="text-gray-300 font-bold uppercase text-xs tracking-[0.2em]">Nessuna pipeline di ingestion configurata</p>
           </div>
        )}
      </div>

      {taskLogs && (
        <div className="mt-8 bg-gray-900 rounded-3xl overflow-hidden shadow-2xl border border-gray-800 animate-in slide-in-from-bottom-4 duration-300">
           <div className="flex justify-between items-center p-4 bg-gray-800/50 border-b border-gray-700/50">
              <span className="text-[10px] font-mono text-gray-400 uppercase tracking-widest font-bold">Execution Output (Real-time Logs)</span>
              <button onClick={() => setTaskLogs('')} className="p-1 hover:bg-gray-700 rounded-lg text-gray-400 transition-colors"><X size={14} /></button>
           </div>
           <pre className="p-6 text-green-400 font-mono text-xs overflow-auto max-h-[400px] leading-relaxed custom-scrollbar whitespace-pre-wrap">
              {taskLogs}
           </pre>
        </div>
      )}
    </div>
  );
};
