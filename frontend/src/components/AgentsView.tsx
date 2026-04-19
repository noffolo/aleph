import React from 'react';
import { Bot, Plus, Trash2 } from 'lucide-react';

interface Agent {
  id: string;
  name: string;
  model: string;
  systemPrompt: string;
}

interface AgentsViewProps {
  agents: Agent[];
  onCreateAgent: () => void;
  onDeleteAgent: (id: string) => void;
}

export const AgentsView: React.FC<AgentsViewProps> = ({ agents, onCreateAgent, onDeleteAgent }) => {
  return (
    <div className="max-w-6xl mx-auto space-y-8">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Gestore Agenti</h2>
          <p className="text-gray-500 text-sm mt-1">Configura entità AI specializzate connesse ai tuoi dati locali (Ollama).</p>
        </div>
        <button onClick={onCreateAgent} className="flex items-center space-x-2 bg-blue-600 text-white px-6 py-3 rounded-2xl font-bold hover:bg-blue-700 transition-all shadow-lg shadow-blue-200">
           <Plus size={20} />
           <span>Nuovo Agente</span>
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {agents.map(a => (
          <div key={a.id} className="bg-white p-6 rounded-3xl border border-gray-100 shadow-sm hover:shadow-xl transition-all group relative">
             <button 
                onClick={(e) => { e.stopPropagation(); if (confirm('Sei sicuro di voler eliminare questo agente?')) onDeleteAgent(a.id); }}
                className="absolute top-6 right-6 p-2 text-gray-300 hover:text-red-500 hover:bg-red-50 rounded-xl transition-all"
             >
                <Trash2 size={18} />
             </button>
             <div className="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-600 mb-4 group-hover:bg-blue-600 group-hover:text-white transition-colors">
                <Bot size={24} />
             </div>
             <h3 className="text-xl font-bold mb-1">{a.name}</h3>
             <div className="inline-block px-2 py-1 bg-gray-100 rounded-md text-[10px] font-mono font-bold text-gray-500 mb-4 uppercase">{a.model}</div>
             <p className="text-sm text-gray-500 line-clamp-4 mb-6 leading-relaxed">
                {a.systemPrompt || "Nessun prompt di sistema configurato."}
             </p>
             <div className="flex items-center space-x-2 border-t pt-4 border-gray-50">
                <div className="h-2 w-2 bg-green-500 rounded-full"></div>
                <span className="text-[10px] font-bold text-gray-400 uppercase tracking-widest">Servizio Attivo</span>
             </div>
          </div>
        ))}
        {agents.length === 0 && (
          <div className="col-span-full py-20 bg-gray-50 border-2 border-dashed border-gray-200 rounded-3xl text-center">
             <Bot size={48} className="mx-auto text-gray-300 mb-4" />
             <p className="text-gray-400 font-medium font-mono uppercase text-xs tracking-widest">Nessun agente configurato per questo workspace</p>
          </div>
        )}
      </div>
    </div>
  );
};
