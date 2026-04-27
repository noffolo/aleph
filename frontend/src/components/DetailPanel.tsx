import React from 'react';
import { X } from 'lucide-react';
import { t } from '../i18n';

interface DetailPanelProps {
  selectedRow: any;
  onClose: () => void;
}

export const DetailPanel: React.FC<DetailPanelProps> = ({ selectedRow, onClose }) => {
  if (!selectedRow) return null;

  return (
    <div className="absolute inset-y-0 right-0 w-[450px] bg-surface z-50 p-10 overflow-auto border-l border-border font-mono">
      <div className="flex justify-between items-center mb-10">
        <div>
          <h3 className="text-xl font-bold tracking-tight text-text">{t('detail.title')}</h3>
          <p className="text-[10px] font-mono text-textDim uppercase tracking-widest mt-1">Dettaglio Atomico del Record</p>
        </div>
        <button onClick={onClose} className="p-2 hover:bg-surfaceAlt text-textMuted hover:text-text transition-colors">
          <X size={20} />
        </button>
      </div>
      <div className="space-y-6">
        {selectedRow?.values ? Object.entries(selectedRow.values).map(([key, val]) => (
          <div key={key} className="group">
            <label className="block text-[10px] font-bold text-textDim uppercase tracking-[0.2em] mb-1 group-hover:text-primary transition-colors">
               {key}
            </label>
            <div className="text-text text-sm break-words bg-surfaceAlt p-3 border border-border leading-relaxed group-hover:border-primary/30 transition-all">
              {val != null && val !== '' ? String(val) : <span className="text-textDim">{t('detail.empty')}</span>}
            </div>
          </div>
        )) : <p className="text-textDim italic">{t('detail.noData')}</p>}
      </div>
    </div>
  );
};