import React, { useState } from 'react';
import { Briefcase, Plus, Key, Lock, ArrowRight, X, Trash2, Binary, Sparkles, AlertTriangle } from 'lucide-react';
import { t } from '../i18n';

interface Project {
  id: string;
  name: string;
}

interface WorkspaceOnboardingProps {
  projects: Project[];
  onSelectProject: (id: string, key: string) => void;
  onDeleteProject: (id: string, apiKey: string) => void;
  onCreateProject: () => void;
}

const DeleteConfirmModal: React.FC<{ project: Project; onConfirm: (apiKey: string) => void; onCancel: () => void }> = ({ project, onConfirm, onCancel }) => {
  const [keyInput, setKeyInput] = useState('');

  return (
    <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-8" onClick={onCancel}>
      <div className="max-w-md w-full bg-surface p-12 rounded-lg shadow-lg space-y-8 relative" onClick={e => e.stopPropagation()}>
        <button onClick={onCancel} className="absolute top-8 right-8 p-3 bg-surface-alt rounded-lg text-textMuted hover:text-text transition-all">
          <X size={20} />
        </button>
        <div className="flex flex-col items-center text-center space-y-4">
          <div className="w-16 h-16 bg-danger/10 text-danger rounded-lg flex items-center justify-center">
            <AlertTriangle size={32} />
          </div>
          <h3 className="text-2xl font-black text-danger tracking-tighter uppercase italic">Elimina {project.name}</h3>
          <p className="text-textMuted text-sm leading-relaxed">Questa azione è irreversibile. Tutti i dati, gli agenti e le ontologie saranno eliminati permanentemente. Inserisci l'API Key dello spazio di lavoro per confermare.</p>
        </div>
        <div className="relative">
          <Key className="absolute left-4 top-4 text-textDim" size={20} />
          <input
            autoFocus
            type="password"
            value={keyInput}
            onChange={e => setKeyInput(e.target.value)}
            className="w-full pl-14 pr-6 py-4 bg-surface-alt border-2 border-transparent rounded-lg focus:bg-surface focus:border-danger outline-none transition-all font-mono text-sm text-text"
            placeholder={t('setup.apiKey')}
          />
        </div>
        <button
          onClick={() => onConfirm(keyInput)}
          disabled={!keyInput.trim()}
          className="w-full py-4 bg-danger text-white rounded-lg font-black text-xs uppercase tracking-widest hover:bg-danger/90 transition-all disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center space-x-2"
        >
          <Trash2 size={16} />
          <span>Elimina definitivamente</span>
        </button>
      </div>
    </div>
  );
};

