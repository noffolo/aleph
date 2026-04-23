import React, { useState } from 'react';
import { Database, Zap, ArrowRight, CheckCircle2, ShieldCheck, Key, Copy, Activity, AlertTriangle } from 'lucide-react';

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
  const [error, setError] = useState('');

  const handleStep1 = async () => {
    setLoading(true);
    setError('');
    try {
      const id = await onCreateProject(projectName);
      setProjectID(id);
      setStep(2);
    } catch (err: any) {
      setError(err.message || 'Errore nella creazione del progetto');
    } finally {
      setLoading(false);
    }
  };

  const handleStep2 = async () => {
    setLoading(true);
    setError('');
    try {
      const key = await onCreateApiKey(projectID, 'Admin Key (Wizard)');
      setApiKey(key);
      setStep(3);
    } catch (err: any) {
      setError(err.message || 'Errore nella generazione della chiave');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-surface z-[300] flex flex-col items-center justify-center p-8 bg-gradient-to-br from-primary/5 to-surface">
       <div className="max-w-xl w-full">
        {error && <div className="mb-6 p-4 bg-danger/10 text-danger rounded-lg text-sm text-center">{error}</div>}
         <div className="flex items-center justify-between mb-12">
           {[1, 2, 3].map(s => (
             <div key={s} className="flex items-center">
                <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold text-sm transition-all ${step >= s ? 'bg-primary text-white shadow-lg' : 'bg-surface-alt text-textMuted'}`}>
                   {step > s ? <CheckCircle2 size={20} /> : s}
                </div>
                {s < 3 && <div className={`w-20 h-1 mx-2 rounded-full ${step > s ? 'bg-primary' : 'bg-surface-alt'}`}></div>}
             </div>
           ))}
        </div>

        {step === 1 && (
          <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
             <div className="text-center">
                 <h2 className="text-4xl font-bold tracking-tight text-text mb-4">Crea il tuo spazio di lavoro</h2>
                 <p className="text-textMuted">Assegna un nome per organizzare i dati e gli agenti del progetto.</p>
             </div>
             <input 
                autoFocus
                value={projectName}
                onChange={e => setProjectName(e.target.value)}
                className="w-full p-6 bg-surface border border-border rounded-lg text-center text-2xl font-bold focus:ring-4 focus:ring-primary/10 outline-none transition-all shadow-lg"
                 placeholder="xyz"
             />
             <button 
                onClick={handleStep1}
                disabled={!projectName || loading}
                className="w-full py-5 bg-primary text-white rounded-lg font-bold text-lg hover:bg-primary/90 transition-all shadow-lg  flex items-center justify-center space-x-3"
             >
                {loading ? <Activity size={24} className="animate-spin" /> : <><span>Prosegui</span> <ArrowRight size={24} /></>}
             </button>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500 text-center">
             <ShieldCheck size={64} className="mx-auto text-primary" />
             <div>
                  <h2 className="text-4xl font-bold tracking-tight text-text mb-4">Proteggi lo spazio di lavoro</h2>
                  <p className="text-textMuted">Genera una chiave di accesso univoca (API Key) per proteggere lo spazio di lavoro '{projectName}'. <strong className="text-warning">Salvala subito: non sarà più visibile dopo la chiusura di questa schermata.</strong></p>
             </div>
             <button 
                onClick={handleStep2}
                 className="w-full py-5 bg-surface-alt text-text rounded-lg font-bold text-lg hover:bg-surface transition-all shadow-lg flex items-center justify-center space-x-3"
             >
                {loading ? <Activity size={24} className="animate-spin" /> : <><span>Genera API Key Protetta</span> <Key size={24} /></>}
             </button>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-8 animate-in fade-in zoom-in-95 duration-500 text-center">
              <div className="w-20 h-20 bg-success/10 text-success rounded-full flex items-center justify-center mx-auto mb-6">
                 <CheckCircle2 size={40} />
              </div>
               <h2 className="text-4xl font-bold tracking-tight text-text">Spazio di lavoro pronto</h2>
              <div className="p-6 bg-warning/10 rounded-lg border-2 border-warning/30 text-left space-y-3">
                 <div className="flex items-start space-x-3">
                    <AlertTriangle size={20} className="text-warning shrink-0 mt-0.5" />
                    <div>
                       <div className="font-black text-warning uppercase tracking-widest text-xs mb-1">Attenzione: salva questa chiave!</div>
                        <p className="text-warning text-sm leading-relaxed">Questa è l'unica volta che potrai vedere questa API Key. Se la perdi, <strong>non potrai più accedere allo spazio di lavoro</strong>. Non è recuperabile in alcun modo.</p>
                    </div>
                 </div>
              </div>
              <div className="p-6 bg-surface rounded-lg border border-success/20 shadow-sm text-left font-mono text-xs text-textMuted break-all relative group">
                 <div className="font-bold text-success mb-2 uppercase tracking-widest">Tua API Key</div>
                 {apiKey}
                 <button 
                    onClick={() => { navigator.clipboard.writeText(apiKey).then(() => alert("Copiata!")).catch(() => {}); }}
                    className="absolute top-4 right-4 p-2 bg-surface-alt rounded-lg text-textMuted hover:text-primary opacity-0 group-hover:opacity-100 transition-all"
                    title="Copia negli appunti"
                 >
                    <Copy size={14} />
                 </button>
              </div>
              <button 
                 onClick={() => onComplete(projectID, apiKey)}
                 className="w-full py-5 bg-primary text-white rounded-lg font-bold text-lg hover:bg-primary/90 transition-all shadow-lg "
              >
                  Inizia
              </button>
          </div>
        )}
      </div>
    </div>
  );
};

