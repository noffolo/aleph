import React, { useState } from 'react';
import { Briefcase, Plus, Key, Lock, ArrowRight, X, Trash2, Binary, Sparkles } from 'lucide-react';

interface Project {
  id: string;
  name: string;
}

interface WorkspaceOnboardingProps {
  projects: Project[];
  onSelectProject: (id: string, key: string) => void;
  onDeleteProject: (id: string) => void;
  onCreateProject: () => void;
}

export const WorkspaceOnboarding: React.FC<WorkspaceOnboardingProps> = ({ projects, onSelectProject, onDeleteProject, onCreateProject }) => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [keyInput, setKeyInput] = useState('');

  if (selectedId) {
    const project = projects.find(p => p.id === selectedId);
    return (
      <div className="flex h-screen bg-white items-center justify-center p-8 bg-gradient-to-br from-blue-50 to-white animate-in fade-in zoom-in duration-500">
        <div className="max-w-md w-full bg-white p-12 rounded-[56px] shadow-2xl border border-blue-50 space-y-10 relative overflow-hidden group">
          <div className="absolute -top-24 -right-24 w-48 h-48 bg-blue-600/5 rounded-full group-hover:scale-150 transition-transform duration-1000"></div>
          <button onClick={() => setSelectedId(null)} className="absolute top-8 right-8 p-3 bg-gray-50 rounded-2xl text-gray-400 hover:text-gray-900 transition-all hover:bg-gray-100">
            <X size={20} />
          </button>
          
          <div className="text-center space-y-4">
            <div className="w-20 h-20 bg-blue-600 rounded-[28px] flex items-center justify-center text-white mx-auto shadow-2xl shadow-blue-200 mb-6 rotate-3 group-hover:rotate-0 transition-transform duration-500">
               <Lock size={36} className="fill-current/20" />
            </div>
            <h2 className="text-4xl font-black text-blue-950 tracking-tighter uppercase italic">Sblocca {project?.name}</h2>
            <p className="text-gray-500 font-medium">L'accesso a questo universo richiede una chiave simmetrica.</p>
          </div>

          <div className="space-y-6">
             <div className="relative group/input">
                <Key className="absolute left-6 top-5 text-gray-300 group-focus-within/input:text-blue-500 transition-colors" size={24} />
                <input 
                   autoFocus
                   type="password"
                   value={keyInput}
                   onChange={e => setKeyInput(e.target.value)}
                   onKeyDown={e => e.key === 'Enter' && onSelectProject(selectedId, keyInput)}
                   className="w-full pl-16 pr-6 py-5 bg-gray-50 border-2 border-transparent rounded-3xl focus:bg-white focus:border-blue-600 outline-none transition-all font-mono text-lg shadow-inner"
                   placeholder="Inserisci API Key..."
                />
             </div>
             <button 
                onClick={() => onSelectProject(selectedId, keyInput)}
                className="w-full py-5 bg-blue-600 text-white rounded-[28px] text-xs font-black uppercase tracking-[0.2em] hover:bg-blue-700 transition-all shadow-2xl shadow-blue-200 flex items-center justify-center space-x-3 group/btn"
             >
                <span>Accedi al Nexus</span>
                <ArrowRight size={20} className="group-hover:translate-x-1 transition-transform" />
             </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-white items-center justify-center p-12 bg-gray-50 overflow-hidden relative">
      {/* Decorative Elements */}
      <div className="absolute top-0 left-0 w-full h-full opacity-[0.03] pointer-events-none overflow-hidden">
         <Binary size={400} className="absolute -top-20 -left-20 rotate-12" />
         <Sparkles size={300} className="absolute bottom-0 right-0 -rotate-12" />
      </div>

      <div className="max-w-5xl w-full flex flex-col items-center space-y-16 relative z-10">
        <div className="flex flex-col items-center text-center space-y-6">
           <div className="flex items-center space-x-4 animate-in fade-in slide-in-from-top duration-700">
              <div className="w-16 h-16 bg-blue-600 rounded-[20px] flex items-center justify-center text-white shadow-2xl shadow-blue-100">
                 <Binary size={36} />
              </div>
              <h1 className="text-7xl font-black tracking-tighter text-blue-950 uppercase italic leading-none">Aleph v2</h1>
           </div>
           <p className="text-2xl text-gray-500 font-medium leading-relaxed max-w-2xl animate-in fade-in slide-in-from-bottom duration-1000 delay-200">
             Il primo Sistema Operativo Predittivo per i tuoi dati. 
             <span className="block text-gray-300 mt-2">Scegli un universo esistente o creane uno nuovo.</span>
           </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 w-full animate-in fade-in slide-in-from-bottom-8 duration-1000 delay-500">
          {projects.map(p => (
            <div 
              key={p.id} 
              onClick={() => setSelectedId(p.id)} 
              className="p-10 bg-white rounded-[48px] border border-gray-100 shadow-xl hover:shadow-2xl cursor-pointer transition-all hover:-translate-y-2 group relative overflow-hidden"
            >
               <div className="absolute -bottom-8 -right-8 w-24 h-24 bg-blue-50 rounded-full group-hover:scale-150 transition-transform duration-700"></div>
               <button 
                  onClick={(e) => { e.stopPropagation(); if (confirm('Sei sicuro di voler eliminare l\'intero progetto?')) onDeleteProject(p.id); }}
                  className="absolute top-8 right-8 p-3 text-gray-100 hover:text-red-500 hover:bg-red-50 rounded-2xl transition-all z-10"
               >
                  <Trash2 size={20} />
               </button>
               <div className="flex flex-col h-full justify-between relative z-10">
                  <div className="space-y-4">
                    <div className="w-12 h-12 bg-blue-50 text-blue-600 rounded-2xl flex items-center justify-center group-hover:bg-blue-600 group-hover:text-white transition-all">
                       <Briefcase size={24} />
                    </div>
                    <div>
                       <div className="font-black text-2xl text-blue-950 tracking-tight group-hover:text-blue-600 transition-colors pr-8 uppercase italic leading-tight">{p.name}</div>
                       <div className="text-[10px] font-black text-gray-300 uppercase tracking-widest mt-2">Space ID: {p.id}</div>
                    </div>
                  </div>
                  <div className="mt-8 flex items-center text-blue-600 font-black text-[10px] uppercase tracking-widest opacity-0 group-hover:opacity-100 transition-all translate-x-2 group-hover:translate-x-0">
                     <span>Accedi</span>
                     <ArrowRight size={14} className="ml-2" />
                  </div>
               </div>
            </div>
          ))}
          
          <div 
            onClick={onCreateProject} 
            className="p-10 bg-white rounded-[48px] border-4 border-dashed border-gray-100 hover:border-blue-600 cursor-pointer flex flex-col items-center justify-center space-y-4 group transition-all hover:bg-blue-50/50"
          >
             <div className="w-16 h-16 bg-gray-50 text-gray-300 rounded-[20px] flex items-center justify-center group-hover:bg-blue-600 group-hover:text-white transition-all">
                <Plus size={32} />
             </div>
             <span className="text-xs font-black text-gray-400 group-hover:text-blue-600 uppercase tracking-widest transition-all text-center">Inizializza Nuovo Universo</span>
          </div>

          {/* Guest Mode Option */}
          <div 
            onClick={() => onSelectProject('universo-demo', 'guest-access')}
            className="md:col-span-3 p-8 bg-blue-900 rounded-[40px] flex items-center justify-between group cursor-pointer hover:bg-blue-800 transition-all shadow-2xl shadow-blue-200"
          >
             <div className="flex items-center space-x-6 text-white">
                <div className="p-4 bg-white/10 rounded-2xl">
                   <Sparkles size={24} className="text-blue-300" />
                </div>
                <div>
                   <h3 className="text-xl font-black uppercase italic tracking-tighter">Esplora Aleph senza limiti</h3>
                   <p className="text-blue-200 text-xs font-medium uppercase tracking-widest">Accedi all'Universo Demo come Ospite</p>
                </div>
             </div>
             <ArrowRight size={24} className="text-white group-hover:translate-x-2 transition-transform" />
          </div>
        </div>
      </div>
    </div>
  );
};