export const WorkspaceOnboarding: React.FC<WorkspaceOnboardingProps> = ({ projects, onSelectProject, onDeleteProject, onCreateProject }) => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [keyInput, setKeyInput] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<Project | null>(null);

  if (selectedId) {
    const project = projects.find(p => p.id === selectedId);
    return (
      <div className="flex h-screen bg-surface items-center justify-center p-8 bg-gradient-to-br from-primary/5 to-surface animate-in fade-in zoom-in duration-500">
        <div className="max-w-md w-full bg-surface p-12 rounded-lg shadow-lg border border-primary/10 space-y-10 relative overflow-hidden group">
          <div className="absolute -top-24 -right-24 w-48 h-48 bg-primary/5 rounded-full group-hover:scale-150 transition-transform duration-1000"></div>
          <button onClick={() => setSelectedId(null)} className="absolute top-8 right-8 p-3 bg-surface-alt rounded-lg text-textMuted hover:text-text transition-all hover:bg-border">
            <X size={20} />
          </button>
          
          <div className="text-center space-y-4">
            <div className="w-20 h-20 bg-primary rounded-lg flex items-center justify-center text-white mx-auto shadow-lg mb-6 rotate-3 group-hover:rotate-0 transition-transform duration-500">
               <Lock size={36} className="fill-current/20" />
            </div>
            <h2 className="text-4xl font-black text-text tracking-tighter uppercase italic">Sblocca {project?.name}</h2>
            <p className="text-textMuted font-medium">Inserisci l'API Key per accedere a questo spazio di lavoro.</p>
          </div>

          <div className="space-y-6">
             <div className="relative group/input">
                <Key className="absolute left-6 top-5 text-textDim group-focus-within/input:text-primary transition-colors" size={24} />
                <input 
                   autoFocus
                   type="password"
                   value={keyInput}
                   onChange={e => setKeyInput(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && keyInput.trim() && onSelectProject(selectedId, keyInput)}
                   className="w-full pl-16 pr-6 py-5 bg-surface-alt border-2 border-transparent rounded-lg focus:bg-surface focus:border-primary outline-none transition-all font-mono text-lg shadow-inner text-text"
                   placeholder={t('onboarding.apiKey')}
                />
             </div>
             <button 
                onClick={() => onSelectProject(selectedId, keyInput)}
                className="w-full py-5 bg-primary text-white rounded-lg text-xs font-black uppercase tracking-[0.2em] hover:bg-primary/90 transition-all shadow-lg flex items-center justify-center space-x-3 group/btn"
             >
                 <span>Accedi</span>
                <ArrowRight size={20} className="group-hover:translate-x-1 transition-transform" />
             </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-surface items-center justify-center p-12 bg-surface-alt overflow-hidden relative">
      {/* Decorative Elements */}
      <div className="absolute top-0 left-0 w-full h-full opacity-[0.03] pointer-events-none overflow-hidden">
         <Binary size={400} className="absolute -top-20 -left-20 rotate-12" />
         <Sparkles size={300} className="absolute bottom-0 right-0 -rotate-12" />
      </div>

      <div className="max-w-5xl w-full flex flex-col items-center space-y-16 relative z-10">
        <div className="flex flex-col items-center text-center space-y-6">
           <div className="flex items-center space-x-4 animate-in fade-in slide-in-from-top duration-700">
              <div className="w-16 h-16 bg-primary rounded-lg flex items-center justify-center text-white shadow-lg">
                 <Binary size={36} />
              </div>
               <h1 className="text-7xl font-black tracking-tighter text-text uppercase italic leading-none">Aleph</h1>
           </div>
            <p className="text-2xl text-textMuted font-medium leading-relaxed max-w-2xl animate-in fade-in slide-in-from-bottom duration-1000 delay-200">
              Open Intelligence System
              <span className="block text-textDim mt-2">Scegli o crea uno spazio di lavoro per il tuo progetto.</span>
            </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 w-full animate-in fade-in slide-in-from-bottom-8 duration-1000 delay-500">
          {projects.map(p => (
            <div 
              key={p.id} 
              onClick={() => setSelectedId(p.id)} 
              className="p-10 bg-surface rounded-lg border border-border shadow-lg hover:shadow-lg hover:border-primary/30 cursor-pointer transition-all hover:-translate-y-2 group relative overflow-hidden"
            >
               <div className="absolute -bottom-8 -right-8 w-24 h-24 bg-primary/10 rounded-full group-hover:scale-150 transition-transform duration-700"></div>
               <button 
                  onClick={(e) => { e.stopPropagation(); setDeleteTarget(p); }}
                  className="absolute top-8 right-8 p-3 text-textDim hover:text-danger hover:bg-danger/10 rounded-lg transition-all z-10"
               >
                  <Trash2 size={20} />
               </button>
               <div className="flex flex-col h-full justify-between relative z-10">
                  <div className="space-y-4">
                    <div className="w-12 h-12 bg-primary/10 text-primary rounded-lg flex items-center justify-center group-hover:bg-primary group-hover:text-white transition-all">
                       <Briefcase size={24} />
                    </div>
                    <div>
                       <div className="font-black text-2xl text-text tracking-tight group-hover:text-primary transition-colors pr-8 uppercase italic leading-tight">{p.name}</div>
                       <div className="text-[10px] font-black text-textDim uppercase tracking-widest mt-2">Space ID: {p.id}</div>
                    </div>
                  </div>
                  <div className="mt-8 flex items-center text-primary font-black text-[10px] uppercase tracking-widest opacity-0 group-hover:opacity-100 transition-all translate-x-2 group-hover:translate-x-0">
                     <span>Accedi</span>
                     <ArrowRight size={14} className="ml-2" />
                  </div>
               </div>
            </div>
          ))}
          
          <div 
            onClick={onCreateProject} 
            className="p-10 bg-surface rounded-lg border-4 border-dashed border-border hover:border-primary cursor-pointer flex flex-col items-center justify-center space-y-4 group transition-all hover:bg-primary/5"
          >
             <div className="w-16 h-16 bg-surface-alt text-textDim rounded-lg flex items-center justify-center group-hover:bg-primary group-hover:text-white transition-all">
                <Plus size={32} />
             </div>
              <span className="text-xs font-black text-textMuted group-hover:text-primary uppercase tracking-widest transition-all text-center">Nuovo spazio di lavoro</span>
          </div>

        </div>
      </div>

      {deleteTarget && (
        <DeleteConfirmModal
          project={deleteTarget}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={(key) => { onDeleteProject(deleteTarget.id, key); setDeleteTarget(null); }}
        />
      )}
    </div>
  );
};
