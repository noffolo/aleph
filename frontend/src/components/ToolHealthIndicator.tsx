import React from 'react';

interface ToolHealthIndicatorProps {
  status: 'healthy' | 'warning' | 'error' | 'unknown';
  lastCheck?: string;
}

export const ToolHealthIndicator: React.FC<ToolHealthIndicatorProps> = ({ status, lastCheck }) => {
  const colors = {
    healthy: 'bg-success',
    warning: 'bg-warning',
    error: 'bg-danger',
    unknown: 'bg-textDim',
  };

  return (
    <div className="group relative flex items-center justify-center w-3 h-3 rounded-full shadow-sm transition-all" 
         title={lastCheck ? `Last checked: ${lastCheck}` : 'Unknown status'}>
      <div className={`w-full h-full rounded-full ${colors[status]} ${status === 'healthy' ? 'animate-pulse' : ''}`} />
      
      <div className="absolute bottom-full mb-2 hidden group-hover:block z-10">
        <div className="bg-surface border border-border text-textPrimary text-[10px] py-1 px-2 rounded shadow-lg whitespace-nowrap">
          {status === 'healthy' && 'Sistemi Operativi'}
          {status === 'warning' && 'Latenza Rilevata'}
          {status === 'error' && 'Errore di Esecuzione'}
          {status === 'unknown' && 'Stato Non Verificato'}
          {lastCheck && ` • ${lastCheck}`}
        </div>
      </div>
    </div>
  );
};
