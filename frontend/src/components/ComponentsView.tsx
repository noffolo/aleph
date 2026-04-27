import React, { useState } from 'react';
import { Cpu, Search, Zap, ToggleLeft, Plus, Eye } from 'lucide-react';
import { useStore } from '../store/useStore';
import { t } from '../i18n';
import { Timestamp } from '@bufbuild/protobuf';

interface ComponentMetadata {
  id: string;
  name: string;
  description: string;
  version: string;
  type: string;
  category: string;
  source: string;
  status: string;
  approvalStatus: string;
  configSchemaJson?: string;
  executionCommand?: string;
  dependenciesJson?: string;
  inputSchemaJson?: string;
  outputSchemaJson?: string;
  promptTemplate?: string;
  toolIdsJson?: string;
  avgLatencyMs?: number;
  avgBrierScore?: number;
  avgCpuUsage?: number;
  avgMemoryMb?: number;
  avgExecTimeMs?: number;
  trustScore?: number;
  createdByAgentId?: string;
  creationTimestamp?: Timestamp | string;
  lastUpdatedTimestamp?: Timestamp | string;
}

interface ComponentsViewProps {
  components: ComponentMetadata[];
  onUpdateComponentStatus: (id: string, status: string) => void;
  onRegisterComponent: (metadata: Partial<ComponentMetadata>) => void;
  onGetComponent: (id: string) => Promise<ComponentMetadata | null>;
  inline?: boolean;
}

