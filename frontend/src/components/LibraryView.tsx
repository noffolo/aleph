import React from 'react';
import { Book, FileText, Download, Eye, Calendar, Trash2 } from 'lucide-react';

interface Asset {
  id: string;
  name: string;
  type: string;
  createdAt: number;
}

interface LibraryViewProps {
  assets: Asset[];
  onViewAsset: (id: string) => void;
  onDeleteAsset: (id: string) => void;
  selectedAssetContent: string | null;
  setSelectedAssetContent: (val: string | null) => void;
}

export const LibraryView: React.FC<LibraryViewProps> = ({ assets, onViewAsset, onDeleteAsset, selectedAssetContent, setSelectedAssetContent }) => {
  return (
    <div className="max-w-6xl mx-auto space-y-8">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Biblioteca Asset</h2>
          <p className="text-gray-500 text-sm mt-1">Archivio centralizzato di report, analisi e snapshot generati dagli agenti.</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {assets.map(a => (
          <div key={a.id} className="bg-white p-6 rounded-3xl border border-gray-100 shadow-sm hover:shadow-xl transition-all group relative">
             <button 
                onClick={(e) => { e.stopPropagation(); if (confirm('Sei sicuro di voler eliminare questo asset?')) onDeleteAsset(a.id); }}
                className="absolute top-6 right-6 p-2 text-gray-300 hover:text-red-500 hover:bg-red-50 rounded-xl transition-all opacity-0 group-hover:opacity-100"
             >
                <Trash2 size={18} />
             </button>
             <div className="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-600 mb-4 group-hover:bg-blue-600 group-hover:text-white transition-colors">
                <FileText size={24} />
             </div>
             <h3 className="text-xl font-bold mb-1 truncate">{a.name}</h3>
             <div className="flex items-center space-x-2 text-[10px] text-gray-400 font-bold uppercase tracking-widest mb-6">
                <Calendar size={12} />
                <span>{new Date(a.createdAt * 1000).toLocaleDateString()}</span>
                <span className="bg-gray-100 px-2 py-0.5 rounded text-gray-500">{a.type}</span>
             </div>
             <div className="flex items-center space-x-2">
                <button 
                   onClick={() => onViewAsset(a.id)}
                   className="flex-1 py-3 bg-gray-900 text-white rounded-xl text-xs font-bold hover:bg-black transition-colors flex items-center justify-center space-x-2"
                >
                   <Eye size={14} />
                   <span>Leggi Report</span>
                </button>
                <button className="p-3 bg-gray-50 text-gray-400 rounded-xl hover:bg-gray-100 hover:text-gray-900 transition-all">
                   <Download size={16} />
                </button>
             </div>
          </div>
        ))}
        {assets.length === 0 && (
          <div className="col-span-full py-24 bg-white border-2 border-dashed border-gray-100 rounded-[40px] text-center">
             <Book size={48} className="mx-auto text-gray-200 mb-4" />
             <p className="text-gray-400 font-bold uppercase text-[10px] tracking-[0.2em]">Nessun report generato in questo workspace</p>
          </div>
        )}
      </div>

      {selectedAssetContent && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-[100] flex items-center justify-center p-8 animate-in fade-in duration-300">
           <div className="bg-white w-full max-w-4xl max-h-full overflow-hidden rounded-[40px] shadow-2xl flex flex-col animate-in zoom-in-95 duration-300">
              <div className="p-8 border-b flex justify-between items-center bg-gray-50/50">
                 <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-blue-600 rounded-xl flex items-center justify-center text-white"><FileText size={20} /></div>
                    <h3 className="text-2xl font-bold tracking-tight text-gray-900">Visualizzatore Asset</h3>
                 </div>
                 <button 
                    onClick={() => setSelectedAssetContent(null)}
                    className="p-3 hover:bg-white rounded-2xl text-gray-400 hover:text-gray-900 transition-all border border-transparent hover:border-gray-200"
                 >
                    <X size={24} />
                 </button>
              </div>
              <div className="flex-1 overflow-auto p-12 bg-white">
                 <article className="prose prose-blue max-w-none">
                    <pre className="whitespace-pre-wrap font-sans text-lg leading-relaxed text-gray-800">
                       {selectedAssetContent}
                    </pre>
                 </article>
              </div>
              <div className="p-6 border-t bg-gray-50/50 flex justify-end space-x-4">
                 <button className="px-8 py-3 bg-blue-600 text-white rounded-2xl font-bold hover:bg-blue-700 transition-all shadow-lg shadow-blue-200">Esporta in PDF</button>
              </div>
           </div>
        </div>
      )}
    </div>
  );
};

import { X } from 'lucide-react';
