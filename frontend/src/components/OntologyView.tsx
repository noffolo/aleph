import React from 'react';
import { Zap, Save, Code } from 'lucide-react';

interface OntologyViewProps {
  ontologyRaw: string;
  setOntologyRaw: (val: string) => void;
  onEmerge: () => void;
  onSave: () => void;
}

export const OntologyView: React.FC<OntologyViewProps> = ({ ontologyRaw, setOntologyRaw, onEmerge, onSave }) => {
  return (
    <div className="max-w-6xl mx-auto space-y-6 h-full flex flex-col">
       <div className="flex justify-between items-center bg-white p-6 rounded-3xl border border-gray-100 shadow-sm">
          <div>
            <h2 className="text-3xl font-bold tracking-tight">Modellazione Business</h2>
            <p className="text-gray-500 text-sm mt-1">L'Ontologia è il filtro intelligente tra i tuoi dati grezzi e l'AI.</p>
          </div>
          <div className="flex space-x-3">
            <button onClick={onEmerge} className="flex items-center space-x-2 bg-amber-50 text-amber-700 px-6 py-3 rounded-2xl font-bold hover:bg-amber-100 transition-all shadow-sm border border-amber-100">
              <Zap size={20} />
              <span>Emergenza Automatica</span>
            </button>
            <button onClick={onSave} className="flex items-center space-x-2 bg-blue-600 text-white px-6 py-3 rounded-2xl font-bold hover:bg-blue-700 transition-all shadow-lg shadow-blue-200">
              <Save size={20} />
              <span>Pubblica Modello</span>
            </button>
          </div>
       </div>
       
       <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 flex-1 min-h-0">
          <div className="lg:col-span-2 bg-gray-900 rounded-3xl p-8 shadow-2xl border border-gray-800 flex flex-col min-h-[500px]">
            <div className="flex items-center justify-between mb-4 border-b border-gray-800 pb-4">
               <div className="flex items-center space-x-2">
                  <Code size={16} className="text-blue-400" />
                  <span className="text-[10px] font-mono text-gray-500 uppercase tracking-[0.2em] font-bold">Editor Codice DSL Aleph</span>

               </div>
               <span className="text-[10px] text-gray-600 font-mono">core.aleph</span>
            </div>
            <textarea 
               value={ontologyRaw}
               onChange={(e) => setOntologyRaw(e.target.value)}
               className="flex-1 w-full bg-transparent text-blue-50 font-mono text-sm outline-none resize-none leading-relaxed custom-scrollbar pt-2"
               spellCheck={false}
               placeholder="// Inizia a definire gli oggetti... es: object Appalto ..."
            />
          </div>

          <div className="bg-white rounded-3xl border border-gray-100 p-8 shadow-sm space-y-8 overflow-auto custom-scrollbar">
             <div>
                <h4 className="font-bold text-gray-900 mb-2">Visual Glossary</h4>
                <p className="text-xs text-gray-400 leading-relaxed">Struttura rilevata nel tuo modello.</p>
             </div>

             <div className="space-y-6">
                {ontologyRaw.split('object').filter(s => s.trim()).map((block, i) => {
                   const lines = block.trim().split('\n');
                   const name = lines[0].trim();
                   const props = lines.filter(l => l.includes('property')).map(l => l.split('property')[1].trim().split(' ')[0]);
                   return (
                     <div key={i} className="p-5 bg-blue-50/50 rounded-2xl border border-blue-100 animate-in fade-in duration-500">
                        <div className="flex items-center space-x-2 mb-3">
                           <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                           <span className="font-bold text-blue-900 text-sm">{name}</span>
                        </div>
                        <div className="space-y-1">
                           {props.map(p => (
                             <div key={p} className="text-[10px] text-blue-600 font-mono flex items-center space-x-1">
                                <span>•</span>
                                <span>{p}</span>
                             </div>
                           ))}
                        </div>
                     </div>
                   );
                })}
                
                <div className="p-4 bg-gray-50 rounded-2xl border border-dashed border-gray-200">
                   <h5 className="font-bold text-gray-400 text-[10px] uppercase tracking-widest mb-2">Tips</h5>
                   <p className="text-[10px] text-gray-400 leading-relaxed">
                      Usa <b>relation</b> per collegare due oggetti e abilitare la vista Grafo nell'Explorer.
                   </p>
                </div>
             </div>
          </div>
       </div>
    </div>
  );
};
