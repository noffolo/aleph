import React from 'react';
import { Zap, Save, Code } from 'lucide-react';

interface OntologyViewProps {
  ontologyRaw: string;
  setOntologyRaw: (val: string) => void;
  onEmerge: () => void;
  onSave: () => void;
  inline?: boolean;
}

export const OntologyView: React.FC<OntologyViewProps> = ({ ontologyRaw, setOntologyRaw, onEmerge, onSave, inline = false }) => {
  return (
    <div className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-6 h-full flex flex-col'}>
       <div className={(inline ? '' : 'bg-surface p-6 rounded-lg border border-border shadow-sm ') + 'flex justify-between items-center'}>
          <div>
            <h2 className="text-3xl font-bold tracking-tight">Modellazione Business</h2>
            <p className="text-textMuted text-sm mt-1">L'Ontologia è il filtro intelligente tra i tuoi dati grezzi e l'AI.</p>
          </div>
          <div className="flex space-x-3">
            <button onClick={onEmerge} className="flex items-center space-x-2 bg-warning/10 text-warning px-6 py-3 rounded-lg font-bold hover:bg-warning/10 transition-all shadow-sm border border-warning/20">
              <Zap size={20} />
              <span>Emergenza Automatica</span>
            </button>
            <button onClick={onSave} className="flex items-center space-x-2 bg-primary text-background px-6 py-3 rounded-lg font-bold hover:bg-primary/90 transition-all shadow-lg ">
              <Save size={20} />
              <span>Pubblica Modello</span>
            </button>
          </div>
       </div>
       
       <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 flex-1 min-h-0">
           <div className="lg:col-span-2 bg-background rounded-lg p-8   border border-border flex flex-col min-h-[500px]">
             <div className="flex items-center justify-between mb-4 border-b border-border pb-4">
               <div className="flex items-center space-x-2">
                  <Code size={16} className="text-primary/70" />
                  <span className="text-[10px] font-mono text-textMuted uppercase tracking-[0.2em] font-bold">Editor Codice DSL Aleph</span>

               </div>
               <span className="text-[10px] text-textMuted font-mono">core.aleph</span>
            </div>
            <textarea 
               value={ontologyRaw}
               onChange={(e) => setOntologyRaw(e.target.value)}
               className="flex-1 w-full bg-transparent text-primary/80 font-mono text-sm outline-none resize-none leading-relaxed custom-scrollbar pt-2"
               spellCheck={false}
               placeholder="// Inizia a definire gli oggetti... es: object Appalto ..."
            />
          </div>

          <div className="bg-surface rounded-lg border border-border p-8 shadow-sm space-y-8 overflow-auto custom-scrollbar">
             <div>
                 <h4 className="font-bold text-text mb-2">Glossario Visivo</h4>
                <p className="text-xs text-textMuted leading-relaxed">Struttura rilevata nel tuo modello.</p>
             </div>

              <div className="space-y-6">
                  {(() => {
                    type Block = {name: string; type: string; props: string[]; relations: string[]; values: string[]};
                    const blocks: Block[] = [];
                    ontologyRaw.split('\n').forEach(line => {
                      const objMatch = line.match(/^object\s+(\w+)/);
                      const enumMatch = line.match(/^enum\s+(\w+)/);
                      if (objMatch) {
                        blocks.push({ name: objMatch[1], type: 'object', props: [], relations: [], values: [] });
                      } else if (enumMatch) {
                        blocks.push({ name: enumMatch[1], type: 'enum', props: [], relations: [], values: [] });
                      } else if (blocks.length > 0) {
                        const propMatch = line.match(/^\s*property\s+(\w+)/);
                        const relMatch = line.match(/^\s*relation\s+(\w+)/);
                        const valMatch = line.match(/^\s*value\s+(\w+)/);
                        if (propMatch) blocks[blocks.length - 1].props.push(propMatch[1]);
                        else if (relMatch) blocks[blocks.length - 1].relations.push(relMatch[1]);
                        else if (valMatch) blocks[blocks.length - 1].values.push(valMatch[1]);
                      }
                    });
                    return blocks.map((block, i) => (
                      <div key={i} className={`p-5 rounded-lg border animate-in fade-in duration-500 ${block.type === 'enum' ? 'bg-warning/10/50 border-warning/20' : 'bg-primary/10/50 border-primary/20'}`}>
                         <div className="flex items-center space-x-2 mb-3">
                            <div className={`w-2 h-2 rounded-full ${block.type === 'enum' ? 'bg-warning/100' : 'bg-primary/100'}`}></div>
                            <span className={`font-bold text-sm ${block.type === 'enum' ? 'text-text' : 'text-text'}`}>{block.name}</span>
                            <span className="text-[9px] font-mono bg-surface/60 px-1.5 py-0.5 rounded uppercase text-textMuted">{block.type}</span>
                         </div>
                         {block.props.length > 0 && (
                           <div className="space-y-1 mb-2">
                           {block.props.map(p => (
                             <div key={p} className="text-[10px] text-primary font-mono flex items-center space-x-1"><span>•</span><span>{p}</span></div>
                           ))}
                         </div>
                       )}
                         {block.relations.length > 0 && (
                           <div className="space-y-1 mb-2">
                             {block.relations.map(r => (
                               <div key={r} className="text-[10px] text-primary font-mono flex items-center space-x-1"><span>→</span><span>{r}</span></div>
                             ))}
                           </div>
                         )}
                         {block.values.length > 0 && (
                           <div className="space-y-1">
                             {block.values.map(v => (
                               <div key={v} className="text-[10px] text-warning font-mono flex items-center space-x-1"><span>◇</span><span>{v}</span></div>
                             ))}
                           </div>
                         )}
                      </div>
                    ));
                  })()}
                
                <div className="p-4 bg-surface-alt rounded-lg border border-dashed border-border">
                   <h5 className="font-bold text-textMuted text-[10px] uppercase tracking-widest mb-2">Tips</h5>
                   <p className="text-[10px] text-textMuted leading-relaxed">
                      Usa <b>relation</b> per collegare due oggetti e abilitare la vista Grafo nell'Explorer.
                   </p>
                </div>
             </div>
          </div>
       </div>
    </div>
  );
};
