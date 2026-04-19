import React from 'react';
import { BarChart3, Hash, ListFilter, Activity } from 'lucide-react';

interface ColumnStats {
  columnName: string;
  min: string;
  max: string;
  count: number;
  uniqueCount: number;
  topValues: { [key: string]: number };
}

interface DataHealthViewProps {
  stats: ColumnStats[];
}

export const DataHealthView: React.FC<DataHealthViewProps> = ({ stats }) => {
  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {stats.map(s => (
          <div key={s.columnName} className="bg-white p-8 rounded-[32px] border border-gray-100 shadow-sm hover:shadow-xl transition-all">
            <div className="flex items-center justify-between mb-6">
               <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-blue-50 rounded-xl flex items-center justify-center text-blue-600"><Hash size={20} /></div>
                  <h3 className="font-bold text-gray-900 truncate max-w-[150px]">{s.columnName}</h3>
               </div>
               <span className="text-[10px] font-bold text-gray-400 uppercase tracking-widest bg-gray-50 px-2 py-1 rounded-md">Profilo Dati</span>
            </div>
            
            <div className="grid grid-cols-2 gap-4 mb-8">
               <div className="p-4 bg-gray-50/50 rounded-2xl border border-gray-100">
                  <div className="text-[9px] font-bold text-gray-400 uppercase mb-1">Unici</div>
                  <div className="text-xl font-bold text-gray-900">{s.uniqueCount}</div>
               </div>
               <div className="p-4 bg-gray-50/50 rounded-2xl border border-gray-100">
                  <div className="text-[9px] font-bold text-gray-400 uppercase mb-1">Record</div>
                  <div className="text-xl font-bold text-gray-900">{s.count}</div>
               </div>
            </div>

            <div className="space-y-3">
               <div className="text-[10px] font-bold text-gray-400 uppercase tracking-widest flex items-center space-x-2">
                  <BarChart3 size={12} />
                  <span>Distribuzione Top 5</span>
               </div>
               {Object.entries(s.topValues).sort((a,b) => b[1] - a[1]).map(([val, count]) => (
                  <div key={val} className="space-y-1">
                     <div className="flex justify-between text-[11px] font-medium text-gray-600">
                        <span className="truncate pr-4 italic">{val === "null" || val === "" ? "[Vuoto]" : val}</span>
                        <span className="font-bold text-gray-400">{count}</span>
                     </div>
                     <div className="h-1.5 w-full bg-gray-50 rounded-full overflow-hidden">
                        <div className="h-full bg-blue-500 rounded-full" style={{ width: `${(count / s.count) * 100}%` }}></div>
                     </div>
                  </div>
               ))}
            </div>

            <div className="mt-8 pt-6 border-t border-gray-50 flex justify-between text-[10px] font-mono text-gray-400">
               <div className="truncate pr-2" title={s.min}>MIN: {s.min}</div>
               <div className="truncate pl-2 border-l border-gray-100" title={s.max}>MAX: {s.max}</div>
            </div>
          </div>
        ))}
      </div>
      
      {stats.length === 0 && (
         <div className="py-40 text-center">
            <Activity size={48} className="mx-auto text-gray-100 mb-4" />
            <p className="text-gray-400 font-bold uppercase text-xs tracking-widest">Seleziona un oggetto per analizzare la salute dei dati</p>
         </div>
      )}
    </div>
  );
};
