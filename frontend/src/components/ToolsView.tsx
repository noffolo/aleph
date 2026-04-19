import React from 'react';
import { Terminal, Plus, Code } from 'lucide-react';

interface Tool {
  id: string;
  name: string;
  description: string;
  code: string;
}

interface ToolsViewProps {
  tools: Tool[];
  onCreateTool: () => void;
}

export const ToolsView: React.FC<ToolsViewProps> = ({ tools, onCreateTool }) => {
  return (
    <div className="max-w-6xl mx-auto space-y-8">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Strumenti Operativi</h2>
          <p className="text-gray-500 text-sm mt-1">Definisci funzioni eseguibili dagli agenti (SQL, Python, API).</p>
        </div>
        <button onClick={onCreateTool} className="flex items-center space-x-2 bg-gray-900 text-white px-6 py-3 rounded-2xl font-bold hover:bg-black transition-all shadow-lg">
           <Plus size={20} />
           <span>Crea Strumento</span>
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {tools.map(t => (
          <div key={t.id} className="bg-white p-8 rounded-3xl border border-gray-100 shadow-sm hover:shadow-xl transition-all flex flex-col h-full">
             <div className="flex items-center space-x-3 mb-4">
                <div className="w-10 h-10 bg-gray-50 rounded-xl flex items-center justify-center text-gray-500"><Terminal size={20} /></div>
                <h3 className="text-xl font-bold">{t.name}</h3>
             </div>
             <p className="text-sm text-gray-500 mb-6 flex-1">{t.description}</p>
             <div className="bg-gray-50 p-4 rounded-2xl border border-gray-100 mb-6">
                <div className="flex items-center space-x-2 mb-2"><Code size={12} className="text-gray-400" /><span className="text-[10px] font-bold text-gray-400 uppercase tracking-widest">Code Preview</span></div>
                <pre className="text-[10px] font-mono text-gray-400 line-clamp-3 overflow-hidden">{t.code}</pre>
             </div>
             <button className="w-full py-3 bg-gray-50 text-gray-600 rounded-xl text-xs font-bold hover:bg-gray-100 transition-colors uppercase tracking-widest">Modifica Codice</button>
          </div>
        ))}
      </div>
    </div>
  );
};
