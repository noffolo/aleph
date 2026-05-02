import React, { useEffect, useState } from 'react';

interface DataPanelProps {
  data: any;
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children?: React.ReactNode;
}

export const DataPanel: React.FC<DataPanelProps> = ({ data, isOpen, onClose, title, children }) => {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (isOpen) {
      requestAnimationFrame(() => setVisible(true));
    } else {
      setVisible(false);
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div className={`flex flex-col w-80 lg:w-96 h-full border-l border-border bg-surface shrink-0 transition-transform duration-200 ${visible ? 'translate-x-0' : 'translate-x-full'}`}>
      <div className="h-9 flex items-center justify-between px-4 border-b border-border shrink-0">
        <span className="text-xs font-bold text-textMuted uppercase tracking-widest">{title || 'DETAIL'}</span>
        <button onClick={onClose} className="text-textMuted hover:text-text text-xs transition-colors font-mono" aria-label="Close data panel">ESC</button>
      </div>
      <div className="flex-1 overflow-auto p-4 font-mono text-sm custom-scrollbar">
        {data ? (
          <div className="space-y-3">
            {Object.entries(data).map(([key, value]) => (
              <div key={key} className="flex flex-col gap-0.5">
                <span className="text-[10px] text-textDim uppercase">{key}</span>
                <span className="text-text break-all">{String(value ?? '—')}</span>
              </div>
            ))}
          </div>
        ) : (
          <span className="text-textDim">No data selected</span>
        )}
        {children}
      </div>
    </div>
  );
};