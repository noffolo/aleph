import React from 'react';
import { LayoutGrid, Binary, Book, Settings, Zap, Bot, Terminal, ChevronRight } from 'lucide-react';

interface SidebarItemProps {
  icon: any;
  label: string;
  active?: boolean;
  onClick: () => void;
  badge?: string;
}

const SidebarItem: React.FC<SidebarItemProps> = ({ icon: Icon, label, active = false, onClick, badge }) => (
  <div 
    onClick={onClick} 
    className={`group flex items-center justify-between p-3.5 cursor-pointer rounded-2xl transition-all duration-300 ${active ? 'bg-blue-600 text-white shadow-lg shadow-blue-200' : 'hover:bg-gray-100 text-gray-500'}`}
  >
    <div className="flex items-center space-x-3">
       <Icon size={20} className={active ? 'text-white' : 'group-hover:text-blue-600 transition-colors'} />
       <span className="font-bold text-sm tracking-tight">{label}</span>
    </div>
    {badge && <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold ${active ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-400'}`}>{badge}</span>}
    {active && <ChevronRight size={14} className="text-blue-300" />}
  </div>
);

interface SidebarProps {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  projectID: string;
  onShowOnboarding: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ activeTab, setActiveTab, projectID, onShowOnboarding }) => {
  return (
    <div className="w-72 border-r bg-white p-6 flex flex-col h-full">
      <div className="flex items-center space-x-3 mb-12 px-2">
         <div className="w-10 h-10 bg-gradient-to-tr from-blue-700 to-blue-500 rounded-2xl flex items-center justify-center text-white shadow-xl shadow-blue-100">
            <Binary size={24} />
         </div>
         <div className="flex flex-col">
            <span className="text-xl font-black tracking-tighter leading-none text-blue-900 uppercase">Aleph</span>
            <span className="text-[10px] font-bold text-blue-400 uppercase tracking-widest mt-1">Data OS • v2.0</span>
         </div>
      </div>

      <nav className="flex-1 space-y-2">
         <div className="text-[10px] font-black text-gray-300 uppercase tracking-[0.2em] px-4 mb-4">Core Systems</div>
         <SidebarItem icon={LayoutGrid} label="Explorer" active={activeTab === 'Explorer'} onClick={() => setActiveTab('Explorer')} />
         <SidebarItem icon={Bot} label="Copilot" active={activeTab === 'Copilot'} onClick={() => setActiveTab('Copilot')} />
         <SidebarItem icon={Zap} label="Oracle" active={activeTab === 'Oracle'} onClick={() => setActiveTab('Oracle')} badge="AI" />
         <SidebarItem icon={Book} label="Library" active={activeTab === 'Library'} onClick={() => setActiveTab('Library')} />

         <div className="pt-8 text-[10px] font-black text-gray-300 uppercase tracking-[0.2em] px-4 mb-4">Modeling & Governance</div>
         <SidebarItem icon={Zap} label="Model Designer" active={activeTab === 'Ontologies'} onClick={() => setActiveTab('Ontologies')} />
         <SidebarItem icon={Terminal} label="Source Connectors" active={activeTab === 'Data Sources'} onClick={() => setActiveTab('Data Sources')} />
         <SidebarItem icon={Settings} label="Governance" active={activeTab === 'Agents'} onClick={() => setActiveTab('Agents')} />
      </nav>
      
      <div className="mt-auto pt-6 border-t space-y-6">
         <div>
            <div className="px-4 mb-3 text-[10px] font-bold text-gray-400 uppercase tracking-widest flex items-center justify-between">
               <span>Team Presence</span>
               <div className="flex space-x-1">
                  <div className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse"></div>
               </div>
            </div>
            <div className="flex -space-x-3 px-4">
               {['IO', 'JD', 'AM'].map((u, i) => (
                 <div key={i} className="w-9 h-9 rounded-full border-4 border-white bg-gray-100 flex items-center justify-center text-[10px] font-black text-gray-400 shadow-sm hover:z-10 transition-all cursor-help" title={`Membro: ${u}`}>{u}</div>
               ))}
               <div className="w-9 h-9 rounded-full border-4 border-white bg-blue-50 flex items-center justify-center text-[10px] font-black text-blue-600 shadow-sm">+1</div>
            </div>
         </div>

         <button 
           onClick={onShowOnboarding}
           className="w-full text-left bg-gray-50 p-4 rounded-[24px] hover:bg-blue-50 transition-all group border border-transparent hover:border-blue-100"
         >
            <div className="text-[10px] font-bold text-gray-400 group-hover:text-blue-400 uppercase tracking-widest mb-1">Current Space</div>
            <div className="flex items-center justify-between">
               <span className="font-black text-sm text-gray-900 truncate pr-2 uppercase italic tracking-tighter">{projectID || 'None'}</span>
               <Settings size={14} className="text-gray-300 group-hover:text-blue-500" />
            </div>
         </button>
      </div>
    </div>
  );
};