export const ComponentsView: React.FC<ComponentsViewProps> = ({ components, onUpdateComponentStatus, onRegisterComponent, onGetComponent, inline = false }) => {
  const [filter, setFilter] = useState('');

  const filtered = components.filter(c =>
    c.name.toLowerCase().includes(filter.toLowerCase()) ||
    c.type.toLowerCase().includes(filter.toLowerCase()) ||
    c.category.toLowerCase().includes(filter.toLowerCase())
  );

  const statusColor = (status: string) => {
    switch (status) {
      case 'active': case 'running': return 'bg-success/10 text-success';
      case 'paused': case 'idle': return 'bg-warning/10 text-warning';
      case 'error': case 'failed': return 'bg-danger/10 text-danger';
      default: return 'bg-surface-alt text-textMuted';
    }
  };

  const openRegister = () => {
    useStore.getState().setSlideOverContent({ type: 'component-form', title: t('components.register'), data: undefined });
  };

  const openDetail = (id: string) => {
    const local = components.find(c => c.id === id);
    useStore.getState().setSlideOverContent({ type: 'component-detail', title: local?.name || t('components.edit'), data: { componentId: id } });
  };

  return (
    <div className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-8'}>
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">{t('components.title')}</h2>
          <p className="text-textMuted text-sm mt-1">{t('components.subtitle')}</p>
        </div>
        <button onClick={openRegister} className="flex items-center space-x-2 bg-primary text-white px-6 py-3 rounded-lg font-bold hover:bg-primary/90 transition-all shadow-lg ">
          <Plus size={20} />
          <span>{t('components.register')}</span>
        </button>
      </div>

      <div className="relative">
        <Search className="absolute left-4 top-4 text-textMuted" size={20} />
        <input
          type="text"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder={t('components.search')}
          className="w-full pl-12 pr-4 py-4 bg-surface border border-border rounded-lg focus:outline-none focus:ring-4 focus:ring-primary/10 transition-all text-lg shadow-sm"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filtered.map(c => (
          <div key={c.id} className="bg-surface p-6 rounded-lg border border-border shadow-sm hover:shadow-lg transition-all group">
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className="w-10 h-10 bg-primary/10 rounded-xl flex items-center justify-center text-primary"><Cpu size={20} /></div>
                <div>
                  <h3 className="font-bold text-base truncate max-w-[180px]">{c.name}</h3>
                  <div className="flex items-center space-x-2 mt-0.5">
                    <span className="text-[10px] font-mono bg-surface-alt text-textMuted px-2 py-0.5 rounded uppercase">{c.type}</span>
                    <span className="text-[10px] font-mono bg-primary/10 text-primary px-2 py-0.5 rounded uppercase">{c.category}</span>
                  </div>
                </div>
              </div>
              <span className={`text-[9px] font-bold uppercase px-2 py-1 rounded-lg ${statusColor(c.status)}`}>{c.status}</span>
            </div>

            <p className="text-sm text-textMuted mb-4 line-clamp-2 leading-relaxed">{c.description}</p>

            <div className="grid grid-cols-3 gap-2 mb-4 text-center">
              <div className="p-2 bg-surface-alt rounded-xl">
                <div className="text-[9px] text-textMuted uppercase font-bold">v{c.version}</div>
              </div>
              {c.avgLatencyMs != null && (
                <div className="p-2 bg-surface-alt rounded-xl">
                  <div className="text-[9px] text-textMuted uppercase font-bold">{c.avgLatencyMs.toFixed(0)}ms</div>
                </div>
              )}
              {c.avgExecTimeMs != null && (
                <div className="p-2 bg-surface-alt rounded-xl">
                   <div className="text-[9px] text-textMuted uppercase font-bold">Esecuzione {c.avgExecTimeMs.toFixed(0)}ms</div>
                </div>
              )}
              {c.trustScore != null && (
                 <div className="p-2 bg-surface-alt rounded-xl" title={t('components.betaScore')}>
                    <div className="text-[9px] text-textMuted uppercase font-bold">Trust (beta) {c.trustScore.toFixed(2)}</div>
                  </div>
               )}
              {c.avgBrierScore != null && (
                 <div className="p-2 bg-surface-alt rounded-xl" title={t('components.betaScore')}>
                    <div className="text-[9px] text-textMuted uppercase font-bold">Brier (beta) {c.avgBrierScore.toFixed(3)}</div>
                  </div>
               )}
              {c.avgCpuUsage != null && (
                <div className="p-2 bg-surface-alt rounded-xl">
                  <div className="text-[9px] text-textMuted uppercase font-bold">CPU {c.avgCpuUsage.toFixed(1)}%</div>
                </div>
              )}
              {c.avgMemoryMb != null && (
                <div className="p-2 bg-surface-alt rounded-xl">
                  <div className="text-[9px] text-textMuted uppercase font-bold">{c.avgMemoryMb.toFixed(0)}MB</div>
                </div>
              )}
            </div>

            {c.executionCommand && (
              <div className="mb-3 text-[10px] font-mono text-textMuted bg-surface-alt p-2 rounded-lg truncate" title={c.executionCommand}>$ {c.executionCommand}</div>
            )}

            {c.approvalStatus && c.approvalStatus !== 'approved' && (
              <div className="mb-3">
                <span className={`text-[9px] font-bold uppercase px-2 py-1 rounded-lg ${c.approvalStatus === 'pending' ? 'bg-warning/10 text-warning' : 'bg-danger/10 text-danger'}`}>{c.approvalStatus}</span>
              </div>
            )}

            <div className="flex items-center justify-between pt-4 border-t border-border">
              <span className="text-[10px] text-textMuted font-mono">{c.source}</span>
              <div className="flex items-center space-x-2">
                <button onClick={() => openDetail(c.id)} className="flex items-center space-x-1 text-[10px] font-bold text-primary bg-primary/10 px-3 py-1.5 rounded-lg hover:bg-primary/20 transition-colors">
                  <Eye size={12} /><span>Dettagli</span>
                </button>
                {c.status === 'active' ? (
                  <button onClick={() => onUpdateComponentStatus(c.id, 'paused')} className="flex items-center space-x-1 text-[10px] font-bold text-warning bg-warning/10 px-3 py-1.5 rounded-lg hover:bg-warning/10 transition-colors">
                    <ToggleLeft size={12} />
                    <span>Pausa</span>
                  </button>
                ) : (
                  <button onClick={() => onUpdateComponentStatus(c.id, 'active')} className="flex items-center space-x-1 text-[10px] font-bold text-success bg-success/10 px-3 py-1.5 rounded-lg hover:bg-success/10 transition-colors">
                    <Zap size={12} />
                    <span>Attiva</span>
                  </button>
                )}
              </div>
            </div>
          </div>
        ))}
        {filtered.length === 0 && (
          <div className="col-span-full py-20 bg-surface border-2 border-dashed border-border rounded-lg text-center">
            <Cpu size={48} className="mx-auto text-textDim mb-4" />
            <p className="text-textMuted font-bold uppercase text-xs tracking-widest">
              {filter ? 'Nessun componente corrisponde al filtro' : 'Nessun componente registrato nel catalogo'}
            </p>
          </div>
        )}
      </div>
    </div>
  );
};