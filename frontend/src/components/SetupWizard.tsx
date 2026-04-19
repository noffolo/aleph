import React, { useState } from 'react';
import { Database, Zap, ArrowRight, CheckCircle2, ShieldCheck, Key, Copy } from 'lucide-react';

interface SetupWizardProps {
  onComplete: (projectID: string, apiKey: string) => void;
  onCreateProject: (name: string) => Promise<string>;
  onCreateApiKey: (projectID: string, label: string) => Promise<string>;
}

export const SetupWizard: React.FC<SetupWizardProps> = ({ onComplete, onCreateProject, onCreateApiKey }) => {
  const [step, setStep] = useState(1);
  const [projectName, setProjectName] = useState('');
  const [projectID, setProjectID] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(false);

  const handleStep1 = async () => {
    setLoading(true);
    const id = await onCreateProject(projectName);
    setProjectID(id);
    setLoading(false);
    setStep(2);
  };

  const handleStep2 = async () => {
    setLoading(true);
    const key = await onCreateApiKey(projectID, 'Admin Key (Wizard)');
    setApiKey(key);
    setLoading(false);
    setStep(3);
  };

  return (
    <div className="fixed inset-0 bg-white z-[300] flex flex-col items-center justify-center p-8 bg-gradient-to-br from-blue-50 to-white">
      <div className="max-w-xl w-full">
        <div className="flex items-center justify-between mb-12">
           {[1, 2, 3].map(s => (
             <div key={s} className="flex items-center">
                <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold text-sm transition-all ${step >= s ? 'bg-blue-600 text-white shadow-lg' : 'bg-gray-100 text-gray-400'}`}>
                   {step > s ? <CheckCircle2 size={20} /> : s}
                </div>
                {s < 3 && <div className={`w-20 h-1 mx-2 rounded-full ${step > s ? 'bg-blue-600' : 'bg-gray-100'}`}></div>}
             </div>
           ))}
        </div>

        {step === 1 && (
          <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
             <div className="text-center">
                <h2 className="text-4xl font-bold tracking-tight text-blue-900 mb-4">Crea il tuo Workspace</h2>
                <p className="text-gray-500">Aleph ha bisogno di un nome per organizzare i tuoi dati e i tuoi agenti.</p>
             </div>
             <input 
                autoFocus
                value={projectName}
                onChange={e => setProjectName(e.target.value)}
                className="w-full p-6 bg-white border border-gray-200 rounded-3xl text-center text-2xl font-bold focus:ring-4 focus:ring-blue-500/10 outline-none transition-all shadow-xl"
                placeholder="es: Analisi Appalti 2026"
             />
             <button 
                onClick={handleStep1}
                disabled={!projectName || loading}
                className="w-full py-5 bg-blue-600 text-white rounded-3xl font-bold text-lg hover:bg-blue-700 transition-all shadow-xl shadow-blue-100 flex items-center justify-center space-x-3"
             >
                {loading ? <Activity size={24} className="animate-spin" /> : <><span>Prosegui</span> <ArrowRight size={24} /></>}
             </button>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500 text-center">
             <ShieldCheck size={64} className="mx-auto text-blue-600" />
             <div>
                <h2 className="text-4xl font-bold tracking-tight text-blue-900 mb-4">Metti in sicurezza il nodo</h2>
                <p className="text-gray-500">Generiamo una chiave di accesso univoca (API Key) per proteggere il tuo workspace '{projectName}'.</p>
             </div>
             <button 
                onClick={handleStep2}
                className="w-full py-5 bg-blue-900 text-white rounded-3xl font-bold text-lg hover:bg-black transition-all shadow-xl flex items-center justify-center space-x-3"
             >
                {loading ? <Activity size={24} className="animate-spin" /> : <><span>Genera API Key Protetta</span> <Key size={24} /></>}
             </button>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-8 animate-in fade-in zoom-in-95 duration-500 text-center">
             <div className="w-20 h-20 bg-green-100 text-green-600 rounded-full flex items-center justify-center mx-auto mb-6">
                <CheckCircle2 size={40} />
             </div>
             <h2 className="text-4xl font-bold tracking-tight text-blue-900">Workspace Pronto!</h2>
             <div className="p-6 bg-white rounded-3xl border border-green-100 shadow-sm text-left font-mono text-xs text-gray-500 break-all relative group">
                <div className="font-bold text-green-600 mb-2 uppercase tracking-widest">Tua API Key (salvala!)</div>
                {apiKey}
                <button 
                   onClick={() => { navigator.clipboard.writeText(apiKey); alert("Copiata!"); }}
                   className="absolute top-4 right-4 p-2 bg-gray-50 rounded-lg text-gray-400 hover:text-blue-600 opacity-0 group-hover:opacity-100 transition-all"
                   title="Copia negli appunti"
                >
                   <Copy size={14} />
                </button>
             </div>             <button 
                onClick={() => onComplete(projectID, apiKey)}
                className="w-full py-5 bg-blue-600 text-white rounded-3xl font-bold text-lg hover:bg-blue-700 transition-all shadow-xl shadow-blue-100"
             >
                Inizia ad Analizzare
             </button>
          </div>
        )}
      </div>
    </div>
  );
};

import { Activity } from 'lucide-react';
