import React from 'react';
import { Zap, Plus } from 'lucide-react';

interface Skill {
  id: string;
  name: string;
  description: string;
}

interface SkillsViewProps {
  skills: Skill[];
  onCreateSkill: () => void;
}

export const SkillsView: React.FC<SkillsViewProps> = ({ skills, onCreateSkill }) => {
  return (
    <div className="max-w-6xl mx-auto space-y-8">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Skill Framework</h2>
          <p className="text-gray-500 text-sm mt-1">Pacchetti di capacità e prompt che trasformano gli agenti in specialisti.</p>
        </div>
        <button onClick={onCreateSkill} className="flex items-center space-x-2 bg-blue-600 text-white px-6 py-3 rounded-2xl font-bold hover:bg-blue-700 transition-all shadow-lg shadow-blue-200">
           <Plus size={20} />
           <span>Crea Skill</span>
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {skills.map(s => (
          <div key={s.id} className="bg-white p-6 rounded-3xl border border-gray-100 shadow-sm hover:shadow-xl transition-all">
             <div className="w-12 h-12 bg-amber-50 rounded-2xl flex items-center justify-center text-amber-600 mb-4"><Zap size={24} /></div>
             <h3 className="text-xl font-bold mb-1">{s.name}</h3>
             <p className="text-sm text-gray-500 leading-relaxed mb-6">{s.description}</p>
             <button className="w-full py-2 bg-amber-50 text-amber-700 rounded-xl text-[10px] font-bold uppercase tracking-widest hover:bg-amber-100 transition-colors">Dettagli Skill</button>
          </div>
        ))}
        <div className="col-span-full py-20 bg-blue-50/30 border-2 border-dashed border-blue-100 rounded-3xl text-center">
           <Zap size={48} className="mx-auto text-blue-200 mb-4" />
           <p className="text-blue-400 font-bold uppercase text-[10px] tracking-widest">Nessuna Skill personalizzata trovata</p>
           <button className="mt-4 text-sm text-blue-600 font-bold hover:underline">Importa Skill da JSON</button>
        </div>
      </div>
    </div>
  );
};
