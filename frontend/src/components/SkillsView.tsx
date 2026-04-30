import React, { useCallback } from 'react';
import { Zap, Plus, Trash2, Play } from 'lucide-react';
import { useStore } from '../store/useStore';
import { useCursorPagination } from '../hooks/useCursorPagination';
import { skillClient } from '../api/factory';
import { ListSkillsRequest } from '../api/proto/aleph/v1/query_pb';
import { useQueryState } from 'nuqs';
import { t } from '../i18n';

interface Skill {
  id: string;
  name: string;
  description: string;
  toolIds?: string[];
}

interface Tool {
  id: string;
  name: string;
}

export interface SkillsViewProps {
  skills: Skill[];
  tools: Tool[];
  onCreateSkill: (name: string, description: string, toolIds: string[]) => void;
  onViewSkillDetail: (skill: Skill) => void;
  onDeleteSkill: (id: string) => void;
  onRunSkill: (id: string) => void;
  inline?: boolean;
}

export const SkillsView: React.FC<SkillsViewProps> = React.memo(({ skills: initialSkills, tools, onCreateSkill, onViewSkillDetail, onDeleteSkill, onRunSkill, inline = false }) => {
  const setSkills = useStore(state => state.setSkills);
  const projectId = useStore(state => state.selectedObject);
  const [searchQuery, setSearchQuery] = useQueryState('q', { defaultValue: '' });

  const { items: skills, hasMore, loadMore, loading } = useCursorPagination({
    clientMethod: skillClient.listSkills,
    requestBuilder: useCallback((cursor: string) => new ListSkillsRequest({ projectId, after: cursor, limit: 25 }), [projectId]),
    responseExtractor: (res) => ({ items: (res.skills || []) as any[], nextCursor: res.nextCursor }),
    storeSetter: setSkills as (items: any[]) => void,
    initialItems: initialSkills as any[],
  });

  const filteredSkills = skills.filter(s => 
    !searchQuery || 
    s.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
    s.description.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const openCreate = useCallback(() => {
    useStore.getState().setSlideOverContent({ type: 'skill-form', title: t('skills.create'), data: { tools } });
  }, [tools]);

  return (
    <div 
      className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-8'} 
      role="region" 
      aria-label="Skills"
    >
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">{t('skills.title')}</h2>
          <p className="text-textMuted text-sm mt-1">{t('skills.subtitle')}</p>
        </div>
          <button 
            onClick={openCreate} 
            className="flex items-center space-x-2 bg-primary text-background px-6 py-3 rounded-lg font-bold hover:bg-primary/90 transition-all shadow-lg focus:ring-2 focus:ring-primary"
            aria-label="Create new skill"
          >
            <Plus size={20} />
            <span>{t('skills.create')}</span>
        </button>
       </div>

       <input
         type="text"
         value={searchQuery}
         onChange={e => setSearchQuery(e.target.value)}
         placeholder={t('skills.search')}
         className="w-full max-w-md px-4 py-2 bg-surface-alt border border-border rounded-lg text-sm font-mono text-textPrimary placeholder-textDim focus:outline-none focus:border-primary/50 focus:ring-2 focus:ring-primary"
       />

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
         {filteredSkills.map(s => (
           <div key={s.id} className="bg-surface p-6 rounded-lg border border-border shadow-sm hover:shadow-lg transition-all group relative">

               <button 
                   onClick={(e) => { e.stopPropagation(); if (confirm('Eliminare questa skill?')) onDeleteSkill(s.id); }}
                   className="absolute top-6 right-6 p-2 text-textDim hover:text-danger hover:bg-danger/10 rounded-xl transition-all opacity-0 group-hover:opacity-100 focus:ring-2 focus:ring-primary"
                   aria-label={`Delete skill ${s.name}`}
                >
                 <Trash2 size={16} />
              </button>
              <div className="w-12 h-12 bg-warning/10 rounded-lg flex items-center justify-center text-warning mb-4"><Zap size={24} /></div>
              <h3 className="text-xl font-bold mb-1">{s.name}</h3>
              <p className="text-sm text-textMuted leading-relaxed mb-4">{s.description}</p>
              {s.toolIds && s.toolIds.length > 0 && (
                <div className="flex flex-wrap gap-1 mb-4">
                    {s.toolIds.map((tid: string) => {
                      const tool = tools.find(t => t.id === tid);
                      return <span key={tid} className="text-[9px] bg-primary/10 text-primary px-2 py-0.5 rounded font-mono">{tool?.name || tid}</span>;
                    })}
                </div>
              )}
              <div className="flex space-x-2">
                  <button 
                    onClick={() => onRunSkill(s.id)} 
                    className="flex-1 py-2 bg-primary text-background rounded-xl text-[10px] font-bold uppercase tracking-widest hover:bg-primary/90 transition-colors flex items-center justify-center space-x-1 focus:ring-2 focus:ring-primary"
                    aria-label={`Execute skill ${s.name}`}
                  >
                  <Play size={12} />
                  <span>Esegui</span>
                </button>
                 <button 
                   onClick={() => onViewSkillDetail(s)} 
              className="flex-1 py-2 bg-warning/10 text-warning rounded-xl text-[10px] font-bold uppercase tracking-widest hover:bg-warning/10 transition-colors focus:ring-2 focus:ring-primary"
                    aria-label={`View details for ${s.name}`}
                 >Dettagli</button>
              </div>
           </div>
        ))}
        {skills.length === 0 && (
          <div className="col-span-full py-20 bg-primary/10/30 border-2 border-dashed border-primary/20 rounded-lg text-center">
             <Zap size={48} className="mx-auto text-primary/30 mb-4" />
              <p className="text-primary/70 font-bold uppercase text-[10px] tracking-widest">Nessuna Skill personalizzata trovata</p>
          </div>
        )}
      </div>

      {hasMore && (
        <div className="flex justify-center">
          <button 
            onClick={loadMore} 
            disabled={loading}
             className="rounded-lg border border-border px-4 py-2 text-sm text-textMuted hover:text-textPrimary hover:border-textMuted transition-colors disabled:opacity-50 focus:ring-2 focus:ring-primary"
          >
            {loading ? 'Caricamento...' : 'Carica Altri'}
          </button>
        </div>
      )}
    </div>
  );
});
