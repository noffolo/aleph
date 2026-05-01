import React, { useState } from 'react';
import { Plus, Activity, X, Trash2, Database, Globe, FileText, Link, Play, Mail, Rss, Github, Terminal } from 'lucide-react';
import { useStore } from '../store/useStore';
import { t } from '../i18n';
import { SkeletonLoader } from './SkeletonLoader';
import { InlineError } from './ui/InlineError';

interface IngestionTask {
  id: string;
  name: string;
  sourceType: string;
  status: string;
  progress: number;
}

interface DataSourcesViewProps {
  tasks: IngestionTask[];
  onAddSource: (config: { name: string; sourceType: string; configJson: string }) => void;
  onRunTask: (id: string) => void;
  onViewLogs: (id: string) => void;
  onDeleteTask: (id: string) => void;
  taskLogs: string;
  setTaskLogs: (val: string) => void;
  inline?: boolean;
  isLoading?: boolean;
  error?: string | null;
}

export const DataSourcesView: React.FC<DataSourcesViewProps> = React.memo(({
  tasks, onAddSource, onRunTask, onViewLogs, onDeleteTask, taskLogs, setTaskLogs, inline = false, isLoading, error
}) => {
  const openForm = () => {
    useStore.getState().setSlideOverContent({ type: 'datasource-form', title: t('datasources.title'), data: undefined });
  };

  if (isLoading) return <SkeletonLoader />;
  if (error) return <div className="max-w-6xl mx-auto"><InlineError message={error} /></div>;

  return (
    <div className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-8'}>
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">{t('datasources.title')}</h2>
          <p className="text-textMuted text-sm mt-1">{t('datasources.subtitle')}</p>
        </div>
<button onClick={openForm} className="flex items-center space-x-2 bg-primary text-background px-6 py-3 rounded-lg font-bold hover:bg-primary/90 transition-all shadow-lg focus:ring-2 focus:ring-primary" aria-label="Add data source">
            <Plus size={20} />
            <span className="font-bold text-sm">{t('datasources.create')}</span>
         </button>
      </div>

      <div className="grid grid-cols-1 gap-4">
        {tasks.map(task => (
          <div key={task.id} className="bg-surface p-6 rounded-lg border border-border shadow-sm flex items-center justify-between hover:border-primary/30 transition-colors">
             <div className="flex items-center space-x-6 flex-1">
                 <div className={`w-14 h-14 rounded-lg flex items-center justify-center ${task.status === 'running' || task.status === 'esecuzione' ? 'bg-warning/10 text-warning' : task.status === 'completato' || task.status === 'completed' ? 'bg-success/10 text-success' : task.status === 'fallito' || task.status === 'failed' ? 'bg-danger/10 text-danger' : 'bg-surface-alt text-textMuted'}`}>
                     <Activity size={28} className={task.status === 'running' || task.status === 'esecuzione' ? 'animate-pulse' : ''} />
                </div>
                <div className="flex-1">
                   <div className="flex items-center space-x-3 mb-1">
                      <h3 className="font-bold text-xl">{task.name}</h3>
                      <span className="text-[10px] font-mono bg-primary/10 text-primary px-2 py-0.5 rounded uppercase font-bold tracking-widest">{task.sourceType}</span>
                   </div>
                   <div className="flex items-center space-x-4">
                      <div className="flex-1 bg-border h-2 rounded-full overflow-hidden max-w-md">
                         <div className="bg-primary h-full transition-all duration-700 ease-out" style={{ width: `${task.progress}%` }}></div>
                      </div>
                      <span className="text-xs font-bold text-textMuted">{task.progress}%</span>
                   </div>
                </div>
             </div>
             <div className="flex items-center space-x-3 ml-8 border-l pl-8 border-border">
                <button onClick={() => onViewLogs(task.id)} className="px-5 py-2.5 text-sm font-bold text-textMuted hover:bg-surface-alt rounded-lg transition-colors focus:ring-2 focus:ring-primary" aria-label={`View logs for ${task.name}`}>Logs</button>
                <button 
                    onClick={() => onRunTask(task.id)} 
                    disabled={task.status === 'running' || task.status === 'esecuzione'} 
                      className={`px-8 py-2.5 rounded-lg text-sm font-bold transition-all flex items-center space-x-2 focus:ring-2 focus:ring-primary ${(task.status === 'running' || task.status === 'esecuzione') ? 'bg-border text-textMuted' : 'bg-surface-alt text-text hover:bg-border shadow-lg'}`}
                 >
                    <Play size={14} />
                    <span>{(task.status === 'running' || task.status === 'esecuzione') ? t('datasources.status.running') : task.status === 'completato' || task.status === 'completed' ? t('datasources.status.completed') : task.status === 'fallito' || task.status === 'failed' ? t('datasources.status.failed') : t('datasources.status.execute')}</span>
                </button>
<button 
                     onClick={(e) => { e.stopPropagation(); if (confirm(t('datasources.confirmDelete'))) onDeleteTask(task.id); }}
                     className="p-2.5 text-textDim hover:text-danger hover:bg-danger/10 rounded-lg transition-all focus:ring-2 focus:ring-primary"
                     aria-label={`Delete task ${task.name}`}
                  >
                   <Trash2 size={18} />
                </button>
             </div>
          </div>
        ))}
        {tasks.length === 0 && (
           <div className="py-20 bg-surface border-2 border-dashed border-border rounded-lg text-center">
              <Database size={48} className="mx-auto text-textDim mb-4" />
              <p className="text-textDim font-bold uppercase text-xs tracking-[0.2em] mb-2">{t('datasources.noPipeline')}</p>
                <p className="text-textMuted text-sm">{t('datasources.empty')}</p>
                <button onClick={openForm} className="mt-6 px-6 py-3 bg-primary text-background rounded-lg font-bold hover:bg-primary/90 transition-all shadow-lg focus:ring-2 focus:ring-primary" aria-label="Add data source">{t('datasources.create')}</button>
           </div>
        )}
      </div>

      {taskLogs && (
        <div className="mt-8 bg-background rounded-3xl overflow-hidden shadow-2xl border border-border">
           <div className="flex justify-between items-center p-4 bg-border/50 border-b border-border/50">
              <span className="text-[10px] font-mono text-textMuted uppercase tracking-widest font-bold">{t('datasources.logOutput')}</span>
                <button onClick={() => setTaskLogs('')} className="p-1 hover:bg-surface-alt rounded-lg text-textMuted transition-colors focus:ring-2 focus:ring-primary" aria-label="Close log panel"><X size={14} /></button>
            </div>
            <pre className="p-6 text-success font-mono text-xs overflow-auto max-h-[400px] leading-relaxed custom-scrollbar whitespace-pre-wrap">
              {taskLogs}
           </pre>
        </div>
      )}
    </div>
  );
});