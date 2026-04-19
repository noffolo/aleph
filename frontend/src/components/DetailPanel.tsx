import React from 'react';
import { X } from 'lucide-react';

interface DetailPanelProps {
  selectedRow: any;
  onClose: () => void;
}

export const DetailPanel: React.FC<DetailPanelProps> = ({ selectedRow, onClose }) => {
  if (!selectedRow) return null;

  return (
    <div className="absolute inset-y-0 right-0 w-[450px] bg-white shadow-[-30px_0_60px_rgba(0,0,0,0.1)] z-50 p-10 overflow-auto border-l border-gray-100 animate-in slide-in-from-right-8 duration-300">
      <div className="flex justify-between items-center mb-10">
        <div>
          <h3 className="text-3xl font-bold tracking-tight">Ispezione Dati</h3>
          <p className="text-[10px] font-mono text-gray-400 uppercase tracking-widest mt-1">Dettaglio Atomico del Record</p>
        </div>
        <button onClick={onClose} className="p-3 hover:bg-gray-50 rounded-2xl text-gray-400 hover:text-gray-900 transition-all border border-transparent hover:border-gray-100">
          <X size={28} />
        </button>
      </div>
      <div className="space-y-8">
        {Object.entries(selectedRow.values).map(([key, val]) => (
          <div key={key} className="group">
            <label className="block text-[10px] font-bold text-gray-400 uppercase tracking-[0.2em] mb-2 group-hover:text-blue-500 transition-colors">
               {key}
            </label>
            <div className="text-gray-900 font-medium break-words bg-gray-50/50 p-5 rounded-2xl border border-gray-100 leading-relaxed shadow-sm group-hover:shadow-md transition-all group-hover:bg-white">
              {val as string || <span className="italic text-gray-300">Null/Empty</span>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
