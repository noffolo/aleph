import React from 'react';
import { BarChart3, Hash, Activity } from 'lucide-react';

interface ColumnStats {
  columnName: string;
  min: string;
  max: string;
  count: bigint | number;
  uniqueCount: bigint | number;
  topValues: { [key: string]: bigint | number };
}

interface DataHealthViewProps {
  stats: ColumnStats[];
  inline?: boolean;
}

const toNum = (v: bigint | number): number => typeof v === 'bigint' ? Number(v) : v;

export const DataHealthView: React.FC<DataHealthViewProps> = React.memo(({ stats, inline = false }) => {
  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {stats.map(s => {
          const countNum = toNum(s.count);
          return (
            <div key={s.columnName} className="bg-surface p-8 rounded-lg border border-border shadow-sm hover:shadow-lg shadow-primary/5 transition-all">
              <div className="flex items-center justify-between mb-6">
                 <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-primary/10 rounded-xl flex items-center justify-center text-primary"><Hash size={20} /></div>
                    <h3 className="font-bold text-text truncate max-w-[150px]">{s.columnName}</h3>
                 </div>
                 <span className="text-[10px] font-bold text-textMuted uppercase tracking-widest bg-surface-alt px-2 py-1 rounded-md">Profilo Dati</span>
              </div>
              
              <div className="grid grid-cols-2 gap-4 mb-8">
                 <div className="p-4 bg-surface-alt rounded-2xl border border-border">
                    <div className="text-[9px] font-bold text-textMuted uppercase mb-1">Unici</div>
                    <div className="text-xl font-bold text-text">{toNum(s.uniqueCount).toLocaleString('it-IT')}</div>
                 </div>
                 <div className="p-4 bg-surface-alt rounded-2xl border border-border">
                    <div className="text-[9px] font-bold text-textMuted uppercase mb-1">Record</div>
                    <div className="text-xl font-bold text-text">{countNum.toLocaleString('it-IT')}</div>
                 </div>
              </div>

              <div className="space-y-3">
                 <div className="text-[10px] font-bold text-textMuted uppercase tracking-widest flex items-center space-x-2">
                    <BarChart3 size={12} />
                    <span>Distribuzione Top 5</span>
                 </div>
                 {Object.entries(s.topValues).sort((a,b) => toNum(b[1]) - toNum(a[1])).slice(0, 5).map(([val, count]) => {
                    const countVal = toNum(count);
                    return (
                     <div key={val} className="space-y-1">
                        <div className="flex justify-between text-[11px] font-medium text-textMuted">
                           <span className="truncate pr-4 italic">{val === "null" || val === "" ? "[Vuoto]" : val}</span>
                           <span className="font-bold text-textDim">{countVal.toLocaleString('it-IT')}</span>
                        </div>
                        <div className="h-1.5 w-full bg-surface-alt rounded-full overflow-hidden">
                           <div className="h-full bg-primary rounded-full" style={{ width: `${countNum > 0 ? (countVal / countNum) * 100 : 0}%` }}></div>
                        </div>
                     </div>
                    );
                 })}
              </div>

              <div className="mt-8 pt-6 border-t border-border flex justify-between text-[10px] font-mono text-textMuted">
                 <div className="truncate pr-2" title={s.min}>MIN: {s.min}</div>
                 <div className="truncate pl-2 border-l border-border" title={s.max}>MAX: {s.max}</div>
              </div>
            </div>
          );
        })}
      </div>
      
      {stats.length === 0 && (
         <div className="py-40 text-center">
         <Activity size={48} className="mx-auto text-textDim mb-4" />
             <p className="text-textMuted font-bold uppercase text-xs tracking-widest">Seleziona un oggetto per analizzare la salute dei dati</p>
         </div>
      )}
    </div>
  );
});
