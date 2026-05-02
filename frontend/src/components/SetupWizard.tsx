import React, { useState } from 'react';
import { Database, Zap, ArrowRight, CheckCircle2, ShieldCheck, Key, Copy, Activity, AlertTriangle } from 'lucide-react';
import { t } from '../i18n';

interface SetupWizardProps {
  onComplete: (projectID: string, apiKey: string) => void;
  onCreateProject: (name: string) => Promise<string>;
  onCreateApiKey: (projectID: string, label: string) => Promise<string>;
}

export const SetupWizard: React.FC<SetupWizardProps> = ({ onComplete, onCreateProject, onCreateApiKey }) => {
  const [step, setStep] = useState(1);
  const [language, setLanguage] = useState<'it' | 'en'>('it');
  const [projectName, setProjectName] = useState('');
  const [projectID, setProjectID] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [showKey, setShowKey] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const copy = {
    it: {
      createTitle: 'Crea il tuo spazio di lavoro',
      createSubtitle: 'Assegna un nome per organizzare i dati e gli agenti del progetto.',
      continue: 'Prosegui',
      protectTitle: 'Proteggi lo spazio di lavoro',
      protectSubtitle: `Genera una chiave di accesso univoca (API Key) per proteggere lo spazio di lavoro '${projectName}'.`,
      saveWarning: 'Salvala subito: non sarà più visibile dopo la chiusura di questa schermata.',
      generateKey: 'Genera API Key Protetta',
      readyTitle: 'Spazio di lavoro pronto',
      attention: 'Attenzione: salva questa chiave!',
      keyWarning: "Questa è l'unica volta che potrai vedere questa API Key. Se la perdi, non potrai più accedere allo spazio di lavoro. Non è recuperabile in alcun modo.",
      yourKey: 'Tua API Key',
      start: 'Inizia',
    },
    en: {
      createTitle: 'Create your workspace',
      createSubtitle: 'Choose a name to organize this project data and agents.',
      continue: 'Continue',
      protectTitle: 'Protect your workspace',
      protectSubtitle: `Generate a unique API key to protect '${projectName}'.`,
      saveWarning: 'Save it now: it will not be visible after you leave this screen.',
      generateKey: 'Generate protected API key',
      readyTitle: 'Workspace ready',
      attention: 'Important: save this key!',
      keyWarning: 'This is the only time this API key will be visible. If you lose it, you will not be able to access the workspace again. It cannot be recovered.',
      yourKey: 'Your API key',
      start: 'Start',
    },
  }[language];

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
          <div className="flex items-center justify-between mb-8">
            {[1, 2, 3].map(s => (
              <div key={s} className="flex items-center">
                <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold text-sm transition-all ${step >= s ? 'bg-primary text-white shadow-lg' : 'bg-surface-alt text-textMuted'}`}>
                  {step > s ? <CheckCircle2 size={20} /> : s}
                </div>
                {s < 3 && <div className={`w-20 h-1 mx-2 rounded-full ${step > s ? 'bg-primary' : 'bg-surface-alt'}`}></div>}
              </div>
            ))}
         </div>
         <div className="flex justify-end mb-8">
           <div className="inline-flex rounded border border-border bg-surface-alt p-1" aria-label="Language">
             {(['it', 'en'] as const).map((value) => (
               <button
                 key={value}
                 type="button"
                 onClick={() => setLanguage(value)}
                 className={`px-3 py-1.5 rounded text-xs font-bold uppercase transition-colors focus:ring-2 focus:ring-primary ${language === value ? 'bg-primary text-white' : 'text-textMuted hover:text-text'}`}
                 aria-pressed={language === value}
               >
                 {value.toUpperCase()}
               </button>
             ))}
           </div>
         </div>

         {step === 1 && (
           <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
              <div className="text-center">
                  <h2 className="text-4xl font-bold tracking-tight text-text mb-4">{copy.createTitle}</h2>
                  <p className="text-textMuted">{copy.createSubtitle}</p>
              </div>
              <input 
                 autoFocus
                 value={projectName}
                 onChange={e => setProjectName(e.target.value)}
                 className="w-full p-6 bg-surface border border-border rounded-lg text-center text-2xl font-bold focus:ring-4 focus:ring-primary/10 outline-none transition-all shadow-lg"
                  placeholder="workspace-name"
              />
              <button 
                 onClick={handleStep1}
                 disabled={!projectName || loading}
                 className="w-full py-5 bg-primary text-white rounded-lg font-bold text-lg hover:bg-primary/90 transition-all shadow-lg flex items-center justify-center space-x-3"
              >
                 {loading ? <Activity size={24} className="animate-spin" /> : <><span>{copy.continue}</span> <ArrowRight size={24} /></>}
              </button>
           </div>
         )}

         {step === 2 && (
           <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500 text-center">
              <ShieldCheck size={64} className="mx-auto text-primary" />
              <div>
                   <h2 className="text-4xl font-bold tracking-tight text-text mb-4">{copy.protectTitle}</h2>
                   <p className="text-textMuted">{copy.protectSubtitle} <strong className="text-warning">{copy.saveWarning}</strong></p>
              </div>
              <button 
                 onClick={handleStep2}
                  className="w-full py-5 bg-surface-alt text-text rounded-lg font-bold text-lg hover:bg-surface transition-all shadow-lg flex items-center justify-center space-x-3"
              >
                 {loading ? <Activity size={24} className="animate-spin" /> : <><span>{copy.generateKey}</span> <Key size={24} /></>}
              </button>
           </div>
         )}

         {step === 3 && (
           <div className="space-y-8 animate-in fade-in zoom-in-95 duration-500 text-center">
               <div className="w-20 h-20 bg-success/10 text-success rounded-full flex items-center justify-center mx-auto mb-6">
                  <CheckCircle2 size={40} />
               </div>
                <h2 className="text-4xl font-bold tracking-tight text-text">{copy.readyTitle}</h2>
               <div className="p-6 bg-warning/10 rounded-lg border-2 border-warning/30 text-left space-y-3">
                  <div className="flex items-start space-x-3">
                     <AlertTriangle size={20} className="text-warning shrink-0 mt-0.5" />
                     <div>
                        <div className="font-black text-warning uppercase tracking-widest text-xs mb-1">{copy.attention}</div>
                         <p className="text-warning text-sm leading-relaxed">{copy.keyWarning}</p>
                     </div>
                  </div>
               </div>
               <div className="p-6 bg-surface rounded-lg border border-success/20 shadow-sm text-left font-mono text-xs text-textMuted break-all relative group">
                  <div className="font-bold text-success mb-2 uppercase tracking-widest">{copy.yourKey}</div>
                  {showKey ? apiKey : "API key created. Save it now — it won't be shown again."}
                  <button 
                     onClick={() => { 
                       navigator.clipboard.writeText(apiKey).then(() => { 
                         alert(t('setup.copied'));
                         setShowKey(false);
                       }).catch(() => {}); 
                     }}
                     className={`absolute top-4 right-4 p-2 bg-surface-alt rounded-lg text-textMuted hover:text-primary opacity-0 group-hover:opacity-100 transition-all ${!showKey && 'hidden'}`}
                     title={t('setup.copy')}
                  >
                     <Copy size={14} />
                  </button>
               </div>
               <button 
                  onClick={() => onComplete(projectID, apiKey)}
                  className="w-full py-5 bg-primary text-white rounded-lg font-bold text-lg hover:bg-primary/90 transition-all shadow-lg"
               >
                   {copy.start}
               </button>
           </div>
         )}
       </div>
     </div>
   );
};